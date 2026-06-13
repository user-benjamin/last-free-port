# Development Workflow

The daily-driver guide. For what the game *is*, see [proposal.md](proposal.md);
for the client/server message contract, see [protocol.md](protocol.md).

## The setup

Two terminals, always:

```sh
# Terminal 1 — the server's point of view
make logs

# Terminal 2 — the player's point of view
make play
```

Every client action should produce a server log line in terminal 1. When
something breaks, the first question is "did the server see it?" — this
layout answers it instantly. Run `make play` twice for two players.

`make help` lists every command.

## Which loop am I in?

There are three edit loops with different speeds. Know which one you're in:

### 1. Client code (GDScript) — instant

Edit `.gd`/`.tscn` files, run `make play` again (or F5 in the editor).
No build step. `print()` output goes to the terminal (or the editor's
Output panel). Dev shortcut: `LFP_AUTOLOGIN=ben make play` skips the
login form.

### 2. Assets (images, audio) — needs an import

**Gotcha:** Godot never uses your asset file directly. It converts each
asset into an optimized format cached in `client/.godot/imported/` and the
game loads *that*. The cache refreshes when:

- the **editor** is open and sees the file change (automatic), or
- something runs `godot --headless --path client --import`.

`make play` runs the import step automatically, so the normal flow is just:
replace the file, `make play`. If art still looks stale, you're probably
running the game from an open editor that hasn't regained focus — click the
editor window once so it rescans, then run.

Per-asset import settings live in the `.import` sidecar files next to each
asset — **commit those**. The `.godot/` cache directory is gitignored.

GIFs don't work in Godot at all — see [../client/assets/README.md](../client/assets/README.md).

### 3. Server code (Go) — two gears

Honest gear (containers, same as prod):

```sh
make up      # rebuilds changed images (~15-30s) and restarts
```

Fast gear (native, ~1s rebuilds) for when you're iterating hard:

```sh
docker compose stop game-server   # free port 8081
make dev-server                   # Ctrl+C, edit, re-run
make up                           # back to containers when done
```

Postgres and Valkey always stay in containers. Database migrations run
automatically as a one-shot `migrate` service every `make up` — to add
schema, drop a numbered `.up.sql`/`.down.sql` pair in `server/migrations/`
and `make up` applies it. Run `make test` before
considering a server change done.

## Watching state

| Window | Command | Shows |
|---|---|---|
| Logs | `make logs` | structured slog output from api + game-server |
| Database | `make psql` | what actually got persisted |
| Containers | `docker compose ps` | what's running and healthy |

## Rules that keep the project honest

- **Client sends intent, never truth.** If you're writing "player gains
  item" logic in GDScript, it belongs in `server/` instead.
- Every new gameplay feature starts as a message type in
  [protocol.md](protocol.md) and `server/internal/protocol/protocol.go` —
  keep them in sync.
- Postgres is the only source of truth. Valkey contents must be safe to
  lose at any moment.

## When things look wrong

| Symptom | Likely cause | Fix |
|---|---|---|
| New/changed art doesn't show | stale import cache | `make play` (auto-imports) or focus the editor |
| "No reply from the harbormaster" on login | backend not running | `make up` |
| Client connects but nothing happens | game-server died after start | `make logs`, look at last lines |
| `connection refused` on 8081 with dev-server | container still holds the port | `docker compose stop game-server` |
| Docker errors mentioning the daemon socket | Docker Desktop not running | `open -a Docker`, wait, retry |
