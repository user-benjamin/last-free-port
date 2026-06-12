package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

// Server accepts WebSocket connections and attaches them to the Hub.
type Server struct {
	hub *Hub
}

func NewServer() *Server {
	return &Server{hub: NewHub()}
}

// HandleWS runs one player's connection: handshake, then a read pump
// (intents in) on this goroutine and a write pump (snapshots out) on a
// second one. The hub itself is never touched directly — channels only.
func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Dev only: the Godot desktop client sends no Origin header and a
		// browser export runs on a different origin than the server.
		// Lock this down when auth lands (proposal §16).
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		slog.Warn("websocket accept failed", "error", err)
		return
	}
	defer conn.CloseNow()

	p, ok := s.handshake(r.Context(), conn)
	if !ok {
		return
	}

	// Write pump: drains the player's frame channel until the connection
	// context dies. Frames are pre-marshaled by the hub.
	go func() {
		for {
			select {
			case frame := <-p.send:
				if err := conn.Write(r.Context(), websocket.MessageText, frame); err != nil {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	}()

	s.hub.joins <- p
	defer func() { s.hub.leaves <- p.id }()

	// Read pump: everything the player wants is an intent; the hub decides.
	for {
		var env protocol.Envelope
		if err := wsjson.Read(r.Context(), conn, &env); err != nil {
			slog.Info("player disconnected", "player_id", p.id)
			return
		}
		switch env.Type {
		case protocol.TypeMoveIntent:
			var intent protocol.MoveIntent
			if err := json.Unmarshal(env.Data, &intent); err != nil {
				continue // garbage in, nothing out
			}
			s.hub.moves <- moveReq{id: p.id, dx: intent.DX, dy: intent.DY}
		default:
			// Unknown types are logged and ignored so old clients don't
			// break the session when the protocol grows.
			slog.Info("unhandled message", "player_id", p.id, "type", env.Type)
		}
	}
}

// handshake expects hello as the first message and replies with welcome.
// Real session tickets (validated against the API) replace this when auth
// lands (proposal §16).
func (s *Server) handshake(ctx context.Context, conn *websocket.Conn) (*player, bool) {
	hsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var env protocol.Envelope
	if err := wsjson.Read(hsCtx, conn, &env); err != nil {
		slog.Warn("read hello failed", "error", err)
		return nil, false
	}
	if env.Type != protocol.TypeHello {
		writeMsg(hsCtx, conn, protocol.TypeError, protocol.Error{
			Code: "expected_hello", Message: "first message must be hello",
		})
		return nil, false
	}
	var hello protocol.Hello
	if err := json.Unmarshal(env.Data, &hello); err != nil {
		writeMsg(hsCtx, conn, protocol.TypeError, protocol.Error{
			Code: "bad_hello", Message: "hello data was not valid JSON",
		})
		return nil, false
	}

	p := &player{
		id:   newPlayerID(),
		name: hello.Name,
		x:    randomCoord(WorldW),
		y:    randomCoord(WorldH),
		send: make(chan []byte, 8),
	}
	slog.Info("player connected", "player_id", p.id, "name", p.name)

	err := writeMsg(hsCtx, conn, protocol.TypeWelcome, protocol.Welcome{
		PlayerID:   p.id,
		ServerTime: time.Now().UTC().Format(time.RFC3339),
		Motd:       "No flag, no fortune. Welcome to the cove.",
		SpawnX:     p.x,
		SpawnY:     p.y,
		WorldW:     WorldW,
		WorldH:     WorldH,
	})
	if err != nil {
		slog.Warn("write welcome failed", "player_id", p.id, "error", err)
		return nil, false
	}
	return p, true
}

func writeMsg(ctx context.Context, conn *websocket.Conn, msgType string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, protocol.Envelope{Type: msgType, Data: raw})
}

func newPlayerID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failing means the host is broken
	}
	return hex.EncodeToString(b)
}

// randomCoord spawns players in the middle 60% of an axis so nobody starts
// pinned to a wall.
func randomCoord(max float64) float64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		panic(err)
	}
	return max*0.2 + max*0.6*(float64(n.Int64())/1000.0)
}
