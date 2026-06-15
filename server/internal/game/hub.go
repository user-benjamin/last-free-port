// Package game is the authoritative simulation. The Hub goroutine owns all
// world state; nothing else may touch it. Connections talk to the Hub only
// through channels, which is what lets the tick loop run lock-free.
package game

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

// InventoryStore is what the simulation needs from the persistence layer:
// load a player's inventory on join, and grant items on gather. Declared at
// the consumer so tests inject an in-memory fake and CI needs no Postgres;
// *inventory.Store is the production implementation.
type InventoryStore interface {
	Load(ctx context.Context, userID string) (map[string]int, error)
	Grant(ctx context.Context, userID, itemType string, n int) (int, error)
}

const (
	// TickRate is how many times per second the world advances and a
	// snapshot is broadcast. 20Hz is plenty for walking-speed gameplay.
	TickRate = 20
	tickDt   = 1.0 / TickRate

	// MoveSpeed is world units per second. Only the server knows this —
	// clients send direction, never distance.
	MoveSpeed = 180.0

	// World bounds. The cove is one rectangle for now.
	WorldW = 1280.0
	WorldH = 720.0

	// TalkRange is how close (world units) a player must stand to an NPC
	// to talk. Enforced server-side; the client prompt is advisory.
	TalkRange = 48.0
)

type player struct {
	id string
	// userID is the stable Postgres identity from the redeemed ticket —
	// what persistent state (inventory, bases) will key on. id is the
	// per-session presence; userID is the account.
	userID string
	name   string
	x, y   float64
	// dx/dy are the player's current held direction, already clamped to
	// unit length. Applied every tick until a new MoveIntent replaces it.
	dx, dy float64
	// send carries pre-marshaled frames to this player's write pump.
	send chan []byte
}

type moveReq struct {
	id     string
	dx, dy float64
}

type talkReq struct {
	playerID string
	npcID    string
}

type gatherReq struct {
	playerID string
	nodeID   string
}

// Hub runs the world. Create with NewHub, which starts the tick loop.
type Hub struct {
	joins   chan *player
	leaves  chan string
	moves   chan moveReq
	talks   chan talkReq
	gathers chan gatherReq
	players map[string]*player
	npcs    []NPC
	// npcStates is precomputed once: NPCs don't move yet, so their part of
	// every snapshot is identical.
	npcStates []protocol.NPCState
	// resources are the live gatherable nodes; only the hub goroutine
	// touches their availability/respawn state.
	resources []*resourceNode
	inv       InventoryStore
}

func NewHub(npcs []NPC, resources []ResourceNode, inv InventoryStore) *Hub {
	h := &Hub{
		joins:     make(chan *player, 8),
		leaves:    make(chan string, 8),
		moves:     make(chan moveReq, 256),
		talks:     make(chan talkReq, 64),
		gathers:   make(chan gatherReq, 64),
		players:   make(map[string]*player),
		npcs:      npcs,
		resources: newResourceNodes(resources),
		inv:       inv,
	}
	for _, n := range npcs {
		h.npcStates = append(h.npcStates, protocol.NPCState{ID: n.ID, Name: n.Name, X: n.X, Y: n.Y})
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()
	for {
		select {
		case p := <-h.joins:
			h.players[p.id] = p
			slog.Info("player joined world", "player_id", p.id, "name", p.name)
		case id := <-h.leaves:
			delete(h.players, id)
			slog.Info("player left world", "player_id", id)
		case m := <-h.moves:
			if p, ok := h.players[m.id]; ok {
				p.dx, p.dy = clampDir(m.dx, m.dy)
			}
		case t := <-h.talks:
			h.talk(t)
		case g := <-h.gathers:
			h.gather(g)
		case <-ticker.C:
			h.tick()
		}
	}
}

// gather validates a gather request against authoritative state on the hub
// goroutine (proximity + node availability), then depletes the node and
// hands the Postgres write to a separate goroutine. The DB write must never
// run inline: a slow database would stall the single hub goroutine and
// freeze the world for everyone. State is therefore optimistic — the node
// depletes immediately and persistence catches up.
func (h *Hub) gather(g gatherReq) {
	p, ok := h.players[g.playerID]
	if !ok {
		return
	}
	for _, n := range h.resources {
		if n.def.ID != g.nodeID {
			continue
		}
		if !n.available {
			h.sendTo(p, protocol.TypeError, protocol.Error{
				Code: "depleted", Message: "nothing left here — give it time",
			})
			return
		}
		if math.Hypot(p.x-n.def.X, p.y-n.def.Y) > GatherRange {
			h.sendTo(p, protocol.TypeError, protocol.Error{
				Code: "too_far", Message: "you're not close enough to gather that",
			})
			return
		}
		n.available = false
		n.respawnAt = time.Now().Add(respawnByType[n.def.Type])
		slog.Info("gather", "player_id", p.id, "node_id", n.def.ID, "type", n.def.Type)
		// Capture stable, immutable fields for the async write — never the
		// player pointer or hub state.
		go h.persistGather(p.userID, p.send, n.def.Type)
		return
	}
	h.sendTo(p, protocol.TypeError, protocol.Error{
		Code: "no_such_node", Message: "there's nothing there to gather",
	})
}

// persistGather runs off the hub goroutine: it writes the grant to Postgres
// and reports the new authoritative total to the player. send is a buffered
// channel safe to use from any goroutine; userID and the channel are set at
// connect time and never mutated, so reading them here is race-free.
func (h *Hub) persistGather(userID string, send chan []byte, itemType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	qty, err := h.inv.Grant(ctx, userID, itemType, 1)
	if err != nil {
		slog.Error("gather grant failed", "user_id", userID, "type", itemType, "error", err)
		trySend(send, mustFrame(protocol.TypeError, protocol.Error{
			Code: "grant_failed", Message: "couldn't pocket that — try again",
		}))
		return
	}
	trySend(send, mustFrame(protocol.TypeInventory, protocol.Inventory{
		ItemType: itemType, Quantity: qty,
	}))
}

// talk validates a conversation request and pushes the reply straight into
// the requesting player's send channel. Range is checked here, against the
// authoritative positions — the client's "you are close enough" prompt is
// cosmetic.
func (h *Hub) talk(t talkReq) {
	p, ok := h.players[t.playerID]
	if !ok {
		return
	}
	for _, n := range h.npcs {
		if n.ID != t.npcID {
			continue
		}
		if math.Hypot(p.x-n.X, p.y-n.Y) > TalkRange {
			h.sendTo(p, protocol.TypeError, protocol.Error{
				Code: "too_far", Message: n.Name + " can't hear you from there.",
			})
			return
		}
		line := n.Lines[rand.IntN(len(n.Lines))]
		slog.Info("npc talk", "player_id", p.id, "npc_id", n.ID)
		h.sendTo(p, protocol.TypeDialogue, protocol.Dialogue{
			NPCID: n.ID, NPCName: n.Name, Line: line,
		})
		return
	}
	h.sendTo(p, protocol.TypeError, protocol.Error{
		Code: "no_such_npc", Message: "nobody by that name here",
	})
}

func (h *Hub) sendTo(p *player, msgType string, data any) {
	trySend(p.send, mustFrame(msgType, data))
}

// mustFrame marshals a message, logging and returning nil on the (should be
// impossible) marshal failure so callers can skip the send.
func mustFrame(msgType string, data any) []byte {
	frame, err := marshalEnvelope(msgType, data)
	if err != nil {
		slog.Error("marshal failed", "type", msgType, "error", err)
		return nil
	}
	return frame
}

// trySend delivers a frame without ever blocking: a backed-up client drops
// the frame rather than stalling the sender (the hub goroutine, or a gather
// persistence goroutine).
func trySend(send chan []byte, frame []byte) {
	if frame == nil {
		return
	}
	select {
	case send <- frame:
	default:
	}
}

// tick respawns due nodes, advances every player, and broadcasts one
// snapshot. Respawns run even with nobody connected so timers stay honest;
// the broadcast is skipped when the cove is empty.
func (h *Hub) tick() {
	now := time.Now()
	for _, n := range h.resources {
		if !n.available && now.After(n.respawnAt) {
			n.available = true
		}
	}

	if len(h.players) == 0 {
		return
	}

	state := protocol.State{
		Players:   make([]protocol.PlayerState, 0, len(h.players)),
		NPCs:      h.npcStates,
		Resources: h.resourceStates(),
	}
	for _, p := range h.players {
		p.x = clamp(p.x+p.dx*MoveSpeed*tickDt, 0, WorldW)
		p.y = clamp(p.y+p.dy*MoveSpeed*tickDt, 0, WorldH)
		state.Players = append(state.Players, protocol.PlayerState{
			ID: p.id, Name: p.name, X: p.x, Y: p.y,
		})
	}

	frame := mustFrame(protocol.TypeState, state)
	for _, p := range h.players {
		trySend(p.send, frame)
	}
}

// resourceStates snapshots the live nodes for broadcast. Built per tick (not
// precomputed like NPCs) because availability changes; cheap at this count.
func (h *Hub) resourceStates() []protocol.ResourceState {
	out := make([]protocol.ResourceState, len(h.resources))
	for i, n := range h.resources {
		out[i] = protocol.ResourceState{
			ID: n.def.ID, Type: n.def.Type, X: n.def.X, Y: n.def.Y, Available: n.available,
		}
	}
	return out
}

// clampDir caps the intent vector at unit length and rejects garbage, so a
// modified client can't move faster by sending dx=9000.
func clampDir(dx, dy float64) (float64, float64) {
	if math.IsNaN(dx) || math.IsNaN(dy) || math.IsInf(dx, 0) || math.IsInf(dy, 0) {
		return 0, 0
	}
	if length := math.Hypot(dx, dy); length > 1 {
		return dx / length, dy / length
	}
	return dx, dy
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}

func marshalEnvelope(msgType string, data any) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(protocol.Envelope{Type: msgType, Data: raw})
}
