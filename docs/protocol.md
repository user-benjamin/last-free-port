# Wire Protocol v0

Transport: WebSocket, text frames, JSON payloads.
Endpoint: `ws://<game-server>/ws` (port 8081 in dev).

This document is the contract between the Godot client and the Go game
server. `server/internal/protocol/protocol.go` is the typed version — keep
the two in sync.

## Envelope

Every message in both directions:

```json
{ "type": "<message type>", "data": { } }
```

## Principles

- The client sends **intent**, never truth. The server is the only authority.
- Unknown message types are logged and ignored on both sides, so the
  protocol can grow without breaking old clients mid-session.
- Handshake messages have a 10-second server-side timeout.

## Client → Server

### `hello`

First message after connecting. Phase 0 placeholder — will be replaced by a
`join` message carrying a short-lived session ticket issued by the API
(proposal §16).

```json
{ "type": "hello", "data": { "name": "deckhand" } }
```

## Server → Client

### `welcome`

Reply to a valid `hello`. The session is live after this.

```json
{
  "type": "welcome",
  "data": {
    "player_id": "f3a91c0d22b04e1f",
    "server_time": "2026-06-11T17:00:00Z",
    "motd": "No flag, no fortune. Welcome to the cove."
  }
}
```

### `error`

Protocol-level failure. The server closes the connection after sending.

```json
{
  "type": "error",
  "data": { "code": "expected_hello", "message": "first message must be hello" }
}
```

Current codes: `expected_hello`, `bad_hello`.
