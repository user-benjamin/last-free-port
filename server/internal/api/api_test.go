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

type fakeUsers struct{ id string }

func (f fakeUsers) Upsert(_ context.Context, _ string) (string, error) { return f.id, nil }

type fakeTickets struct{ ticket string }

func (f fakeTickets) Issue(_ context.Context, _ auth.UserInfo) (string, error) {
	return f.ticket, nil
}

func post(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestLoginHappyPath(t *testing.T) {
	srv := New(fakeUsers{id: "uuid-1"}, fakeTickets{ticket: "t-123"})
	rec := post(t, srv.Routes(), `{"username":"anne"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var resp loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.UserID != "uuid-1" || resp.Ticket != "t-123" || resp.Username != "anne" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestLoginTrimsAndValidatesUsername(t *testing.T) {
	srv := New(fakeUsers{id: "uuid-1"}, fakeTickets{ticket: "t"})
	for _, bad := range []string{`{"username":"a"}`, `{"username":""}`, `{"username":"  x  "}`, `{}`, `not json`} {
		rec := post(t, srv.Routes(), bad)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: expected 400, got %d", bad, rec.Code)
		}
	}

	rec := post(t, srv.Routes(), `{"username":"  anne  "}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("trimmed valid username should pass, got %d", rec.Code)
	}
	var resp loginResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Username != "anne" {
		t.Errorf("expected trimmed username, got %q", resp.Username)
	}
}
