# Wishlist

Where ideas land before they're committed work. The [README](../README.md#development-priorities)
holds the *committed* near-term milestones; the [proposal](proposal.md) holds the
*full* design vision. This file is the bridge: the running list of things we
want, roughly shaped and roughly ordered, so nothing good gets lost and
nothing half-formed sneaks into a sprint before it's ready.

**Guiding principle (unchanged):** build the smallest version that feels like a
living pirate world, then expand only where the fun demands it.

## Legend

| Tag | Meaning |
|---|---|
| ✅ **Done** | Shipped and playable on `main` |
| 🔜 **Next** | Committed or nearly so — see README priorities |
| 📐 **Planned** | Wanted soon; shape is mostly understood |
| 🌫️ **Someday** | On the vision board; needs a design pass before code |

---

## The north star: a demo worth showing

The goal is a vertical slice that's genuinely *worth showing off* — not a tech
demo, a **moment**. The target moment:

> You land in the cove, gather driftwood and scrap iron off the shore, craft a
> hammer, build a campfire, play a quick fire-lighting minigame to get it lit,
> cook your first meal — while a friend wanders the same cove and Silas the
> smuggler comments on what you're carrying. It looks good, it's on a real
> domain, and it remembers you tomorrow.

Everything below serves that moment or the world it lives in. The rough path:

| Stage | What it adds | Pulls in |
|---|---|---|
| **A. Survive & make** | Scrap iron + stone nodes, crafting v1, campfire + cook | Crafting, minigame seed |
| **B. Build the cove** | First placeable, persistent base objects | Base building v1 (README #4) |
| **C. Go public** | Real domain, TLS, a landing page, off-machine backups | Cloud hosting (README #3) |
| **D. Make it pretty** | Readability + art pass so screenshots sell it | "Look better" |
| **E. Give it stakes** | A story thread, a rival who remembers you | Storyline, nemesis seed |

Stages can interleave, but **A → make it fun** and **C/D → make it showable**
are the two that matter most for "worth showing."

---

## Features

### Core gameplay

| Feature | Status | Notes |
|---|---|---|
| **Inventory** | ✅ Done | Persists in Postgres; toggle panel + legend ([PR #9](https://github.com/user-benjamin/last-free-port/pull/9)) 🎉 |
| **Build / crafting mechanics** | 📐 Planned | The next big playable beat. Full design in **[crafting.md](crafting.md)** (workstations, recipe tree, progression, durability). |
| **Skill tree / level progression** | 📐 Planned | Resolved: **use-based proficiency** (you get better at what you do), not an XP tree — see [crafting.md](crafting.md#progression--four-axes-no-skill-tree). |
| **Combat** | 🌫️ Someday | Grounded and dangerous: melee, pistols, boarding (proposal §11.5). Big system; gated behind the make/build loop being fun first. |
| **Minigames** | 🌫️ Someday | A framework, not one-offs (proposal §12). **First instance arrives early** as the campfire fire-lighting game — a tiny, contained proof. |

### World & story

| Feature | Status | Notes |
|---|---|---|
| **Storyline** | 🌫️ Someday | Not a "save the world" quest — a *world-pressure* spine: "The Pardon is Coming" (proposal §6–7). Start with one rumor thread in the cove. |
| **Merchant / economy system** | 🌫️ Someday | Silas is the seed. Buy/sell, regional prices, contraband (proposal §19.5). Needs currency as a server-authoritative item type first. |
| **Reputation** | 🌫️ Someday | Local & faction-specific, not a good/evil bar (proposal §10.2). Hooks into NPC reactions and the nemesis system. |
| **Nemesis system** | 🌫️ Someday | A named rival captain who remembers defeats, raids your cove, mocks you in the tavern (proposal §19.4). Grounded rivalry, no resurrection. |
| **Cities, villages, settlements** | 🌫️ Someday | Beyond the one cove: a smuggler port, a Crown town. Each a place with NPCs, trade, and heat. |

### Presentation & shell

| Feature | Status | Notes |
|---|---|---|
| **Game menu** | 📐 Planned | Pause / settings / leave, and a home for gore + accessibility toggles (proposal §17, §12.3). Currently only a "Return to Port" button exists. |
| **"Look better" — readability + art pass** | 📐 Planned | Honest note: *it's hard to see and a bit ugly right now.* Contrast, camera framing, UI legibility, a cohesive art pass. This is what makes screenshots sell. |
| **Landing page** | 🌫️ Someday | DNS straight to login, or a welcome page first? See the open question below. |

### Infrastructure

| Feature | Status | Notes |
|---|---|---|
| **Cloud hosting** | 🔜 Next | README milestone #3. Terraform → Hetzner VPS, Caddy TLS, GHCR images from CI, tested backups. *Now safe to do — auth landed.* |

---

## Open question: landing page vs. straight-to-login

When someone hits the domain, what do they get?

- **Option A — DNS straight to the game/login.** `lastfreeport.example` → the
  login screen (or a browser export of the client). Fastest path; least to build.
- **Option B — a welcome/marketing page first.** A real landing page (pitch,
  screenshots, a "Play" button, maybe email capture) that *links* to the
  client. Better for showing off; it's the page you'd put on a résumé.

**Leaning:** B, eventually — a landing page *is* a portfolio artifact (proposal
§3, §23). But it only earns its keep once the game behind it looks good (the
"look better" pass). Until then, A is fine. Worth a short ADR when we reach
cloud hosting, since it shapes the DNS/Caddy routing.

---

## Crafting v1 — first slice

> The full crafting & progression design — workstations, the recipe tree, the
> four progression axes, durability, and fuel — now lives in
> **[crafting.md](crafting.md)**. This is just the *first slice* we'd ship.

Not the whole system at once. The first playable crafting beat is the
**campfire loop** — it threads gathering, crafting, station placement, and the
first minigame through one short, legible chain with a visible payoff:

```
gather stone + driftwood
        │
        ▼
craft  ──► Campfire (an unlit placed object in the world)
        │
        ▼
interact ──► fire-lighting minigame  (flint & steel / bellows timing)
        │            │
        │            ├─ server validates: allowed? plausible duration? score in bounds?
        │            ▼
        │       graded result (fail · poor · normal · good · great · perfect)
        ▼
   Campfire is LIT  ──► cooking unlocked
        │
        ▼
cook raw food ──► a meal (heals / buffs)   ── and later: light, a gathering spot
```

It's the proposal's "one craftable item + one minigame" MVP slice (§18, §12)
made concrete and pirate-flavored — and it proves the `craft_intent` →
server-validates → persist path that every later recipe reuses.
