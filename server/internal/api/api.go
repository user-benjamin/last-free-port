// Package api holds the HTTP handlers, separated from cmd/api wiring so
// they're testable: handlers depend on the two small interfaces below, not
// on Postgres/Valkey directly, so tests run against fakes in microseconds.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
)

// UserStore is what login needs from the account store.
type UserStore interface {
	Upsert(ctx context.Context, username string) (string, error)
}

// TicketIssuer is what login needs from the ticket mint.
type TicketIssuer interface {
	Issue(ctx context.Context, user auth.UserInfo) (string, error)
}

type Server struct {
	users   UserStore
	tickets TicketIssuer
}

func New(users UserStore, tickets TicketIssuer) *Server {
	return &Server{users: users, tickets: tickets}
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /v1/login", s.login)
	return mux
}

type loginRequest struct {
	Username string `json:"username"`
}

type loginResponse struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	AccessToken string `json:"access_token"`
	Ticket      string `json:"ticket"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "request body must be JSON")
		return
	}
	username := strings.TrimSpace(req.Username)
	// Same rule the DB CHECK enforces — validating here too gives the
	// player a readable error instead of a constraint violation.
	if n := len([]rune(username)); n < 2 || n > 24 {
		httpError(w, http.StatusBadRequest, "username must be 2-24 characters")
		return
	}

	userID, err := s.users.Upsert(r.Context(), username)
	if err != nil {
		slog.Error("login: upsert failed", "username", username, "error", err)
		httpError(w, http.StatusServiceUnavailable, "the harbormaster is unavailable")
		return
	}

	ticket, err := s.tickets.Issue(r.Context(), auth.UserInfo{UserID: userID, Username: username})
	if err != nil {
		slog.Error("login: ticket issue failed", "user_id", userID, "error", err)
		httpError(w, http.StatusServiceUnavailable, "the harbormaster is unavailable")
		return
	}

	slog.Info("login", "user_id", userID, "username", username)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{
		UserID:   userID,
		Username: username,
		// Still a stub: a real access token (for future REST calls like
		// inventory queries) arrives with real credentials.
		AccessToken: "dev-token-" + userID,
		Ticket:      ticket,
	})
}

func httpError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
