package game

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/redis/go-redis/v9"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

// testEnv is a game server wired to an in-process Valkey (miniredis), so
// every test exercises the real ticket Issue/Redeem path with no external
// dependencies.
type testEnv struct {
	srv     *httptest.Server
	tickets *auth.Tickets
}

func newTestEnv(t *testing.T, spawnX, spawnY float64, npcs []NPC) *testEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	tickets := auth.NewTickets(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	gs := &Server{
		hub:     NewHub(npcs),
		tickets: tickets,
		spawn:   func() (float64, float64) { return spawnX, spawnY },
	}
	srv := httptest.NewServer(http.HandlerFunc(gs.HandleWS))
	t.Cleanup(srv.Close)
	return &testEnv{srv: srv, tickets: tickets}
}

func (e *testEnv) dialRaw(t *testing.T, ctx context.Context) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http")
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// join performs the full ticket flow: issue against Valkey, send join,
// expect welcome.
func (e *testEnv) join(t *testing.T, ctx context.Context, username string) (*websocket.Conn, protocol.Welcome) {
	t.Helper()
	ticket, err := e.tickets.Issue(ctx, auth.UserInfo{UserID: "user-" + username, Username: username})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	conn := e.dialRaw(t, ctx)
	joinMsg, _ := json.Marshal(protocol.Join{Ticket: ticket})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeJoin, Data: joinMsg}); err != nil {
		t.Fatalf("send join: %v", err)
	}

	env := readUntilType(t, ctx, conn, protocol.TypeWelcome)
	var welcome protocol.Welcome
	if err := json.Unmarshal(env.Data, &welcome); err != nil {
		t.Fatalf("decode welcome: %v", err)
	}
	return conn, welcome
}

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

func readStateUntil(t *testing.T, ctx context.Context, conn *websocket.Conn, ok func(protocol.State) bool) protocol.State {
	t.Helper()
	for {
		env := readUntilType(t, ctx, conn, protocol.TypeState)
		var state protocol.State
		if err := json.Unmarshal(env.Data, &state); err != nil {
			t.Fatalf("decode state: %v", err)
		}
		if ok(state) {
			return state
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

func TestJoinWithValidTicket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newTestEnv(t, 500, 500, nil)

	_, welcome := env.join(t, ctx, "anne")
	if welcome.PlayerID == "" {
		t.Error("welcome has empty player_id")
	}
	if welcome.SpawnX != 500 || welcome.SpawnY != 500 {
		t.Errorf("unexpected spawn (%f, %f)", welcome.SpawnX, welcome.SpawnY)
	}
}

func TestJoinWithInvalidTicketIsRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newTestEnv(t, 500, 500, nil)

	conn := env.dialRaw(t, ctx)
	joinMsg, _ := json.Marshal(protocol.Join{Ticket: "forged-nonsense"})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeJoin, Data: joinMsg}); err != nil {
		t.Fatalf("send join: %v", err)
	}

	envlp := readUntilType(t, ctx, conn, protocol.TypeError)
	var e protocol.Error
	if err := json.Unmarshal(envlp.Data, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Code != "bad_ticket" {
		t.Errorf("expected bad_ticket, got %q", e.Code)
	}
}

func TestTicketIsSingleUse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newTestEnv(t, 500, 500, nil)

	ticket, err := env.tickets.Issue(ctx, auth.UserInfo{UserID: "u1", Username: "anne"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	joinMsg, _ := json.Marshal(protocol.Join{Ticket: ticket})

	// First use succeeds.
	conn1 := env.dialRaw(t, ctx)
	if err := wsjson.Write(ctx, conn1, protocol.Envelope{Type: protocol.TypeJoin, Data: joinMsg}); err != nil {
		t.Fatalf("send join: %v", err)
	}
	readUntilType(t, ctx, conn1, protocol.TypeWelcome)

	// Replaying the same ticket must fail.
	conn2 := env.dialRaw(t, ctx)
	if err := wsjson.Write(ctx, conn2, protocol.Envelope{Type: protocol.TypeJoin, Data: joinMsg}); err != nil {
		t.Fatalf("send join: %v", err)
	}
	envlp := readUntilType(t, ctx, conn2, protocol.TypeError)
	var e protocol.Error
	if err := json.Unmarshal(envlp.Data, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Code != "bad_ticket" {
		t.Errorf("expected bad_ticket on replay, got %q", e.Code)
	}
}

func TestRejectsNonJoinFirstMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newTestEnv(t, 500, 500, nil)

	conn := env.dialRaw(t, ctx)
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeMoveIntent}); err != nil {
		t.Fatalf("send: %v", err)
	}
	envlp := readUntilType(t, ctx, conn, protocol.TypeError)
	var e protocol.Error
	if err := json.Unmarshal(envlp.Data, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Code != "expected_join" {
		t.Errorf("expected expected_join, got %q", e.Code)
	}
}

func TestMovementIsServerAuthoritative(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	env := newTestEnv(t, 500, 500, nil)

	conn, welcome := env.join(t, ctx, "mover")

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

func TestPlayersSeeEachOther(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	env := newTestEnv(t, 500, 500, nil)

	_, welcomeA := env.join(t, ctx, "anne")
	connB, welcomeB := env.join(t, ctx, "bartholomew")

	readStateUntil(t, ctx, connB, func(s protocol.State) bool {
		seen := map[string]bool{}
		for _, p := range s.Players {
			seen[p.ID] = true
		}
		return seen[welcomeA.PlayerID] && seen[welcomeB.PlayerID]
	})
}

func TestNPCsAppearInSnapshots(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	npc := NPC{ID: "npc_test", Name: "Testy", X: 100, Y: 100, Lines: []string{"arr"}}
	env := newTestEnv(t, 500, 500, []NPC{npc})

	conn, _ := env.join(t, ctx, "watcher")
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
	env := newTestEnv(t, 110, 100, []NPC{npc}) // 10 units away

	conn, _ := env.join(t, ctx, "talker")
	intent, _ := json.Marshal(protocol.TalkIntent{NPCID: "npc_test"})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeTalkIntent, Data: intent}); err != nil {
		t.Fatalf("send talk: %v", err)
	}

	envlp := readUntilType(t, ctx, conn, protocol.TypeDialogue)
	var d protocol.Dialogue
	if err := json.Unmarshal(envlp.Data, &d); err != nil {
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
	env := newTestEnv(t, 600, 600, []NPC{npc}) // far across the island

	conn, _ := env.join(t, ctx, "shouter")
	intent, _ := json.Marshal(protocol.TalkIntent{NPCID: "npc_test"})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeTalkIntent, Data: intent}); err != nil {
		t.Fatalf("send talk: %v", err)
	}

	envlp := readUntilType(t, ctx, conn, protocol.TypeError)
	var e protocol.Error
	if err := json.Unmarshal(envlp.Data, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Code != "too_far" {
		t.Errorf("expected too_far, got %q", e.Code)
	}
}
