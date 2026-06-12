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
	TypeHello      = "hello"
	TypeWelcome    = "welcome"
	TypeError      = "error"
	TypeMoveIntent = "move_intent"
	TypeState      = "state"
)

// Hello is the first message a client sends after connecting.
// It will be replaced by a join message carrying an API-issued session
// ticket once the auth flow lands (proposal §16).
type Hello struct {
	Name string `json:"name"`
}

// Welcome is the server's reply to a valid Hello.
type Welcome struct {
	PlayerID   string  `json:"player_id"`
	ServerTime string  `json:"server_time"`
	Motd       string  `json:"motd"`
	SpawnX     float64 `json:"spawn_x"`
	SpawnY     float64 `json:"spawn_y"`
	WorldW     float64 `json:"world_w"`
	WorldH     float64 `json:"world_h"`
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

// State is the world snapshot broadcast to every connected client each
// tick. Full snapshots are fine at small player counts; delta encoding is
// a YAGNI item until profiling says otherwise.
type State struct {
	Players []PlayerState `json:"players"`
}

// Error reports a protocol-level failure to the client.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
