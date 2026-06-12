// Package game is the authoritative simulation. The Hub goroutine owns all
// world state; nothing else may touch it. Connections talk to the Hub only
// through channels, which is what lets the tick loop run lock-free.
package game

import (
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/user-benjamin/last-free-port/server/internal/protocol"
)

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
)

type player struct {
	id   string
	name string
	x, y float64
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

// Hub runs the world. Create with NewHub, which starts the tick loop.
type Hub struct {
	joins   chan *player
	leaves  chan string
	moves   chan moveReq
	players map[string]*player
}

func NewHub() *Hub {
	h := &Hub{
		joins:   make(chan *player, 8),
		leaves:  make(chan string, 8),
		moves:   make(chan moveReq, 256),
		players: make(map[string]*player),
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
		case <-ticker.C:
			h.tick()
		}
	}
}

// tick advances every player and broadcasts one snapshot to all of them.
func (h *Hub) tick() {
	if len(h.players) == 0 {
		return
	}

	state := protocol.State{Players: make([]protocol.PlayerState, 0, len(h.players))}
	for _, p := range h.players {
		p.x = clamp(p.x+p.dx*MoveSpeed*tickDt, 0, WorldW)
		p.y = clamp(p.y+p.dy*MoveSpeed*tickDt, 0, WorldH)
		state.Players = append(state.Players, protocol.PlayerState{
			ID: p.id, Name: p.name, X: p.x, Y: p.y,
		})
	}

	frame, err := marshalEnvelope(protocol.TypeState, state)
	if err != nil {
		slog.Error("marshal state failed", "error", err)
		return
	}
	for _, p := range h.players {
		select {
		case p.send <- frame:
		default:
			// Slow client: drop this frame rather than stall the world.
			// It will catch up on the next snapshot.
		}
	}
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
