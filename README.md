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

## Status

Playable today, locally: log in, walk a shared cove with other players
(server-authoritative movement at 20Hz), and talk to Silas Crane, the
smuggler at the dock. Generated 2D art and ambient sea audio. Postgres,
Valkey, and migrations run as part of the Compose stack.

## Development priorities

In order. Each milestone is independently shippable; dates are targets for
a part-time project, not promises.

1. **The cove remembers you** (~late Jun) — gatherable driftwood, inventory
   persisted in Postgres against your stable user id. First playtest demo.
2. **Lock the doors** (~early Jul) — argon2id passwords and WebSocket
   origin lockdown, deliberately *before* any public endpoint exists.
   Crafting v1 rides along.
3. **Remote play** (~mid Jul) — Phase 1 infrastructure: Terraform-managed
   Hetzner VPS, Caddy TLS, GHCR images built by CI, off-machine backups
   with a tested restore. A friend connects from their own house.
4. **Base building v1** (~early Aug) — first placeable, persistent objects.
   Needs a design pass before code.

Deliberately not being built yet (proposal §21): the AI npc-director,
minigames, ship combat, factions, and anything Kubernetes-shaped.
Complexity is added only when pain proves it necessary.

### Working principles

- Trunk-based: short-lived branches, small PRs, `main` always deployable
- The client sends *intent*, never truth — all rules live in `server/`
- Every gameplay feature starts as a message type in [docs/protocol.md](docs/protocol.md)
- Every backend milestone ships alongside a visible, playable payoff
- Story and dialogue are data, not code — see [docs/lore/README.md](docs/lore/README.md)
  to contribute writing without touching Go or Godot
