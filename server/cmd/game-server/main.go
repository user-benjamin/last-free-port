// game-server hosts the authoritative simulation. All world logic lives in
// internal/game; this binary just wires it to HTTP.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/user-benjamin/last-free-port/server/internal/game"
)

func main() {
	addr := envOr("GAME_SERVER_ADDR", ":8081")

	srv := game.NewServer()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /ws", srv.HandleWS)

	slog.Info("game-server listening", "addr", addr, "tick_rate", game.TickRate)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
