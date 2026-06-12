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

### `move_intent`

The held movement direction, sent whenever it changes (including release →
zero). Components are -1..1; the server clamps the vector to unit length
and integrates position itself at its own tick rate and speed — the client
can never state a position, distance, or speed.

```json
{ "type": "move_intent", "data": { "dx": 1.0, "dy": -0.5 } }
```

## Server → Client

### `welcome`

Reply to a valid `hello`. The session is live after this. Includes the
spawn position and world bounds.

```json
{
  "type": "welcome",
  "data": {
    "player_id": "f3a91c0d22b04e1f",
    "server_time": "2026-06-11T17:00:00Z",
    "motd": "No flag, no fortune. Welcome to the cove.",
    "spawn_x": 512.0,
    "spawn_y": 304.0,
    "world_w": 1280.0,
    "world_h": 720.0
  }
}
```

### `state`

The authoritative world snapshot, broadcast to every connected player at
20Hz. Full snapshots (not deltas) — fine at small player counts. Clients
render by interpolating toward these positions and must remove any player
absent from the snapshot.

```json
{
  "type": "state",
  "data": {
    "players": [
      { "id": "f3a91c0d22b04e1f", "name": "anne", "x": 520.4, "y": 304.0 },
      { "id": "8c2e51b09a77d3e0", "name": "bart", "x": 700.0, "y": 412.8 }
    ]
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
