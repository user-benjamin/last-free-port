# CLAUDE.md

Orientation for an AI agent contributing to **The Last Free Port** (codename
Project Corsair). Read this first, then the doc it points you to for the task at
hand. Keep this file short; put detail in the linked docs and keep those in sync.

## What this is

A grounded, no-magic pirate survival / base-building **multiplayer** game. It's a
backend-showcase hobby/portfolio project: the value is in the
**server-authoritative game logic, persistent world, and infrastructure**, not
in art volume. A Godot 4 client renders; all rules live in Go on the server.

- **What the game is:** [docs/proposal.md](docs/proposal.md) (full vision)
- **Idea backlog / near-term shape:** [docs/wishlist.md](docs/wishlist.md)
- **Committed milestones:** [README.md](README.md#development-priorities)
- **Daily dev workflow:** [docs/development.md](docs/development.md) ← read before running anything
- **Client/server message contract:** [docs/protocol.md](docs/protocol.md)

## The non-negotiable rules

Break these and the change is wrong no matter how well it works:

1. **The client sends *intent*, never truth.** If you find yourself writing
   "player gains X" / "player moves to Y" logic in GDScript, it belongs in
   `server/`. The client says "I want to gather node N"; the server decides.
2. **The server validates everything important** — proximity, ownership,
   inventory math, availability. Assume the client is hostile or buggy.
3. **Postgres is the only source of truth.** Anything a player would be angry to
   lose lives there. **Valkey is ephemeral** (sessions, tickets, cache) and must
   be safe to wipe at any moment.
4. **Every gameplay feature starts as a message type** in
   [docs/protocol.md](docs/protocol.md) *and*
   `server/internal/protocol/protocol.go` — keep the two in sync.
5. **Story and content are data, not code.** NPC lines, resource nodes, and
   (soon) recipes live in JSON (`server/internal/game/*.json`), validated at
   startup. Don't hardcode content in Go.

## Repository map

```
client/                 Godot 4 project (GDScript)
  scenes/login|game/     the two scenes; *.gd is the logic, *.tscn the layout
  autoload/session.gd    singleton that survives the login→game scene change
  assets/, tools/        generated art/audio + the scripts that make them
server/
  cmd/api/               HTTP service: accounts, login, sessions (handles passwords)
  cmd/game-server/       authoritative WebSocket simulation (20Hz), redeems tickets
  internal/api/          HTTP handlers (depend on small interfaces, tested with fakes)
  internal/auth/         users (argon2id), session tickets (Valkey GETDEL, single-use)
  internal/game/         the hub, connection handling, content loaders, *.json content
  internal/inventory/    Postgres-backed inventory store
  internal/protocol/     typed mirror of docs/protocol.md
  migrations/            numbered NNNN_*.up.sql / .down.sql pairs
deploy/                  Terraform & deploy tooling (Phase 1 — not built yet)
docs/                    proposal, wishlist, protocol, development, lore
```

## Architecture in one breath

Login is HTTP→`api`, which validates credentials and mints a **short-lived,
single-use session ticket** in Valkey. The client connects to `game-server` over
WebSocket and redeems the ticket (atomic `GETDEL`) to prove identity — the game
server never sees a password (proposal §16). From then on the client streams
*intents*; the server integrates state at 20Hz, broadcasts full snapshots, and
persists durable changes to Postgres.

## Commands you'll actually use

`make help` lists everything. The essentials:

| Command | Does |
|---|---|
| `make up` | Build + start the backend stack (postgres, valkey, api, game-server) |
| `make logs` | Follow api + game-server logs (keep this open) |
| `make play` | Launch the client (auto-imports assets first; backend must be up) |
| `make test` | `go test ./...` — run before calling a server change done |
| `make psql` | psql shell to see what actually persisted |
| `LFP_AUTOLOGIN=Ben LFP_AUTOPASSWORD=... make play` | skip the login form |

Migrations apply automatically as a one-shot `migrate` service on every
`make up`. See [docs/development.md](docs/development.md) for the three edit
loops (client = instant, assets = needs import, server = container vs native).

## Conventions for contributing

- **Trunk-based:** short-lived branch off `main`, small focused PR, `main` stays
  deployable. One concern per PR (don't fold docs into a feature PR, etc.).
- **Schema changes:** add a numbered `.up.sql`/`.down.sql` pair in
  `server/migrations/`; never edit an applied migration. Commit the `.import`
  sidecar files for assets; `.godot/` is gitignored.
- **Go style here:** interfaces are declared at the *consumer* (e.g.
  `TicketRedeemer` in `internal/game`) so packages depend on small local
  contracts and tests use fakes. Comments explain **why**, not what — match that
  density. Embedded content (`//go:embed`) is validated at load so a bad file
  fails loudly at startup / in CI, not mysteriously in game.
- **Verify against reality, don't just assert.** For server changes:
  `make test`, then exercise the live stack (curl the endpoint, watch `make
  logs` / `make psql`). For client changes: a headless run (`godot --headless
  --path client --quit-after N` with autologin) catches script/scene errors;
  rendered look and input still need a human's eyes — say so rather than
  claiming it's verified.
- Use the scratchpad / throwaway files for experiments, and **clean them up**
  (and any seeded test rows) when done.

## Scope discipline (don't over-build)

This project adds complexity only when pain proves it necessary (proposal §21).
**Deliberately NOT built yet:** Kubernetes/any orchestrator, the `npc-director`
AI service, ship combat, factions, the economy, multi-region — and the cloud
infra itself (`deploy/` is empty). Don't introduce these speculatively. When a
task tempts you toward one, prefer the smallest thing that ships the current
milestone's playable payoff.

## Current state (keep this honest as you go)

Playable locally: register/login with a password, walk a shared cove
(server-authoritative movement), talk to Silas the smuggler, gather driftwood
into a Postgres-persisted inventory with a toggle panel. WebSocket origins are
locked down. Only **driftwood** exists as a resource today — scrap iron, stone,
and crafting are designed (see [docs/wishlist.md](docs/wishlist.md)) but not yet
built.
