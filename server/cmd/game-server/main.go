// game-server is the authoritative game server. Phase 0 scope: accept a
// WebSocket connection, complete the hello/welcome handshake, and hold the
// connection open. The session read loop here grows into the simulation
// tick's input feed in Phase 2.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

func main() {
	addr := envOr("GAME_SERVER_ADDR", ":8081")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /ws", handleWS)

	slog.Info("game-server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
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

	handshakeCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var env protocol.Envelope
	if err := wsjson.Read(handshakeCtx, conn, &env); err != nil {
		slog.Warn("read hello failed", "error", err)
		return
	}
	if env.Type != protocol.TypeHello {
		writeMsg(handshakeCtx, conn, protocol.TypeError, protocol.Error{
			Code:    "expected_hello",
			Message: "first message must be hello",
		})
		return
	}
	var hello protocol.Hello
	if err := json.Unmarshal(env.Data, &hello); err != nil {
		writeMsg(handshakeCtx, conn, protocol.TypeError, protocol.Error{
			Code:    "bad_hello",
			Message: "hello data was not valid JSON",
		})
		return
	}

	playerID := newPlayerID()
	slog.Info("player connected", "player_id", playerID, "name", hello.Name)

	err = writeMsg(handshakeCtx, conn, protocol.TypeWelcome, protocol.Welcome{
		PlayerID:   playerID,
		ServerTime: time.Now().UTC().Format(time.RFC3339),
		Motd:       "No flag, no fortune. Welcome to the cove.",
	})
	if err != nil {
		slog.Warn("write welcome failed", "player_id", playerID, "error", err)
		return
	}

	for {
		var next protocol.Envelope
		if err := wsjson.Read(r.Context(), conn, &next); err != nil {
			slog.Info("player disconnected", "player_id", playerID)
			return
		}
		// Unknown types are logged and ignored so old clients don't break
		// the session when the protocol grows.
		slog.Info("message received", "player_id", playerID, "type", next.Type)
	}
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
