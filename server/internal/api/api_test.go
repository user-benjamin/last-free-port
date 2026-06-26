package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
)

// fakeUsers records the last call and returns canned results, so handler
// tests stay in microseconds and never touch argon2 or Postgres.
type fakeUsers struct {
	id          string
	registerErr error
	authErr     error
	gotUser     string
	gotPass     string
}

func (f *fakeUsers) Register(_ context.Context, username, password string) (auth.UserInfo, error) {
	f.gotUser, f.gotPass = username, password
	if f.registerErr != nil {
		return auth.UserInfo{}, f.registerErr
	}
	return auth.UserInfo{UserID: f.id, Username: username}, nil
}

func (f *fakeUsers) Authenticate(_ context.Context, username, password string) (auth.UserInfo, error) {
	f.gotUser, f.gotPass = username, password
	if f.authErr != nil {
		return auth.UserInfo{}, f.authErr
	}
	return auth.UserInfo{UserID: f.id, Username: username}, nil
}

type fakeTickets struct{ ticket string }

func (f fakeTickets) Issue(_ context.Context, _ auth.UserInfo) (string, error) {
	return f.ticket, nil
}

func post(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRegisterHappyPath(t *testing.T) {
	users := &fakeUsers{id: "uuid-1"}
	srv := New(users, fakeTickets{ticket: "t-123"})
	rec := post(t, srv.Routes(), "/v1/register", `{"username":"anne","password":"hunter2hunter"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var resp sessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.UserID != "uuid-1" || resp.Ticket != "t-123" || resp.Username != "anne" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if users.gotPass != "hunter2hunter" {
		t.Errorf("password not forwarded to store, got %q", users.gotPass)
	}
}

func TestLoginHappyPath(t *testing.T) {
	srv := New(&fakeUsers{id: "uuid-1"}, fakeTickets{ticket: "t-123"})
	rec := post(t, srv.Routes(), "/v1/login", `{"username":"anne","password":"hunter2hunter"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var resp sessionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.UserID != "uuid-1" || resp.Ticket != "t-123" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestRegisterUsernameTakenIs409(t *testing.T) {
	srv := New(&fakeUsers{registerErr: auth.ErrUsernameTaken}, fakeTickets{ticket: "t"})
	rec := post(t, srv.Routes(), "/v1/register", `{"username":"anne","password":"hunter2hunter"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for taken username, got %d", rec.Code)
	}
}

func TestLoginBadCredentialsIs401(t *testing.T) {
	srv := New(&fakeUsers{authErr: auth.ErrBadCredentials}, fakeTickets{ticket: "t"})
	rec := post(t, srv.Routes(), "/v1/login", `{"username":"anne","password":"wrongwrongwrong"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad credentials, got %d", rec.Code)
	}
	// The body must not reveal which half was wrong.
	if strings.Contains(rec.Body.String(), "username") && !strings.Contains(rec.Body.String(), "invalid username or password") {
		t.Errorf("401 body should be the generic message, got %s", rec.Body)
	}
}

func TestCredentialValidation(t *testing.T) {
	srv := New(&fakeUsers{id: "uuid-1"}, fakeTickets{ticket: "t"})
	cases := []struct {
		name string
		body string
	}{
		{"username too short", `{"username":"a","password":"hunter2hunter"}`},
		{"username empty", `{"username":"","password":"hunter2hunter"}`},
		{"password too short", `{"username":"anne","password":"short"}`},
		{"password missing", `{"username":"anne"}`},
		{"not json", `not json`},
	}
	for _, tc := range cases {
		for _, path := range []string{"/v1/register", "/v1/login"} {
			rec := post(t, srv.Routes(), path, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s on %s: expected 400, got %d", tc.name, path, rec.Code)
			}
		}
	}
}

func TestUsernameIsTrimmed(t *testing.T) {
	users := &fakeUsers{id: "uuid-1"}
	srv := New(users, fakeTickets{ticket: "t"})
	rec := post(t, srv.Routes(), "/v1/register", `{"username":"  anne  ","password":"hunter2hunter"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("trimmed valid username should pass, got %d", rec.Code)
	}
	if users.gotUser != "anne" {
		t.Errorf("expected trimmed username forwarded to store, got %q", users.gotUser)
	}
}
