# The Last Free Port

A grounded, no-magic pirate survival/base-building multiplayer game.
Codename: **Project Corsair**. Hobby, learning, and portfolio project — the
showcase is the backend: server-authoritative game logic, persistent world
state, Terraform-managed infrastructure, and AI-assisted NPC dialogue.

Full design: [docs/proposal.md](docs/proposal.md) ·
Wire contract: [docs/protocol.md](docs/protocol.md) ·
Dev workflow: [docs/development.md](docs/development.md)

## Stack

| Layer | Choice |
|---|---|
| Client | Godot 4 (GDScript) — exports desktop and browser from one project |
| Backend | Go — `api`, `game-server` (later `npc-director`, `worker`) |
| Protocol | WebSocket + JSON envelopes |
| Data | Postgres (source of truth), Valkey (cache/sessions) |
| Runtime | Docker Compose locally → Terraform-managed Hetzner VPS (Phase 1) |

## Quickstart

Backend:

```sh
docker compose up --build
# api:         http://127.0.0.1:8080/healthz
# game-server: http://127.0.0.1:8081/healthz, ws://127.0.0.1:8081/ws
```

Client: open `client/` in Godot 4.3+ and press Play. You should see the
hello/welcome handshake succeed with a message of the day.

Without Godot:

```sh
curl -s -X POST http://127.0.0.1:8080/v1/login -d '{"username":"ben"}'
```

Tests:

```sh
cd server && go test ./...
```

## Layout

```
client/   Godot project
server/   Go services (cmd/api, cmd/game-server) + shared internal packages
deploy/   Terraform and deployment tooling (Phase 1)
docs/     Proposal, protocol, future ADRs
```

## Roadmap

Phase 0 (current): local Compose stack, hello/welcome handshake, login stub.
See proposal §20 for the full phase plan and §21 for the YAGNI guardrails.
