# Project Proposal — The Last Free Port

> A grounded pirate survival / base-building multiplayer game.
> Codename **Project Corsair**.

This is the full design vision. For *committed* near-term work see the
[README priorities](../README.md#development-priorities); for the *idea backlog*
see the [wishlist](wishlist.md); for the live client/server contract see the
[wire protocol](protocol.md).

## Contents

1. [Working Title](#1-working-title)
2. [One-Sentence Pitch](#2-one-sentence-pitch)
3. [Project Purpose](#3-project-purpose)
4. [Creative Direction](#4-creative-direction)
5. [World Premise](#5-world-premise)
6. [Story Structure](#6-story-structure)
7. [Campaign Acts](#7-campaign-acts)
8. [Core Player Motivations](#8-core-player-motivations)
9. [Core Gameplay Loop](#9-core-gameplay-loop)
10. [World Systems](#10-world-systems)
11. [Core Game Pillars](#11-core-game-pillars)
12. [Minigame and Skill Challenge System](#12-minigame-and-skill-challenge-system)
13. [AI-Aware NPC System](#13-ai-aware-npc-system)
14. [Technical Architecture](#14-technical-architecture)
15. [Database and Persistence](#15-database-and-persistence)
16. [Authentication and Login](#16-authentication-and-login)
17. [Mature Content and Gore](#17-mature-content-and-gore)
18. [MVP Scope](#18-mvp-scope)
19. [Future Mechanics Reserved](#19-future-mechanics-reserved)
20. [Development Roadmap](#20-development-roadmap)
21. [YAGNI Guardrails](#21-yagni-guardrails)
22. [Technical Risks](#22-technical-risks)
23. [Resume / Portfolio Value](#23-resume--portfolio-value)
24. [Success Criteria](#24-success-criteria)
25. [Summary](#25-summary)

---

## 1. Working Title

| | |
|---|---|
| **Working title** | The Last Free Port |
| **Codename** | Project Corsair |
| **Genre** | Pirate survival, crafting, base-building, exploration, and multiplayer adventure |
| **Tone** | Mature, grounded, larger-than-life pirate fiction |
| **Magic** | None |
| **Primary goal** | Hobby project, learning project, and portfolio/résumé showcase |
| **Commercial goal** | Optional later consideration, not the main driver |

## 2. One-Sentence Pitch

A grounded pirate survival/base-building game where players build hideouts,
chase treasure, upgrade ships, fight rival captains, interact with AI-aware
NPCs, and try to become dangerous enough that the world remembers them.

## 3. Project Purpose

This is first and foremost a **hobby and learning project**.

The goal is not to become an independent game studio or build a full commercial
MMO. The goal is to create a compelling, technically impressive vertical slice
that demonstrates:

- multiplayer backend design
- server-authoritative game logic
- persistent world state
- Terraform-managed infrastructure
- containerized deployment
- Postgres persistence
- Valkey caching
- AI-assisted NPC interaction
- base-building systems
- crafting systems
- gameplay architecture
- scope control

If the project looks good enough to show off publicly, it can become a
portfolio piece, demo, landing page, or eventually a Steam page. But the project
should **not** be planned around "going pro" or chasing indie-game lightning.

## 4. Creative Direction

### 4.1 Historical Fantasy, Not Supernatural Fantasy

The game should be grounded in the Golden Age of Piracy, but it is not a strict
historical simulator. The world should feel like **heightened historical
fiction**: epic battles, larger-than-life captains, rare powerful weapons,
dangerous storms, treasure fleets, hidden coves, smuggler ports, naval
crackdowns, duels, ship combat, rival captains, faction betrayals, infamous
taverns, coded maps, and wild rumors.

There is **no real magic**.

| Allowed | Not allowed |
|---|---|
| superstition | spellcasting |
| sailor rumors | cursed gold that is *literally* cursed |
| fake ghost ships | real ghosts |
| legendary weapons | immortal captains |
| psychological horror | sea monsters with magic powers |
| plague ships | chosen-one prophecy |
| lantern tricks | supernatural resurrection |
| mysterious wrecks | |
| larger-than-life reputations | |

A "ghost ship" can exist as a rumor, deception, plague vessel, derelict,
smuggler trick, or psychological terror — but the game should **never confirm
supernatural magic**.

### 4.2 Mature But Not Degenerate

The game is intended for adults or mature players, but not as a shock-content
simulator.

**The game may include:** blood, injury, brutal pirate combat, executions
implied by the world, naval violence, wildlife attacks, disease, betrayal,
drunken tavern danger, blackmail, murder, gore settings, sex work / consensual
relations.

**The game should avoid:** sexual violence, child-involved violence, hate-based
violence, torture spectacle, misery-as-content, gratuitous cruelty for its own
sake.

### 4.3 Slavery and Slave Trading Exclusion

This project will **not** include slavery as a gameplay system, player role,
economy, quest path, faction mechanic, or interactive feature. Slave trading and
related systems are intentionally omitted entirely.

This is a deliberate creative boundary, not an attempt to comprehensively
simulate every historical reality of the period. The goal is pirate adventure,
not a sandbox where players can engage in atrocities.

**Design rule:**

- No player slave trading.
- No slave-market mechanics.
- No human beings as inventory, cargo, loot, or trade goods.
- No questline where the player profits from slavery.
- No faction path built around slavery.

The game world may be historically inspired, but it will be selectively framed
around piracy, treasure, ships, survival, factions, reputation, and
player-driven adventure.

## 5. World Premise

The game takes place in a heightened version of the Caribbean during the
twilight of the pirate republic.

A great treasure fleet has wrecked. Silver, gold, weapons, charts, and royal
secrets are scattered across reefs, islands, wrecks, and hidden coves. Nassau
and the surrounding islands have become a haven for pirates, smugglers,
wreckers, ex-privateers, merchants, deserters, and opportunists. Everyone is
trying to get rich before order returns.

But the age of freebooters is ending. The Crown is preparing to reclaim the
region. Pardons are rumored. Informants are everywhere. Some captains want to
take the pardon. Some want to fight. Some want to betray their rivals. Some want
to vanish with enough treasure to retire.

The player is **not the chosen one**. The player is one more desperate sailor
trying to carve out a future.

> **The core question of the world:** Can you build enough wealth, reputation,
> allies, and firepower before the noose tightens?

## 6. Story Structure

The game should not use a traditional "save the world" main quest. Instead, it
uses a **campaign spine**:

> **The Pardon is Coming.**

The world has a direction, but the player chooses how to respond. Possible
player paths:

- become rich and retire
- build a hidden pirate haven
- take the Crown's pardon
- pretend to take the pardon and betray the Crown
- become a privateer
- hunt rival pirates
- defend Nassau
- abandon Nassau
- build a fortified cove
- control trade routes
- become an infamous captain

The "main quest" should be more of a **world-pressure system** than a linear
campaign.

## 7. Campaign Acts

### Act I — No Flag, No Fortune

The player starts with almost nothing. **Possible opening:** the player arrives
at a remote Bahamian cove with a damaged boat, little money, and a rumor that
Spanish treasure has washed into nearby waters.

Early goals: find food · repair boat · find weapon · earn first coin · meet
local smuggler · learn basic crafting · place first base objects · survive first
fight · discover first treasure lead.

*Act I ends when the player establishes a hideout or obtains a functional ship.*

### Act II — The Wreck Season

The world opens up. Everyone is chasing treasure, maps, cannons, salvage rights,
secret routes, and political leverage.

Midgame goals: salvage wrecks · raid camps · upgrade ship · recruit crew · craft
weapons · build outpost · earn reputation · make faction choices · fight rival
captains · discover rare items.

*Act II ends when the player is known enough that major factions and rival
captains react to them.*

### Act III — The King's Pardon

The Crown arrives. The world begins to change.

Late-game choices: accept pardon · reject pardon · betray other pirates · fight
the Crown · flee to a hidden cove · become a privateer · rob one final convoy ·
found a free port.

For an MMO-lite game, this does not require a hard ending. It can become a
recurring world event, seasonal pressure, or late-game state.

## 8. Core Player Motivations

Players care about treasure, but treasure is not the real end goal. **Treasure
is fuel.** Players want booty because it buys ship upgrades, better cannons,
crew loyalty, base expansion, bribes, rare maps, weapons, tools, supplies,
reputation, political influence, safe harbor, and freedom.

The game should support multiple overlapping motivations:

| Immediate goals | Midgame goals | Long-term goals |
|---|---|---|
| survive | build a hideout | become infamous |
| eat | upgrade ship | build a free port |
| repair | hire crew | retire wealthy |
| arm yourself | craft better gear | fight the Crown |
| get powder | control a cove | become legitimate |
| get coin | rob a convoy | dominate trade routes |
| avoid patrols | make allies | defeat rival captains |
| find safe harbor | gain reputation | leave a permanent mark on the world |

## 9. Core Gameplay Loop

```
Hear rumor or take contract
        ──►
Prepare ship, supplies, crew, weapons
        ──►
Sail to island, wreck, convoy, port, or reef
        ──►
Fight, sneak, salvage, negotiate, duel, trade, or flee
        ──►
Return with loot, scars, maps, enemies, or allies
        ──►
Upgrade ship, base, crew, gear, and reputation
        ──►
World reacts  ──► (back to the top)
```

This loop supports: base building · crafting · ship ownership · treasure
hunting · AI NPCs · factions · rare item recognition · rival captains ·
minigames · reputation · combat · exploration.

## 10. World Systems

### 10.1 Factions

Factions make the world reactive. Initial faction candidates:

- Pirate Commons
- Crown Authorities
- Spanish Salvage Interests
- Merchants and Factors
- Privateers
- Smugglers
- Local Wreckers
- Independent Captains

Each faction should have: reputation · hostility level · trade access · rumor
access · quest access · territory influence · preferred contraband · rival
factions.

### 10.2 Reputation

Reputation should be **local and faction-specific**, not a simple good/evil
meter.

- Pirates respect bold raids.
- Merchants value reliability.
- The Crown values order and the betrayal of pirates.
- Smugglers value discretion.
- Rival captains remember insults.
- Ports remember violence.

NPCs should react to: rare gear · ship name · known victories · known betrayals
· faction reputation · wanted level · prior encounters · crew reputation · duel
outcomes.

### 10.3 Heat / Wanted System

Instead of a simple crime meter, use **"heat."**

- **High Crown heat:** more patrols, fewer legal merchants, higher bounty,
  stronger pirate reputation.
- **High Spanish heat:** fortified salvage camps, hunters sent after the player,
  better black-market prices for stolen Spanish goods.
- **High pirate betrayal:** taverns become dangerous, rivals raid the player
  base, crew morale drops, pirate NPCs refuse trade.

### 10.4 Rival Captains

Rival captains are the seed of the future **nemesis system**. They should be
able to: remember defeats · carry scars · steal contracts · mock the player in
taverns · raid the player hideout · challenge the player to a duel · hunt the
same treasure · accept pardons · betray allies · form alliances · spread rumors.

This creates story without magic.

## 11. Core Game Pillars

### 11.1 Exploration

Players explore islands, reefs, shipwrecks, hidden coves, ports, jungle trails,
caves, forts, salvage camps, smuggler routes, storms, and dangerous waters.

Exploration should produce: resources · maps · rumors · treasure · rare items ·
new NPCs · faction opportunities · rival encounters · base locations.

### 11.2 Base Building

Inspired by Fallout 4-style object placement. Players can build: camps ·
hideouts · docks · workshops · storage · watchtowers · walls · gates · cannon
platforms · taverns · crew quarters · shipyards · warehouses · defensive
structures · decorations.

Base building should be **persistent and server-authoritative**.

### 11.3 Crafting

Players gather resources and craft: tools · weapons · ammo · ship parts · base
structures · food · medicine · clothing · armor · trade goods · repairs · rare
upgrades.

Crafting should be **transactionally validated by the backend**. The client
*requests* crafting; the server *decides* whether it succeeds.

### 11.4 Ship Ownership

Ships are central to the fantasy.

- **Early ship systems:** basic boat ownership, ship inventory, repairs,
  upgrades, docking, sailing, crew slots, damage state.
- **Future ship systems:** ship combat, boarding, cannon minigames, crew
  morale, wind/sail management, ship classes, cargo capacity, smuggling
  compartments, named ships, reputation tied to ship.

### 11.5 Combat

Combat should be **grounded and dangerous**. Possible combat types: melee ·
pistols · muskets · boarding fights · duels · shipboard combat · cannon fire ·
ambushes · wildlife attacks · base raids.

The game can include gore, but gore should be configurable and should not define
the entire game.

## 12. Minigame and Skill Challenge System

Minigames are a long-term design pillar, but **not a v1 requirement**. Their
purpose is to make repeated pirate actions feel tactile and skillful.

Good candidate minigames: cannon reload · cannon aiming · ship repair ·
lockpicking · treasure extraction · fishing · sail trimming · dueling flourishes
· crafting quality · field medicine · blacksmithing · gambling.

### 12.1 Cannon Reload Example

Instead of `Press F → wait → Press F to fire`, use a short skill challenge:
**swab barrel → load powder → load shot → ram charge → aim → fire timing.**

Performance can affect: reload speed · accuracy · damage · misfire chance ·
critical hit chance · cannon overheating · crew morale.

> Poor performance should usually make the action *worse*, not *impossible*.
> Good performance should create a **bounded** advantage.

### 12.2 Generic Minigame Framework

Do not hardcode every minigame separately. Each minigame should define:

- `minigame_id`
- difficulty
- duration window
- allowed score range
- possible grades
- result modifiers
- server validation rules
- accessibility options

**Grades:** `fail` · `poor` · `normal` · `good` · `great` · `perfect`

**Server validation:** player is allowed to perform the action · the minigame
was started by the server · the duration is plausible · the score is within
bounds · the state is still valid · modifiers are capped.

> The client performs the interaction. The server validates and applies the
> result.

### 12.3 Accessibility

Minigames should include options such as: minigame assist · reduced timing
difficulty · auto-complete with baseline result · hold instead of mash · disable
rapid inputs · colorblind-safe UI · reduced screen shake.

Minigames should reward skill without making the game tedious or inaccessible.

## 13. AI-Aware NPC System

AI NPC interaction is a major differentiator, but not required for the earliest
v1. The goal is not fully autonomous NPCs — the goal is that **NPCs feel
observant, contextual, and reactive.**

NPCs should notice: a rare item worn by the player · player reputation · ship
name · a prior encounter · a known betrayal · faction standing · a recent
victory · a duel outcome · wanted level · an unusual weapon · crew reputation.

**Example.** A player approaches a smuggler while wearing a legendary Black
Kraken coat. The NPC says:

> "That coat does not belong on a living man. Black Kraken leather, Port Mercy
> cut. Either you killed someone important, or someone important is looking for
> you."

### 13.1 NPC Director Service

AI should be handled by a separate service: **`npc-director`**.

Responsibilities: build curated NPC context · retrieve relevant player facts ·
retrieve NPC memory · call the external LLM API · validate the structured
response · return dialogue and intent · write approved memories · cache repeated
interactions.

### 13.2 AI Design Rule

The model should generate **flavor, dialogue, tone, and suggested intent**. The
model should **not directly control game state.**

| Allowed AI output | Not allowed directly from AI |
|---|---|
| dialogue | grant item |
| tone | remove item |
| recognized facts | spawn enemy |
| suggested intent | start quest |
| memory write candidate | award money |
| relationship delta suggestion | change faction state |
| | damage player |
| | modify inventory |

> All game actions must be validated by the server.

### 13.3 NPC Memory

NPCs may remember compact facts — e.g. *"Mara Voss remembers that the player
wore the Black Kraken coat and refused to sell it."*

Do **not** store every line of dialogue forever. Store: a recent transcript
(briefly) · long-term memory summaries · important player facts · faction state
· relationship scores.

### 13.4 Cost Control

Use AI selectively. Interaction tiers:

| Tier | Behavior |
|---|---|
| **Tier 0** | static lines |
| **Tier 1** | template lines |
| **Tier 2** | cached AI lines |
| **Tier 3** | live AI call |

Live AI should be reserved for: rare item recognition · important NPCs · rival
captains · quest conversations · duel challenges · faction negotiations ·
nemesis interactions.

## 14. Technical Architecture

The game starts as a simple containerized multiplayer application on one VPS.
This is mostly a 3-tier web application with one extra component: an
authoritative game server.

### 14.0 Locked Stack Decisions (2026-06-11)

- **Presentation — 2D top-down** (Stardew/Core Keeper shape). Chosen over 3D
  low-poly for time-to-playable, solo art scope, and browser export. The server
  simulation is dimension-blind (coordinates only), so this is a client-side
  decision. Base building becomes grid/tile placement, which fits the
  server-authoritative object model unchanged.
- **Client engine — Godot 4** (standard build, GDScript). Free and open source,
  lightweight, fast solo iteration. Exports desktop *and* browser builds from
  the same project, which defers the distribution decision. No built-in
  replication framework competing with the custom authoritative backend.
- **Server language — Go** for all backend services (`api`, `game-server`,
  `npc-director`, `worker`). Aligns with the platform-engineering ecosystem and
  career target; strong fit for concurrent network services; one language across
  the entire backend.
- **Wire protocol — WebSocket carrying JSON.** Works natively in both Godot
  export targets (desktop and browser). UDP is YAGNI at small session sizes;
  revisit only if latency pain proves it. Message types are documented in
  [docs/protocol.md](protocol.md) as the client/server contract.
- **Repo layout — monorepo.** `client/` (Godot project), `server/` (Go
  services), `deploy/` (Compose now, Terraform later), `docs/`.
- **Sequencing — local-first.** Docker Compose from commit one; the Terraform
  VPS is deferred until a friend wants to connect (Phase 1).

### 14.1 Initial Hosting Decision

- **Primary VPS provider:** Hetzner Cloud — Location: Ashburn, Virginia.
  - close enough to Charlotte / East Coast testing
  - cheap
  - Terraform-friendly
  - simple VPS model
  - avoids the AWS/EKS/GKE platform tax
- **Fallback provider:** DigitalOcean.

### 14.2 Initial Runtime Stack

VPS · Ubuntu · Docker · Docker Compose · Caddy · API service · Game server
service · NPC Director service · Worker service · Postgres · Valkey ·
off-machine backups · Cloudflare DNS.

### 14.3 High-Level Architecture

```
Game Client
  │  HTTPS
  ▼
Caddy / Reverse Proxy
  │
  ├──► API Server
  │       ├──► Postgres
  │       ├──► Valkey
  │       └──► NPC Director
  │
  └──► Optional Admin UI

Game Client
  │  WebSocket game protocol (TCP today; UDP only if proven necessary)
  ▼
Authoritative Game Server
  ├──► Postgres
  ├──► Valkey
  └──► API / NPC Director as needed
```

### 14.4 Services

| Service | Responsibility |
|---|---|
| `caddy` | HTTPS, routing, TLS |
| `api` | login, accounts, inventory, crafting, base persistence, sessions |
| `game-server` | live multiplayer simulation, movement, combat, validation |
| `npc-director` | AI-assisted NPC dialogue and memory |
| `worker` | background jobs, backups, cleanup, world ticks |
| `postgres` | source of truth |
| `valkey` | cache, sessions, rate limits, presence, temporary state |

## 15. Database and Persistence

### 15.1 Postgres

Postgres is the **source of truth**. Stores: `users` · `characters` · `ships` ·
`inventories` · `inventory_items` · `bases` · `base_objects` ·
`crafting_recipes` · `crafting_jobs` · `world_regions` · `game_sessions` ·
`npc_profiles` · `npc_memories` · `player_notable_facts` · `item_lore` ·
`faction_reputation` · `dialogue_events`.

> Anything the player would be angry to lose belongs in Postgres.

### 15.2 Valkey

Valkey is for **temporary or cached state**.

| Good uses | Bad uses |
|---|---|
| session tickets | permanent inventory |
| dialogue cache | ship ownership |
| presence | base state |
| rate limits | currency source of truth |
| matchmaking queues | crafting authority |
| temporary locks | rare item ownership |
| hot config | |
| recent NPC interaction cache | |

### 15.3 Base Object Model

```
bases
  id
  owner_player_id
  island_id
  name
  created_at
  updated_at

base_objects
  id
  base_id
  object_type
  position_x / position_y / position_z
  rotation_x / rotation_y / rotation_z
  state_jsonb
  created_at
  updated_at
```

Flexible per-object state lives in `state_jsonb`, e.g.:

```json
{
  "health": 100,
  "locked": true,
  "fuel": 12,
  "skin": "weathered_oak"
}
```

### 15.4 Backups

Backups are **mandatory**. Minimum:

- nightly `pg_dump`
- compress the backup
- upload off-machine
- retain 7 daily, 4 weekly, 3 monthly
- test restore periodically

Potential storage: Cloudflare R2 · Backblaze B2 · S3 · Hetzner Storage Box.

> The database is the game.

## 16. Authentication and Login

The game server should **not** handle passwords.

**Login flow:**

1. Player opens game.
2. Client calls the API over HTTPS.
3. API validates credentials.
4. API returns an access token.
5. Client asks the API to join/create a game session.
6. API creates a short-lived session ticket.
7. Client connects to the game server with the ticket.
8. Game server validates the ticket.
9. Player enters the session.

**Principles:**

- passwords handled only by the API
- game server accepts short-lived tickets
- client sends intent, not truth
- server validates all important actions

## 17. Mature Content and Gore

The game may include mature pirate violence, but gore should be **modular**. The
combat system should emit neutral damage events; a gore/VFX layer decides what
to display based on user settings.

**Settings:** Gore Off · Reduced Gore · Full Gore · Streamer Mode · Blood Decals
On/Off · Corpse Persistence Short/Medium/Long · Screen Blood On/Off.

**Technical principle:** combat logic is separate from gore presentation.

This supports: accessibility · streaming · rating control · regional builds ·
performance · player preference.

## 18. MVP Scope

The first playable version should be small.

### 18.1 MVP Goal

A player can: create an account · log in · create a pirate character · join a
test island/cove · collect resources · place base objects · craft one item · own
or repair a small boat · talk to one NPC · persist state after logout · join the
same session as another player.

### 18.2 MVP Story Slice

The player arrives at a remote cove with a damaged boat and a rumor: a Spanish
strongbox from a wreck has washed into nearby waters.

- A local smuggler will help, but wants a cut.
- A rival captain heard the same rumor.
- A Crown informant is watching the harbor.
- The player needs supplies, a weapon, a place to hide loot, and a way out.

### 18.3 MVP World

Small but complete: one cove · one dock · one small settlement · one nearby
wreck · one base-building area · one NPC smuggler · one rival captain · one rare
item · one craftable item · one boat · one hostile encounter.

## 19. Future Mechanics Reserved

Important to the long-term vision, but should not block MVP.

### 19.1 Ship Combat

cannons · boarding · hull damage · fire · repairs · crew assignments · sail
positioning · ammunition types · ship classes · cargo loss · capturing ships.

### 19.2 Duels

tavern duels · captain duels · honor challenges · wagered duels · non-lethal
duels · reputation consequences · AI rival challenges.

### 19.3 Wildlife

dangerous animals · huntable animals · food sources · rare hides · sharks ·
crocodiles · jungle predators · base threats · environmental encounters.

### 19.4 Nemesis System

named rival captains · scars from prior fights · remembered insults · revenge
raids · promotion after survival · captain alliances · betrayal memory ·
fear/respect scores · rival captains taking pardons.

> **Design rule:** the nemesis system should create grounded rivalries, not
> supernatural resurrection.

### 19.5 Economy

trade goods · regional prices · contraband · smuggling · black-market merchants
· convoys · supply shortages · bribes · ship repair markets · player-to-player
trade.

### 19.6 Factions and Territory

ports changing control · faction patrols · pirate havens · fortified coves ·
trade route influence · local reputation · base raids · pardon politics.

## 20. Development Roadmap

> This is the *original* phase plan. Live, committed milestones are tracked in
> the [README](../README.md#development-priorities); near-term ideas in the
> [wishlist](wishlist.md).

### Phase 0 — Technical Spike
**Goal:** prove the stack works.
Local Docker Compose · API service · game server service · Postgres · Valkey ·
basic client connection · basic login stub · basic object persistence.

### Phase 1 — Online Prototype
**Goal:** deploy the first remote environment.
Terraform VPS · Cloudflare DNS · Caddy HTTPS · Docker Compose deployment ·
Postgres volume · off-machine backups · remote game-server connection · a friend
can connect.

### Phase 2 — Core Gameplay Loop
**Goal:** make the game playable.
resource gathering · inventory · crafting · base placement · base persistence ·
small boat ownership · basic multiplayer synchronization · basic hostile
encounter · admin/debug tools.

### Phase 3 — World Personality
**Goal:** make the world feel alive.
NPC smuggler · rare item recognition prototype · faction reputation · rumor
system · rival captain · simple AI NPC Director integration · world event logs.

### Phase 4 — Pirate Fantasy Expansion
**Goal:** make it feel like a pirate game.
better sailing · treasure maps · ship upgrades · basic combat · loot tables ·
dock/port interactions · Crown patrol threat · more base pieces.

### Phase 5 — Skill Challenge Prototype
**Goal:** prove minigames can enhance core play.
one minigame prototype (likely cannon reload, lockpicking, or ship repair) ·
server-validated result · bounded gameplay modifiers · accessibility setting.

### Phase 6 — Advanced Systems
**Goal:** explore long-term differentiators.
ship combat · duels · wildlife · economy · nemesis system · faction conflict ·
AI memory expansion · multiple sessions · session scheduler.

## 21. YAGNI Guardrails

| Do **not** build yet | Build early |
|---|---|
| Kubernetes | clean architecture |
| EKS/GKE | database migrations |
| service mesh | basic auth/session flow |
| Kafka | server-authoritative inventory |
| multi-region | server-authoritative crafting |
| global sharding | base object persistence |
| full MMO economy | structured logs |
| complex autoscaling | off-machine backups |
| fully autonomous NPC society | Terraform provisioning |
| LLM-controlled combat | simple deploy path |
| voice generation | basic admin tools |
| advanced anti-cheat | |
| massive authored campaign | |
| dozens of minigames | |
| Steam launch plan | |

> **Decision rule:** add complexity only when pain proves it is necessary.

Examples:

- If one VPS is too small, add a second server.
- If Postgres competes with game performance, move the DB to a separate host.
- If manual sessions become painful, add a scheduler.
- If scheduling becomes painful, consider ECS, Agones, or Kubernetes.
- If AI costs grow, increase caching and templates.

## 22. Technical Risks

### 22.1 Scope Creep
**Risk:** the project tries to become a full pirate MMO before the core loop
works.
**Mitigation:** build one cove · one ship · one NPC · one rival · one base area ·
one treasure loop.

### 22.2 Multiplayer Complexity
**Risk:** networking, persistence, and synchronization consume the whole
project.
**Mitigation:** small sessions · server authority · limited player count early ·
simple world model · profile before optimizing.

### 22.3 AI Cost and Weirdness
**Risk:** NPC AI becomes expensive, inconsistent, or hallucinates mechanics.
**Mitigation:** curated context · structured outputs · cache responses ·
validate all actions · limit live AI to important interactions.

### 22.4 Persistence Bugs
**Risk:** players lose bases, items, or ships.
**Mitigation:** Postgres as source of truth · transactions for crafting/inventory
· nightly backups · restore testing · admin recovery tools.

### 22.5 Content Scope
**Risk:** the world requires too much art, animation, sound, and writing.
**Mitigation:** small vertical slice · stylized scope · reuse modular assets ·
focus on systems · avoid a giant campaign.

## 23. Resume / Portfolio Value

A successful vertical slice can demonstrate: infrastructure-as-code · container
orchestration without Kubernetes overkill · cloud deployment · auth/session
design · database modeling · server-authoritative game state · multiplayer
architecture · AI integration · cache strategy · backup strategy · game systems
design · scope management.

**Strong portfolio phrasing:**

> Built a containerized multiplayer pirate survival prototype with
> Terraform-managed VPS infrastructure, Postgres-backed persistent world state,
> Valkey caching, server-authoritative crafting/base-building systems, and
> AI-assisted NPC dialogue using curated game context.

## 24. Success Criteria

The project is successful if it produces a small but complete vertical slice.

- **Minimum success:** player can log in · join a session · gather resources ·
  craft an item · place a base object · interact with an NPC · persist progress ·
  reconnect later · play with a friend · deploy repeatably · restore from backup.
- **Strong success:** an NPC recognizes a rare player item · a rival captain
  remembers the player · a small boat can be repaired/upgraded · the world reacts
  to faction reputation · a basic rumor leads to treasure · a base persists
  across sessions · the project looks good in screenshots/video.
- **Outstanding success:** a short trailer looks compelling · friends want to
  keep playing · the architecture is clean enough to explain in interviews · AI
  NPC interaction feels meaningfully different from static dialogue · one
  minigame prototype proves skill-based interaction.

## 25. Summary

The Last Free Port is a grounded, no-magic pirate survival/base-building
multiplayer project. It should feel like heightened historical fiction:
dangerous, colorful, adult, brutal, funny, and full of larger-than-life
captains. The world is not about saving civilization — it is about treasure,
reputation, survival, betrayal, freedom, and whether the player can become
powerful enough to be remembered.

**Start small:** one VPS · Docker Compose · Caddy · API · game server · NPC
Director · worker · Postgres · Valkey · Cloudflare DNS · off-machine backups.

**The first playable world should be small:** one cove · one dock · one NPC ·
one rival captain · one wreck · one base area · one small boat · one treasure
rumor.

**The long-term vision can include:** ship combat · duels · wildlife · rare
weapons · AI-aware NPCs · rival captains · factions · economy · skill-challenge
minigames · nemesis-like systems.

> **The guiding principle:** build the smallest version that feels like a living
> pirate world, then expand only where the fun demands it.
