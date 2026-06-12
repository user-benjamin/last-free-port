package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"lastfreeport/internal/protocol"
)

func TestHandshake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	hello, _ := json.Marshal(protocol.Hello{Name: "test-pirate"})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeHello, Data: hello}); err != nil {
		t.Fatalf("send hello: %v", err)
	}

	var env protocol.Envelope
	if err := wsjson.Read(ctx, conn, &env); err != nil {
		t.Fatalf("read reply: %v", err)
	}
	if env.Type != protocol.TypeWelcome {
		t.Fatalf("expected welcome, got %q", env.Type)
	}

	var welcome protocol.Welcome
	if err := json.Unmarshal(env.Data, &welcome); err != nil {
		t.Fatalf("decode welcome: %v", err)
	}
	if welcome.PlayerID == "" {
		t.Error("welcome has empty player_id")
	}
	if welcome.Motd == "" {
		t.Error("welcome has empty motd")
	}

	conn.Close(websocket.StatusNormalClosure, "")
}

func TestRejectsNonHelloFirstMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: "move_intent"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	var env protocol.Envelope
	if err := wsjson.Read(ctx, conn, &env); err != nil {
		t.Fatalf("read reply: %v", err)
	}
	if env.Type != protocol.TypeError {
		t.Fatalf("expected error, got %q", env.Type)
	}
}
