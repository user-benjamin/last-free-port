// Package api holds the HTTP handlers, separated from cmd/api wiring so
// they're testable: handlers depend on the two small interfaces below, not
// on Postgres/Valkey directly, so tests run against fakes in microseconds.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
)

// UserStore is what the account endpoints need from the credential store.
// Register and Authenticate return a UserInfo or one of auth's sentinel
// errors (ErrUsernameTaken, ErrBadCredentials); the handlers map those to
// status codes.
type UserStore interface {
	Register(ctx context.Context, username, password string) (auth.UserInfo, error)
	Authenticate(ctx context.Context, username, password string) (auth.UserInfo, error)
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
	mux.HandleFunc("POST /v1/register", s.register)
	mux.HandleFunc("POST /v1/login", s.login)
	return mux
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionResponse struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	AccessToken string `json:"access_token"`
	Ticket      string `json:"ticket"`
}

const (
	minUsername, maxUsername = 2, 24
	// minPassword is a floor, not a policy — argon2id does the real work.
	// Length beats composition rules, so we only reject the trivially short.
	minPassword, maxPassword = 8, 200
)

// register creates an account and logs the player straight in (returns a
// session ticket), so the client needs one round-trip, not two.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	creds, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := s.users.Register(r.Context(), creds.Username, creds.Password)
	switch {
	case errors.Is(err, auth.ErrUsernameTaken):
		httpError(w, http.StatusConflict, "that name is already spoken for")
		return
	case err != nil:
		slog.Error("register: failed", "username", creds.Username, "error", err)
		httpError(w, http.StatusServiceUnavailable, "the harbormaster is unavailable")
		return
	}

	slog.Info("register", "user_id", user.UserID, "username", user.Username)
	s.issueSession(w, r, user)
}

// login verifies credentials and issues a session ticket. It deliberately
// returns the same 401 whether the username is unknown or the password is
// wrong (see auth.ErrBadCredentials).
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	creds, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := s.users.Authenticate(r.Context(), creds.Username, creds.Password)
	switch {
	case errors.Is(err, auth.ErrBadCredentials):
		httpError(w, http.StatusUnauthorized, "invalid username or password")
		return
	case err != nil:
		slog.Error("login: failed", "username", creds.Username, "error", err)
		httpError(w, http.StatusServiceUnavailable, "the harbormaster is unavailable")
		return
	}

	slog.Info("login", "user_id", user.UserID, "username", user.Username)
	s.issueSession(w, r, user)
}

// issueSession mints a ticket for an authenticated identity and writes the
// session response. Shared by register and login so the two stay in lockstep.
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, user auth.UserInfo) {
	ticket, err := s.tickets.Issue(r.Context(), user)
	if err != nil {
		slog.Error("ticket issue failed", "user_id", user.UserID, "error", err)
		httpError(w, http.StatusServiceUnavailable, "the harbormaster is unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessionResponse{
		UserID:   user.UserID,
		Username: user.Username,
		// Still a stub: a real bearer token (for future REST calls like
		// inventory queries) is a separate piece of work from credentials.
		AccessToken: "dev-token-" + user.UserID,
		Ticket:      ticket,
	})
}

// decodeCredentials parses and validates a username/password body, writing a
// 400 and returning ok=false on any problem. Validation mirrors the DB
// CHECK on username and gives the player a readable error.
func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var creds credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		httpError(w, http.StatusBadRequest, "request body must be JSON")
		return creds, false
	}
	creds.Username = strings.TrimSpace(creds.Username)
	if n := len([]rune(creds.Username)); n < minUsername || n > maxUsername {
		httpError(w, http.StatusBadRequest, "username must be 2-24 characters")
		return creds, false
	}
	// Password is not trimmed: leading/trailing spaces are legitimate.
	if n := len(creds.Password); n < minPassword || n > maxPassword {
		httpError(w, http.StatusBadRequest, "password must be 8-200 characters")
		return creds, false
	}
	return creds, true
}

func httpError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
