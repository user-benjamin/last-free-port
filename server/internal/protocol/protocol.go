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
	TypeHello   = "hello"
	TypeWelcome = "welcome"
	TypeError   = "error"
)

// Hello is the first message a client sends after connecting.
// It will be replaced by a join message carrying an API-issued session
// ticket once the auth flow lands (proposal §16).
type Hello struct {
	Name string `json:"name"`
}

// Welcome is the server's reply to a valid Hello.
type Welcome struct {
	PlayerID   string `json:"player_id"`
	ServerTime string `json:"server_time"`
	Motd       string `json:"motd"`
}

// Error reports a protocol-level failure to the client.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
