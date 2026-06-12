package game

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

	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(NewServer().HandleWS))
	t.Cleanup(srv.Close)
	return srv
}

// newTestServerAt pins spawn position and NPC roster for deterministic
// proximity tests.
func newTestServerAt(t *testing.T, x, y float64, npcs []NPC) *httptest.Server {
	t.Helper()
	gs := &Server{
		hub:   NewHub(npcs),
		spawn: func() (float64, float64) { return x, y },
	}
	srv := httptest.NewServer(http.HandlerFunc(gs.HandleWS))
	t.Cleanup(srv.Close)
	return srv
}

// readUntilType skips snapshots and other traffic until a message of the
// wanted type arrives.
func readUntilType(t *testing.T, ctx context.Context, conn *websocket.Conn, wanted string) protocol.Envelope {
	t.Helper()
	for {
		var env protocol.Envelope
		if err := wsjson.Read(ctx, conn, &env); err != nil {
			t.Fatalf("read (waiting for %q): %v", wanted, err)
		}
		if env.Type == wanted {
			return env
		}
	}
}

func dial(t *testing.T, ctx context.Context, srv *httptest.Server, name string) (*websocket.Conn, protocol.Welcome) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })

	hello, _ := json.Marshal(protocol.Hello{Name: name})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeHello, Data: hello}); err != nil {
		t.Fatalf("send hello: %v", err)
	}
	var env protocol.Envelope
	if err := wsjson.Read(ctx, conn, &env); err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	if env.Type != protocol.TypeWelcome {
		t.Fatalf("expected welcome, got %q", env.Type)
	}
	var welcome protocol.Welcome
	if err := json.Unmarshal(env.Data, &welcome); err != nil {
		t.Fatalf("decode welcome: %v", err)
	}
	return conn, welcome
}

// readStateUntil keeps reading snapshots until ok returns true or the
// context expires.
func readStateUntil(t *testing.T, ctx context.Context, conn *websocket.Conn, ok func(protocol.State) bool) protocol.State {
	t.Helper()
	for {
		var env protocol.Envelope
		if err := wsjson.Read(ctx, conn, &env); err != nil {
			t.Fatalf("read state: %v", err)
		}
		if env.Type != protocol.TypeState {
			continue
		}
		var state protocol.State
		if err := json.Unmarshal(env.Data, &state); err != nil {
			t.Fatalf("decode state: %v", err)
		}
		if ok(state) {
			return state
		}
	}
}

func TestHandshake(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv := newTestServer(t)

	_, welcome := dial(t, ctx, srv, "test-pirate")
	if welcome.PlayerID == "" {
		t.Error("welcome has empty player_id")
	}
	if welcome.SpawnX < 0 || welcome.SpawnX > WorldW || welcome.SpawnY < 0 || welcome.SpawnY > WorldH {
		t.Errorf("spawn (%f, %f) outside world bounds", welcome.SpawnX, welcome.SpawnY)
	}
}

func TestRejectsNonHelloFirstMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv := newTestServer(t)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeMoveIntent}); err != nil {
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

func TestMovementIsServerAuthoritative(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv := newTestServer(t)

	conn, welcome := dial(t, ctx, srv, "mover")

	// Push east. Position should increase over ticks but never exceed the
	// world bound, no matter how absurd the intent vector.
	intent, _ := json.Marshal(protocol.MoveIntent{DX: 9000, DY: 0})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeMoveIntent, Data: intent}); err != nil {
		t.Fatalf("send intent: %v", err)
	}

	moved := readStateUntil(t, ctx, conn, func(s protocol.State) bool {
		for _, p := range s.Players {
			if p.ID == welcome.PlayerID && p.X > welcome.SpawnX {
				return true
			}
		}
		return false
	})

	// The clamped unit vector at MoveSpeed for ~one tick: distance moved
	// must be plausible, not the 9000x a trusting server would allow.
	for _, p := range moved.Players {
		if p.ID != welcome.PlayerID {
			continue
		}
		if p.X > WorldW {
			t.Errorf("position %f exceeds world bound %f", p.X, WorldW)
		}
		if delta := p.X - welcome.SpawnX; delta > MoveSpeed {
			t.Errorf("moved %f units in under a second — speed not clamped", delta)
		}
	}
}

func TestEmbeddedContentLoads(t *testing.T) {
	npcs, err := loadNPCs()
	if err != nil {
		t.Fatalf("embedded npcs.json is invalid: %v", err)
	}
	if len(npcs) == 0 {
		t.Fatal("expected at least one NPC in npcs.json")
	}
}

func TestNPCsAppearInSnapshots(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	npc := NPC{ID: "npc_test", Name: "Testy", X: 100, Y: 100, Lines: []string{"arr"}}
	srv := newTestServerAt(t, 500, 500, []NPC{npc})

	conn, _ := dial(t, ctx, srv, "watcher")
	readStateUntil(t, ctx, conn, func(s protocol.State) bool {
		for _, n := range s.NPCs {
			if n.ID == "npc_test" && n.X == 100 && n.Y == 100 {
				return true
			}
		}
		return false
	})
}

func TestTalkInRange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	npc := NPC{ID: "npc_test", Name: "Testy", X: 100, Y: 100, Lines: []string{"arr", "yo ho"}}
	srv := newTestServerAt(t, 110, 100, []NPC{npc}) // 10 units away

	conn, _ := dial(t, ctx, srv, "talker")
	intent, _ := json.Marshal(protocol.TalkIntent{NPCID: "npc_test"})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeTalkIntent, Data: intent}); err != nil {
		t.Fatalf("send talk: %v", err)
	}

	env := readUntilType(t, ctx, conn, protocol.TypeDialogue)
	var d protocol.Dialogue
	if err := json.Unmarshal(env.Data, &d); err != nil {
		t.Fatalf("decode dialogue: %v", err)
	}
	if d.NPCName != "Testy" || d.Line == "" {
		t.Errorf("unexpected dialogue: %+v", d)
	}
}

func TestTalkOutOfRangeIsRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	npc := NPC{ID: "npc_test", Name: "Testy", X: 100, Y: 100, Lines: []string{"arr"}}
	srv := newTestServerAt(t, 600, 600, []NPC{npc}) // far across the island

	conn, _ := dial(t, ctx, srv, "shouter")
	intent, _ := json.Marshal(protocol.TalkIntent{NPCID: "npc_test"})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeTalkIntent, Data: intent}); err != nil {
		t.Fatalf("send talk: %v", err)
	}

	env := readUntilType(t, ctx, conn, protocol.TypeError)
	var e protocol.Error
	if err := json.Unmarshal(env.Data, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Code != "too_far" {
		t.Errorf("expected too_far, got %q", e.Code)
	}
}

func TestPlayersSeeEachOther(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv := newTestServer(t)

	_, welcomeA := dial(t, ctx, srv, "anne")
	connB, welcomeB := dial(t, ctx, srv, "bartholomew")

	readStateUntil(t, ctx, connB, func(s protocol.State) bool {
		seen := map[string]bool{}
		for _, p := range s.Players {
			seen[p.ID] = true
		}
		return seen[welcomeA.PlayerID] && seen[welcomeB.PlayerID]
	})
}
