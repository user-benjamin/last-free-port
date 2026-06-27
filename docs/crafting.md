# Crafting & Progression — design

> **Status: design, not yet built.** Only `driftwood` exists in game today.
> This is the target shape; the [wishlist](wishlist.md#crafting-v1--first-slice)
> tracks which slice ships first.

How players turn gathered resources into tools, stations, gear, and a base —
and how they *progress* without an XP tree. Heavily inspired by Valheim's
workstation + biome-gating loop, re-skinned for a Golden-Age-of-Piracy cove.
Sits under proposal [§11.3 Crafting](proposal.md#113-crafting) and
[§12 Minigames](proposal.md#12-minigame-and-skill-challenge-system).

## Principles

- **Server-authoritative.** The client sends a `craft_intent`; the server checks
  inputs, station proximity, tool tier, and skill, then debits/credits inventory
  and persists. The client never decides a craft succeeded (proposal §11.3).
- **Content is data.** Recipes live in JSON (`server/internal/game/recipes.json`),
  validated at startup like NPCs and resource nodes — adding a recipe is a data
  change, not a code change.
- **Grades don't gate.** Where a minigame is involved (e.g. lighting a fire),
  poor performance makes the result *worse*, never *impossible* (proposal §12.1).

## The core mechanism: workstations vs ingredients

The insight that makes everything else click: **fire is not an ingredient, it's
a station.** "`clay + fire → pot`" really means "craft a pot from clay *while
standing near a lit campfire*." This splits the world cleanly:

- **Ingredients** are *consumed* by a craft (driftwood, scrap iron, clay, sand…).
- **Stations** are *placed objects you craft at* — never consumed, sometimes
  fuel-burning, and **upgradeable** to unlock higher-tier recipes.

| Station | Heat | Fuel? | Unlocks |
|---|---|---|---|
| 🔥 **Campfire** | low | yes (slow) | cooking, firing clay |
| 🏭 **Furnace / forge** | high | yes (hungrier) | smelting metal, melting glass, steel |
| 🪑 **Workbench** | — | no | assembling & **repairing** tools and parts |
| 🥩 **Drying rack** | — | no | preserving fish/meat |

Stations are built with a **hammer** (the first tool), which is what makes the
hammer the keystone of the whole tree.

## Resources

| Resource | In game today? | Source |
|---|---|---|
| **Driftwood** | ✅ yes | gather along the shore |
| **Stone** | ❌ add | rocks at the cove edges (later: pickaxe-gated) |
| **Scrap iron** | ❌ add | rusted wreck-iron on the beach |
| **Clay** | ❌ add | dig from cove banks (shovel-gated) |
| **Sand** | ❌ add | the beach |
| **Plant fiber** | ❌ add | palms / brush |
| **Raw fish** | ❌ add | fishing minigame (needs a rod) |

> Adding a resource is small and deliberate: a node type in `resources.json`
> plus a respawn duration in `resources.go`. The inventory panel already renders
> unknown item types, so new resources appear with no client change.

## Recipe tree

`(station)` = must be near that station. `[tool]` = requires that tool. Recipes
are illustrative starting points, not final balance.

### Tools — the gates that give crafting its purpose
| Output | Recipe | Effect |
|---|---|---|
| 🔨 Hammer | `driftwood + scrap iron` | build & repair stations and structures |
| 🔪 Knife | `scrap iron + driftwood` | fillet fish, cut fiber |
| 🎣 Fishing rod | `driftwood + plant fiber` | unlocks the fishing minigame → raw fish |
| ⛏️ Pickaxe | `iron ingot + driftwood` `(forge)` | mine stone / ore |
| 🪏 Shovel | `iron ingot + driftwood` `(forge)` | dig clay & buried treasure |

### Stations
| Output | Recipe | Notes |
|---|---|---|
| 🔥 Campfire | `stone + driftwood` `[hammer]` | the cooking + fire-firing station |
| 🏭 Furnace | `stone + clay` `[hammer]` | high-heat smelting |
| 🪑 Workbench | `driftwood + scrap iron` `[hammer]` | crafting + repair hub |
| 🥩 Drying rack | `driftwood + plant fiber` `[hammer]` | preserve food |

### Materials & intermediates
| Output | Recipe | Notes |
|---|---|---|
| Iron ingot | `scrap iron` `(furnace)` | clay crucible is the *tool*, not consumed |
| Glass | `sand` `(furnace)` | |
| Brick | `clay` `(furnace)` | sturdier base building |
| Charcoal | `driftwood` `(furnace, slow)` | premium fuel + gunpowder later |
| Rope | `plant fiber ×3` | rigging, lashings, traps |
| Sailcloth | `plant fiber ×2` | sails, bandages, flags |

### Survival & consumables
| Output | Recipe | Notes |
|---|---|---|
| Clay pot | `clay` `(campfire)` | enables stews |
| Cooked fish | `raw fish` `(campfire)` | heal |
| Stew | `cooked fish + …` `(campfire)` `[pot]` | bigger heal / buff |
| Bandage | `sailcloth` | field-medicine minigame later |

### Pirate flavor — the stuff that sells screenshots
| Output | Recipe | Notes |
|---|---|---|
| 🏮 Lantern | `glass + iron ingot` | light; the lore's "lantern tricks" |
| 🔦 Torch | `driftwood + sailcloth` | portable light |
| 🍾 Glass bottle | `glass` `(furnace)` | rum, messages, lamp oil |
| 🍺 Grog | `water + molasses` | crew-morale buff |
| 🗡️ Cutlass | `iron ingot + driftwood` `(forge)` `[hammer]` | first weapon, when combat lands |
| 🏴‍☠️ Black flag | `sailcloth` | base decoration + reputation nudge |

## Progression — four axes, no skill tree

Valheim runs several progression systems at once; that layering creates depth
without an XP tree. We borrow all four:

| Axis | Pirate version | Gates |
|---|---|---|
| **Use-based skill** | use a knife → Knife skill ↑ | *effectiveness*: speed, yield, durability wear |
| **Tool tier** | stone → iron → steel tools | *which nodes you can harvest at all* |
| **Station tier** | upgrade the forge (bellows, anvil…) | *which recipes unlock* |
| **Reach / "biome"** | home cove → reef → deep wreck → distant isle → Crown waters | *which resources exist out there* |

**This resolves the wishlist's "skill tree / level progression?" question:**
progression is **use-based proficiency** — you get better at what you do — not a
tree of unlock points. Grounded and on-theme: a sailor hones his craft. Skill is
per-player state persisted in Postgres and raised server-side on use.

### Reach is gated by the ship — the pirate-perfect synthesis

In Valheim the *boat* is the biome gate. That's already our
[§11.4 ship pillar](proposal.md#114-ship-ownership). So crafting, exploration,
and ship ownership collapse into **one loop**:

```
better tools ─► harvest the home cove ─► build a better boat
      ▲                                          │
      │                                          ▼
better forge ◄─ new resources ◄─ reach the reef / deep wreck / distant isle
```

No artificial gates. The water is the wall; your ship is the key.

## Durability & repair

**Model: wear + repair, never destruction.** (We considered skill-driven
catastrophic breakage and rejected it — losing an item you invested resources in
feels bad and makes players hoard instead of *use*.)

- Tools and gear have **durability** that drops with use.
- **Repair is free at the right station** (workbench/forge) — the reward for
  having built a base. Repairing restores durability; it doesn't cost materials.
- **Skill modulates the wear *rate*, not destruction.** Low Knife skill → the
  blade dulls ~2–3× faster (more repair trips, slower work, more waste); high
  skill → it barely wears. Same felt pressure, no gut-punch.
- **True breakage is reserved for being grossly under-tooled** — e.g. a stone
  knife on iron-hard material. Telegraphed ("this blade won't survive that"),
  never random.

**Worked example — the knife.** Stone knife, Knife skill 5/100: filleting is
slow and the blade dulls in ~3 fish, back to the workbench. As Knife skill
climbs you fillet faster, waste less, and the same knife lasts ~20 fish.
Eventually you forge an iron knife (gated behind iron from a *deep wreck* you
needed a better boat to reach) — sharper, tougher, butchers things the stone
knife never could.

Durability is per-item state, computed and persisted server-side.

## Fuel

Stations that burn fuel start **very slow** — a background chore, not a job. Fuel
rate is a server-side tunable from day one so balance is a number, not a rewrite.

- **Starting feel:** a lit campfire consumes ~1 driftwood every ~3–5 minutes; the
  furnace is hungrier. A handful of driftwood is hours of fire.
- Tighten only if fuel ever feels *too* free. The cove should feel like a
  survival place without making gathering the whole game.
- **Charcoal** later = premium fuel (slower, hotter), giving driftwood a reason
  to graduate.

## Server-authority notes

- New protocol message `craft_intent { recipe_id }` → server validates: recipe
  exists · player has the inputs · player is within range of the required
  station · player has the required tool (and tier) · then debits inputs, credits
  output, applies wear, persists, and replies (mirrors `gather_intent`).
- New persisted state: per-player **skills** (`skill_type → level`), per-item
  **durability**, and placed **stations** (likely `base_objects`, proposal §15.3).
- Recipes, like all content, are validated JSON loaded at startup.

## Open questions

- Do placed stations live in `base_objects` from the start, or a lighter table
  until base-building v1 lands?
- Is a station's "lit"/"fueled" state durable (survives relog) or session-only
  at first?
- Station **upgrades** — built by placing required pieces nearby (Valheim) or by
  consuming materials into the station? The former ties crafting to base-building.
- First cookable implies a food resource (fish → fishing minigame). Which lands
  first, cooking or fishing?
- All numbers above are first-guess feel, to be tuned in playtest.

## The v1 slice

Not all of this at once. The first playable crafting beat (see
[wishlist](wishlist.md#crafting-v1--first-slice)) is the **campfire loop**:
gather `stone + driftwood` → craft a campfire → light it (the first minigame) →
cook your first fish. That alone threads gathering, crafting, station placement,
and minigames through one short, legible loop with a visible payoff.
