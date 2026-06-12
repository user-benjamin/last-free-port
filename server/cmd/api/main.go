// api handles accounts, sessions, inventory, crafting, and base persistence.
// Phase 0 scope: health endpoint and a stub login that returns fake
// credentials. Real credential validation, Postgres-backed accounts, and
// short-lived game-server tickets arrive with the auth flow (proposal §16).
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	addr := envOr("API_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /v1/login", login)

	slog.Info("api listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

type loginRequest struct {
	Username string `json:"username"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	Ticket      string `json:"ticket"`
}

func login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
		return
	}

	slog.Info("login", "username", req.Username)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{
		AccessToken: "dev-token-" + req.Username,
		Ticket:      "dev-ticket-" + req.Username,
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
