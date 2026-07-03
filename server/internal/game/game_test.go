package game

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/redis/go-redis/v9"

	"github.com/user-benjamin/last-free-port/server/internal/auth"
	"github.com/user-benjamin/last-free-port/server/internal/inventory"
	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

// fakeInventory is an in-memory InventoryStore for tests, so the suite needs
// no Postgres. It's mutex-guarded because Grant runs on the hub's async
// persistence goroutine while Load runs on a connection goroutine.
type fakeInventory struct {
	mu    sync.Mutex
	items map[string]map[string]int // userID -> itemType -> qty
}

func newFakeInventory() *fakeInventory {
	return &fakeInventory{items: map[string]map[string]int{}}
}

func (f *fakeInventory) Load(_ context.Context, userID string) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]int{}
	for k, v := range f.items[userID] {
		out[k] = v
	}
	return out, nil
}

func (f *fakeInventory) Grant(_ context.Context, userID, itemType string, n int) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.items[userID] == nil {
		f.items[userID] = map[string]int{}
	}
	f.items[userID][itemType] += n
	return f.items[userID][itemType], nil
}

// Craft mirrors inventory.Store.Craft's semantics in memory: all-or-nothing,
// returning inventory.ErrInsufficient if any input is short before mutating.
func (f *fakeInventory) Craft(_ context.Context, userID string, inputs map[string]int, output string, outputQty int) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	owned := f.items[userID]
	for item, need := range inputs {
		if owned[item] < need {
			return nil, inventory.ErrInsufficient
		}
	}
	if f.items[userID] == nil {
		f.items[userID] = map[string]int{}
	}
	changed := make(map[string]int, len(inputs)+1)
	for item, need := range inputs {
		f.items[userID][item] -= need
		changed[item] = f.items[userID][item]
	}
	f.items[userID][output] += outputQty
	changed[output] = f.items[userID][output]
	return changed, nil
}

// testEnv is a game server wired to an in-process Valkey (miniredis) and a
// fake inventory, so every test exercises the real ticket flow and gather
// path with no external dependencies.
type testEnv struct {
	srv     *httptest.Server
	tickets *auth.Tickets
	inv     *fakeInventory
}

func newTestEnv(t *testing.T, spawnX, spawnY float64, npcs []NPC) *testEnv {
	return newTestEnvFull(t, spawnX, spawnY, npcs, nil)
}

func newTestEnvFull(t *testing.T, spawnX, spawnY float64, npcs []NPC, resources []ResourceNode) *testEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	tickets := auth.NewTickets(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	inv := newFakeInventory()
	gs := &Server{
		hub:     NewHub(npcs, resources, nil, inv),
		tickets: tickets,
		inv:     inv,
		spawn:   func() (float64, float64) { return spawnX, spawnY },
	}
	srv := httptest.NewServer(http.HandlerFunc(gs.HandleWS))
	t.Cleanup(srv.Close)
	return &testEnv{srv: srv, tickets: tickets, inv: inv}
}

// newCraftEnv wires a server whose hub knows the given recipes, so craft tests
// can exercise the full craft_intent path. Players spawn at a fixed point;
// v1 crafting has no proximity requirement, so position doesn't matter.
func newCraftEnv(t *testing.T, recipes []Recipe) *testEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	tickets := auth.NewTickets(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	inv := newFakeInventory()
	gs := &Server{
		hub:     NewHub(nil, nil, recipes, inv),
		tickets: tickets,
		inv:     inv,
		spawn:   func() (float64, float64) { return 500, 500 },
	}
	srv := httptest.NewServer(http.HandlerFunc(gs.HandleWS))
	t.Cleanup(srv.Close)
	return &testEnv{srv: srv, tickets: tickets, inv: inv}
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

func sendGather(t *testing.T, ctx context.Context, conn *websocket.Conn, nodeID string) {
	t.Helper()
	intent, _ := json.Marshal(protocol.GatherIntent{NodeID: nodeID})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeGatherIntent, Data: intent}); err != nil {
		t.Fatalf("send gather: %v", err)
	}
}

func expectError(t *testing.T, ctx context.Context, conn *websocket.Conn, wantCode string) {
	t.Helper()
	envlp := readUntilType(t, ctx, conn, protocol.TypeError)
	var e protocol.Error
	if err := json.Unmarshal(envlp.Data, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Code != wantCode {
		t.Errorf("expected %q, got %q", wantCode, e.Code)
	}
}

func TestEmbeddedResourcesLoad(t *testing.T) {
	nodes, err := loadResources()
	if err != nil {
		t.Fatalf("embedded resources.json is invalid: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected at least one resource node in resources.json")
	}
}

func TestResourceNodesAppearInSnapshot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := ResourceNode{ID: "drift_test", Type: "driftwood", X: 100, Y: 100}
	env := newTestEnvFull(t, 500, 500, nil, []ResourceNode{node})

	conn, _ := env.join(t, ctx, "watcher")
	readStateUntil(t, ctx, conn, func(s protocol.State) bool {
		for _, n := range s.Resources {
			if n.ID == "drift_test" && n.Available {
				return true
			}
		}
		return false
	})
}

func TestGatherInRangeGrantsAndDepletes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := ResourceNode{ID: "drift_test", Type: "driftwood", X: 100, Y: 100}
	env := newTestEnvFull(t, 110, 100, nil, []ResourceNode{node}) // 10 units away

	conn, _ := env.join(t, ctx, "gatherer")
	sendGather(t, ctx, conn, "drift_test")

	// The new authoritative total comes back as an inventory message.
	envlp := readUntilType(t, ctx, conn, protocol.TypeInventory)
	var inv protocol.Inventory
	if err := json.Unmarshal(envlp.Data, &inv); err != nil {
		t.Fatalf("decode inventory: %v", err)
	}
	if inv.ItemType != "driftwood" || inv.Quantity != 1 {
		t.Errorf("expected driftwood=1, got %s=%d", inv.ItemType, inv.Quantity)
	}

	// And the node shows depleted in subsequent snapshots.
	readStateUntil(t, ctx, conn, func(s protocol.State) bool {
		for _, n := range s.Resources {
			if n.ID == "drift_test" {
				return !n.Available
			}
		}
		return false
	})
}

func TestGatherOutOfRangeRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := ResourceNode{ID: "drift_test", Type: "driftwood", X: 100, Y: 100}
	env := newTestEnvFull(t, 600, 600, nil, []ResourceNode{node}) // far away

	conn, _ := env.join(t, ctx, "reacher")
	sendGather(t, ctx, conn, "drift_test")
	expectError(t, ctx, conn, "too_far")
}

func TestGatherDepletedRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := ResourceNode{ID: "drift_test", Type: "driftwood", X: 100, Y: 100}
	env := newTestEnvFull(t, 110, 100, nil, []ResourceNode{node})

	conn, _ := env.join(t, ctx, "greedy")
	sendGather(t, ctx, conn, "drift_test")
	readUntilType(t, ctx, conn, protocol.TypeInventory) // first gather succeeds

	sendGather(t, ctx, conn, "drift_test") // second, while depleted
	expectError(t, ctx, conn, "depleted")
}

func TestGatherUnknownNodeRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newTestEnvFull(t, 100, 100, nil, nil)

	conn, _ := env.join(t, ctx, "confused")
	sendGather(t, ctx, conn, "no_such_node")
	expectError(t, ctx, conn, "no_such_node")
}

// TestInventoryPersistsAcrossReconnect is the milestone in one test: gather,
// disconnect, reconnect as the same account, and the welcome carries the item
// back. (The same username yields the same user id via the ticket.)
func TestInventoryPersistsAcrossReconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := ResourceNode{ID: "drift_test", Type: "driftwood", X: 100, Y: 100}
	env := newTestEnvFull(t, 110, 100, nil, []ResourceNode{node})

	conn, welcome := env.join(t, ctx, "anne")
	if len(welcome.Inventory) != 0 {
		t.Fatalf("new player should start empty, got %v", welcome.Inventory)
	}
	sendGather(t, ctx, conn, "drift_test")
	readUntilType(t, ctx, conn, protocol.TypeInventory) // ensure the grant landed
	conn.Close(websocket.StatusNormalClosure, "")

	// Reconnect as the same account.
	_, welcome2 := env.join(t, ctx, "anne")
	if welcome2.Inventory["driftwood"] != 1 {
		t.Errorf("expected driftwood=1 after reconnect, got %v", welcome2.Inventory)
	}
}

func TestResourceRespawns(t *testing.T) {
	// Shorten driftwood's respawn for this test only; tests run serially so
	// mutating the package var is safe with a deferred restore. Must span
	// several tick periods (50ms at 20Hz): tick() respawns before it
	// broadcasts, so a respawn <= one tick can flip the node back before any
	// snapshot ever shows it depleted.
	original := respawnByType["driftwood"]
	respawnByType["driftwood"] = 250 * time.Millisecond
	defer func() { respawnByType["driftwood"] = original }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	node := ResourceNode{ID: "drift_test", Type: "driftwood", X: 100, Y: 100}
	env := newTestEnvFull(t, 110, 100, nil, []ResourceNode{node})

	conn, _ := env.join(t, ctx, "patient")
	sendGather(t, ctx, conn, "drift_test")
	// Wait for depletion...
	readStateUntil(t, ctx, conn, func(s protocol.State) bool {
		for _, n := range s.Resources {
			if n.ID == "drift_test" {
				return !n.Available
			}
		}
		return false
	})
	// ...then for the respawn.
	readStateUntil(t, ctx, conn, func(s protocol.State) bool {
		for _, n := range s.Resources {
			if n.ID == "drift_test" {
				return n.Available
			}
		}
		return false
	})
}

// --- crafting ---

func hammerRecipe() Recipe {
	return Recipe{
		ID: "hammer", Output: "hammer", OutputQty: 1,
		Inputs: map[string]int{"driftwood": 1, "scrap_iron": 1},
	}
}

func sendCraft(t *testing.T, ctx context.Context, conn *websocket.Conn, recipeID string) {
	t.Helper()
	intent, _ := json.Marshal(protocol.CraftIntent{RecipeID: recipeID})
	if err := wsjson.Write(ctx, conn, protocol.Envelope{Type: protocol.TypeCraftIntent, Data: intent}); err != nil {
		t.Fatalf("send craft: %v", err)
	}
}

// readInventoryUpdates collects n inventory messages into item -> quantity. A
// craft emits one per affected item in no guaranteed order, so the caller
// reads them all and inspects the map.
func readInventoryUpdates(t *testing.T, ctx context.Context, conn *websocket.Conn, n int) map[string]int {
	t.Helper()
	out := map[string]int{}
	for i := 0; i < n; i++ {
		envlp := readUntilType(t, ctx, conn, protocol.TypeInventory)
		var inv protocol.Inventory
		if err := json.Unmarshal(envlp.Data, &inv); err != nil {
			t.Fatalf("decode inventory: %v", err)
		}
		out[inv.ItemType] = inv.Quantity
	}
	return out
}

func TestEmbeddedRecipesLoad(t *testing.T) {
	recipes, err := loadRecipes()
	if err != nil {
		t.Fatalf("embedded recipes.json is invalid: %v", err)
	}
	if len(recipes) == 0 {
		t.Fatal("expected at least one recipe in recipes.json")
	}
}

func TestRecipesAppearInWelcome(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newCraftEnv(t, []Recipe{hammerRecipe()})

	_, welcome := env.join(t, ctx, "smith")
	if len(welcome.Recipes) != 1 || welcome.Recipes[0].ID != "hammer" {
		t.Fatalf("expected hammer recipe in welcome, got %+v", welcome.Recipes)
	}
}

func TestCraftConsumesInputsAndGrantsOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newCraftEnv(t, []Recipe{hammerRecipe()})
	// Seed exactly the inputs. join derives the user id as "user-"+username.
	env.inv.Grant(ctx, "user-smith", "driftwood", 1)
	env.inv.Grant(ctx, "user-smith", "scrap_iron", 1)

	conn, _ := env.join(t, ctx, "smith")
	sendCraft(t, ctx, conn, "hammer")

	got := readInventoryUpdates(t, ctx, conn, 3) // 2 inputs + 1 output
	if got["hammer"] != 1 || got["driftwood"] != 0 || got["scrap_iron"] != 0 {
		t.Errorf("after craft want hammer=1 driftwood=0 scrap_iron=0, got %v", got)
	}
}

func TestCraftWithoutMaterialsRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newCraftEnv(t, []Recipe{hammerRecipe()})

	conn, _ := env.join(t, ctx, "broke") // empty inventory
	sendCraft(t, ctx, conn, "hammer")
	expectError(t, ctx, conn, "missing_materials")
}

func TestCraftUnknownRecipeRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	env := newCraftEnv(t, []Recipe{hammerRecipe()})

	conn, _ := env.join(t, ctx, "dreamer")
	sendCraft(t, ctx, conn, "trebuchet")
	expectError(t, ctx, conn, "no_such_recipe")
}
