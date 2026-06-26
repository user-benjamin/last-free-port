// game-server hosts the authoritative simulation. All world logic lives in
// internal/game; this binary wires it to HTTP, to Valkey (redeeming the
// session tickets the API issues, §16), and to Postgres (player inventory).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
	"github.com/user-benjamin/last-free-port/server/internal/game"
	"github.com/user-benjamin/last-free-port/server/internal/inventory"
)

func main() {
	addr := envOr("GAME_SERVER_ADDR", ":8081")
	valkeyAddr := envOr("VALKEY_ADDR", "127.0.0.1:6379")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: valkeyAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("valkey unreachable", "addr", valkeyAddr, "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err == nil {
		err = pool.Ping(ctx)
	}
	if err != nil {
		slog.Error("postgres unreachable", "error", err)
		os.Exit(1)
	}

	// GAME_ALLOWED_ORIGINS is a comma-separated allowlist of browser Origin
	// host patterns (e.g. "play.example.com,127.0.0.1:*"). Unset means
	// desktop and same-host clients only — see HandleWS.
	allowedOrigins := splitAndTrim(os.Getenv("GAME_ALLOWED_ORIGINS"))

	srv := game.NewServer(auth.NewTickets(rdb), inventory.NewStore(pool), allowedOrigins)
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

// splitAndTrim turns "a, b ,, c" into ["a","b","c"], dropping empty entries
// so a trailing comma or an unset var yields an empty slice, not [""].
func splitAndTrim(csv string) []string {
	var out []string
	for _, part := range strings.Split(csv, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
