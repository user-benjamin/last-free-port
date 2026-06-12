# Writing for The Last Free Port

This is the guide for contributing story: NPC dialogue, rumors, names, and
worldbuilding. No programming required — you edit text files, open a pull
request, and your words come out of a character's mouth in the game.

## The world in three sentences

It's the twilight of the pirate republic in a heightened Caribbean. A
Spanish treasure fleet has wrecked, everyone is scrambling to get rich
before the Crown reclaims the region, and pardons are rumored. Nobody is
the chosen one — everyone is just trying to carve out a future before the
noose tightens.

Full setting: [../proposal.md](../proposal.md) — especially §5 (world
premise), §6–7 (story structure), and §10 (factions).

## The rules of the world (read before writing)

From proposal §4 — these are hard boundaries, not suggestions:

1. **No magic, ever.** Superstition, rumors, fake ghost ships, plague
   ships, lantern tricks: yes. Actual ghosts, curses that literally work,
   sea monsters: no. A good test: every spooky story must have a possible
   mundane explanation.
2. **Grounded, mature, but not shock content.** Violence, betrayal,
   disease, drunken danger exist. Sexual violence, cruelty-as-spectacle,
   and misery-as-content do not.
3. **No slavery content of any kind.** Not as backstory, not as a rumor,
   not as flavor. This is a deliberate creative boundary (proposal §4.3).
4. **No modern slang.** It doesn't need to be period-perfect English, but
   it should never sound like the internet.

## How to add dialogue

NPC dialogue lives in [`server/internal/game/npcs.json`](../../server/internal/game/npcs.json).
Each NPC has a `lines` pool; talking to them returns **one random line per
conversation**, so every line you add is new content a player can
encounter.

1. Edit the file on a branch (never directly on `main` — it's protected
   and will refuse you anyway)
2. Add lines to an existing NPC's `"lines"` array — mind the commas, it's
   JSON and the build will fail loudly if it's malformed (that's CI doing
   its job, not you breaking things)
3. Open a pull request; CI runs the validators; Ben reviews and merges
4. Next time the server restarts, your words are in the game

A good line is self-contained (no follow-up questions possible yet),
hints at something a player might want to *do*, and sounds like a person —
Silas isn't a quest dispenser, he's a smuggler who talks too much.

## What lives where

- `server/internal/game/npcs.json` — NPC definitions and dialogue (in the game)
- `docs/lore/` (this folder) — longer worldbuilding: faction notes,
  character bios, place names, rumor drafts. Markdown, no format rules.
  This material later becomes the curated context the AI NPC system feeds
  on (proposal §13), so nothing written here is wasted.

## Current cast

- **Silas Crane** — smuggler at the dock. Knows everything happening in
  the cove and shares it freely, which should worry you.
