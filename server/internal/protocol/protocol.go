// Package protocol defines the wire messages exchanged between the game
// client and the backend services. docs/protocol.md is the human-readable
// version of this contract; keep the two in sync.
package protocol

import "encoding/json"

// Envelope wraps every message in both directions.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Message types.
const (
	TypeJoin         = "join"
	TypeWelcome      = "welcome"
	TypeError        = "error"
	TypeMoveIntent   = "move_intent"
	TypeState        = "state"
	TypeTalkIntent   = "talk_intent"
	TypeDialogue     = "dialogue"
	TypeGatherIntent = "gather_intent"
	TypeInventory    = "inventory"
)

// Join is the first message a client sends after connecting: a single-use
// session ticket issued by the API (proposal §16). The client never states
// its own identity — the name and user id come out of redeeming the ticket.
type Join struct {
	Ticket string `json:"ticket"`
}

// Welcome is the server's reply to a valid Join. Inventory is the player's
// full inventory at join time (item_type -> quantity), loaded from Postgres
// — this is the moment a returning player gets their stuff back.
type Welcome struct {
	PlayerID   string         `json:"player_id"`
	ServerTime string         `json:"server_time"`
	Motd       string         `json:"motd"`
	SpawnX     float64        `json:"spawn_x"`
	SpawnY     float64        `json:"spawn_y"`
	WorldW     float64        `json:"world_w"`
	WorldH     float64        `json:"world_h"`
	Inventory  map[string]int `json:"inventory"`
}

// MoveIntent is the client's held movement direction (each component
// -1..1). The client can only ever say which way it is pushing — the
// server clamps the vector and integrates position itself, so a hacked
// client cannot teleport or speed-hack.
type MoveIntent struct {
	DX float64 `json:"dx"`
	DY float64 `json:"dy"`
}

// PlayerState is one player's authoritative position within a State.
type PlayerState struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// NPCState is one NPC's authoritative position within a State.
type NPCState struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// ResourceState is one gatherable node within a State. Available flips to
// false when gathered and back to true after its respawn timer; the client
// renders it dimmed/hidden while depleted.
type ResourceState struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Available bool    `json:"available"`
}

// State is the world snapshot broadcast to every connected client each
// tick. Full snapshots are fine at small player counts; delta encoding is
// a YAGNI item until profiling says otherwise.
type State struct {
	Players   []PlayerState   `json:"players"`
	NPCs      []NPCState      `json:"npcs"`
	Resources []ResourceState `json:"resources"`
}

// TalkIntent asks to talk to an NPC. The server validates proximity — the
// client can request, never assert, a conversation.
type TalkIntent struct {
	NPCID string `json:"npc_id"`
}

// Dialogue is one spoken NPC line. Tier 0 of the NPC system (proposal
// §13.4): static lines from the content file, no AI involved.
type Dialogue struct {
	NPCID   string `json:"npc_id"`
	NPCName string `json:"npc_name"`
	Line    string `json:"line"`
}

// GatherIntent asks to gather a resource node. Like TalkIntent, the server
// validates proximity and node availability — the client requests, never
// asserts, a successful gather.
type GatherIntent struct {
	NodeID string `json:"node_id"`
}

// Inventory is sent after a successful gather: the item that changed and its
// new authoritative total. The client never computes its own totals.
type Inventory struct {
	ItemType string `json:"item_type"`
	Quantity int    `json:"quantity"`
}

// Error reports a protocol-level failure to the client.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
