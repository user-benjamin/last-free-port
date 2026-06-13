package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

// TicketRedeemer is the one capability this package needs from the auth
// system: turn a ticket into an identity, exactly once. Declared here (at
// the consumer) so tests can swap in a miniredis-backed implementation.
type TicketRedeemer interface {
	Redeem(ctx context.Context, ticket string) (auth.UserInfo, error)
}

// Server accepts WebSocket connections and attaches them to the Hub.
type Server struct {
	hub     *Hub
	tickets TicketRedeemer
	// spawn is injectable so tests can place players deterministically.
	spawn func() (float64, float64)
}

func NewServer(tickets TicketRedeemer) *Server {
	npcs, err := loadNPCs()
	if err != nil {
		// Content is compiled in: if it's invalid, the binary is defective
		// and must not start. TestEmbeddedContentLoads catches this in CI
		// before it can ship.
		panic(err)
	}
	return &Server{hub: NewHub(npcs), tickets: tickets, spawn: randomSpawn}
}

func randomSpawn() (float64, float64) {
	return randomCoord(WorldW), randomCoord(WorldH)
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
		case protocol.TypeTalkIntent:
			var intent protocol.TalkIntent
			if err := json.Unmarshal(env.Data, &intent); err != nil {
				continue
			}
			s.hub.talks <- talkReq{playerID: p.id, npcID: intent.NPCID}
		default:
			// Unknown types are logged and ignored so old clients don't
			// break the session when the protocol grows.
			slog.Info("unhandled message", "player_id", p.id, "type", env.Type)
		}
	}
}

// handshake expects join (a single-use API-issued ticket) as the first
// message and replies with welcome. The player's identity comes from
// redeeming the ticket — never from anything the client claims (§16).
func (s *Server) handshake(ctx context.Context, conn *websocket.Conn) (*player, bool) {
	hsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var env protocol.Envelope
	if err := wsjson.Read(hsCtx, conn, &env); err != nil {
		slog.Warn("read join failed", "error", err)
		return nil, false
	}
	if env.Type != protocol.TypeJoin {
		writeMsg(hsCtx, conn, protocol.TypeError, protocol.Error{
			Code: "expected_join", Message: "first message must be join",
		})
		return nil, false
	}
	var join protocol.Join
	if err := json.Unmarshal(env.Data, &join); err != nil {
		writeMsg(hsCtx, conn, protocol.TypeError, protocol.Error{
			Code: "bad_join", Message: "join data was not valid JSON",
		})
		return nil, false
	}

	user, err := s.tickets.Redeem(hsCtx, join.Ticket)
	if errors.Is(err, auth.ErrInvalidTicket) {
		writeMsg(hsCtx, conn, protocol.TypeError, protocol.Error{
			Code: "bad_ticket", Message: "that ticket is no good here — log in again",
		})
		return nil, false
	}
	if err != nil {
		slog.Error("ticket redemption failed", "error", err)
		writeMsg(hsCtx, conn, protocol.TypeError, protocol.Error{
			Code: "auth_unavailable", Message: "the harbor office is closed — try again",
		})
		return nil, false
	}

	x, y := s.spawn()
	p := &player{
		id:     newPlayerID(),
		userID: user.UserID,
		name:   user.Username,
		x:      x,
		y:      y,
		send:   make(chan []byte, 8),
	}
	slog.Info("player connected", "player_id", p.id, "user_id", p.userID, "name", p.name)

	err = writeMsg(hsCtx, conn, protocol.TypeWelcome, protocol.Welcome{
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
