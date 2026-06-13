// api handles accounts and sessions. All handler logic lives in
// internal/api; this binary only wires real infrastructure (Postgres,
// Valkey) into it. Fail fast on bad wiring: compose ordering guarantees
// dependencies are up, so a connection failure here is a real defect.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/user-benjamin/last-free-port/server/internal/api"
	"github.com/user-benjamin/last-free-port/server/internal/auth"
)

func main() {
	addr := envOr("API_ADDR", ":8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	valkeyAddr := envOr("VALKEY_ADDR", "127.0.0.1:6379")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err == nil {
		err = pool.Ping(ctx)
	}
	if err != nil {
		slog.Error("postgres unreachable", "error", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{Addr: valkeyAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("valkey unreachable", "addr", valkeyAddr, "error", err)
		os.Exit(1)
	}

	srv := api.New(auth.NewUsers(pool), auth.NewTickets(rdb))

	slog.Info("api listening", "addr", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
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
