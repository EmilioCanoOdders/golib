# Niebla

## Pitch
An isometric survival game about the Gray Fog: a planet-covering mist of self-replicating nanites that digests anything that stands still. You run a small colony of robots around an indestructible fog-repelling core: keep the bubble fed with oil, harvest lilac mineral outside it, and grow, one repulsor at a time. One state, one truth: the whole game is a serializable simulation that can be saved, replayed and later served to many players.

## Core loop
The fog advances in cycles; inside the core's bubble nothing is touched, but the bubble eats oil. Outside it, oil pools and lilac veins wait. Every cycle:
1. Spend oil: keep the bubble up, keep the robots charged.
2. Send charged robots out to harvest oil and lilac, and back before their charge runs out - a robot that stalls in the fog is digested.
3. Mark build jobs (repulsors, pumps, drills); robots fetch the parts and build them.
4. A real repulsor widens or holds the safe ground, but it also drinks oil. The core is free of tension; the need for resources outside is what pushes you out.

## Resources
| Resource | What it is | What it is for |
| --- | --- | --- |
| Oil | Pumped from pools | Bubble and repulsor fuel; robot charge ("food") |
| Lilac mineral | Mined from veins (StarCraft vespene / Dune spice vibe) | Every construction and upgrade |

There is no food and no people: the colonists are robots, and an uncharged robot outside a bubble dies to the fog.

## The core
An indestructible repelling core, deployed at the start of a region. It is the learning zone: a small, permanent bubble with no upkeep, so the first cycles are comfortable. It cannot win the game: its bubble is small, and everything the colony needs lies outside. Growth has a price from the moment you leave.

## Construction model
The player never builds by hand. Marking is building:
1. Choose a blueprint (repulsor, pump, drill) and click a spot: this creates a build job.
2. Idle robots fetch the job: haul lilac to the spot, raise the structure.
3. Harvesting works the same way: mark an oil pool or a lilac vein, and robots commute between it and storage while their charge lasts.

Robots are simple workers with priorities, not pathfinders of genius: the player solves the layout, the robots solve the walking. A later avatar (a character the player moves directly) exists for urgencies, and for multiplayer.

## Architecture
The game is a deterministic simulation first, and a picture of it second. These rules are law; every feature bends around them.

1. **One serializable state.** The whole game is a single value (`State`) that serializes to JSON with no pointers, no channels, no functions. Entities live in ID-keyed tables (`map[int64]Entity`-style, with fixed field structs); every reference between things is an ID, like a relational database. Saving = `golib.SaveData` of the state. Loading the state = loading the game, exactly.
2. **Actions in, state out (flux/redux).** Nothing mutates the state except reducers. An action is a serializable struct (MarkJob, SetPriority, Tick...). `Apply(state, action) -> state` is pure and total: the same action on the same state always gives the same result. `Tick` is just the action the loop sends 60 times per second of game time.
3. **The game is a visualization.** `Update` reads input and produces actions; `Draw` renders the current state and changes nothing. All rules live in the simulation files, free of `golib.Input` and `golib.Screen`, so tests drive them directly.
4. **Determinism.** The simulation never reads the clock (dt is 1/60), never reads globals, and iterates entities in sorted ID order. Gameplay randomness comes from the state's own PRNG (a seed plus counter inside `State`), so a save reproduces its future; `golib.RandomInt`/`RandomFloat` are for looks only (cosmetic particles, menu clouds). A seed plus an action log replays any game - which is also the future multiplayer server: one authoritative sim, clients send actions, receive states.

Consequence for file layout: `state.go` (types), `actions.go` (action types + `Apply`), `sim_*.go` (rules per domain), `world_test.go` (tests call the sim directly); `play.go`/`draw.go` are views. The simulation has landed for robots and deposits: `state.go` holds the state (`State`: robots, stock, what remains of each deposit tile, build jobs), `actions.go` the actions (`Tick`, `SendRobot`, `RecallRobot`) and `Apply`, and `sim_robots.go` the robots' rules; deposits are still terrain-shaped (a remaining-amount grid keyed by tile, not entities), and the region layout stays static data. Apply mutates the state it is given - the state has one owner, and purity here means determinism. `region.go`/`things.go` hold the world's static side, and `catalog.go` (the entity database), `markup.go` (colored text) and `inspect.go` (the inspection panel) are view-side metadata, never state. [README.md](./README.md) is the technical map of the code.

## Screens
- **Title:** the region under the fog, the game's name, Play / Quit.
- **Play:** the isometric region, the camera panning and zooming, the bubble drawn over the ground, robots shuttling, the fog line visible and creeping, resource counters on top. A click inspects a tile: a panel lists what stands there, one expandable card per thing, named and colored by the catalog. An expanded oil or lilac card carries a button - send robot, or recall robot once the tile has one - and a robot standing on the tile shows its own card with what it is doing.
- **Pause:** the frozen region under a message. The fog does not advance while paused.
- **Colony lost** (later): when the core is somehow unreachable, or the player quits the region.

## Controls
| Input | Action |
| --- | --- |
| WASD or arrows | Pan the camera |
| Mouse wheel | Zoom toward the cursor, gliding between whole steps (pixel art stays square at rest) |
| Mouse right, held | Drag the view: grab the ground and move it |
| Mouse left | Select / mark: inspect a tile (a click on a card expands it, a card's button acts), place blueprint, mark harvest, pick robot |
| Mouse right, clicked | Cancel marking / deselect |
| Esc | Pause and resume |
| F11 or Alt+Enter | Fullscreen on and off |

## Rules (MVP)
- One hand-made region (Tiled-informed layout, drawn in code at first): oil pools, lilac veins, buildable ground, fog everywhere else.
- The core starts with two robots, free of charge: one can watch the oil, the other the lilac. Robots built later will cost oil each cycle.
- Sending is by card: expand an oil or lilac card and press send robot; the nearest free robot takes the tile as its post (all busy, the nearest one is retasked). Recall gives the post back.
- A robot's day has its priority built in: bring home what it carries, finish loading, raise the oldest build job (none can be marked yet, but the order is law), work its own post, idle by the core. A post that runs dry releases its robot.
- Lose nothing at the start: the core is indestructible and its bubble has no upkeep.
- Lose robots: any robot outside a bubble with empty charge is digested by the fog.
- The safe zone feeds you: an oil seam and a lilac vein cross under the core, generous and fully inside the bubble, so at least one resource of each type is minable in comfort whatever the fog does outside.
- Lose the region (later): when oil hits zero inside the bubble and no robot can reach fuel, the bubble collapses; that ends the run.
- Difficulty grows with the fog: each cycle the fog presses closer, so idle safe ground shrinks and haul distances grow.
- No win condition in the MVP; the region is the tutorial for the arc.

## Art
The screen is 2K/2 (1280x720) with `Config.PixelArt`, so a 2K monitor scales it by two whole, sharp numbers; 2K/4 (640x360) is the step when pixel-art sprites arrive. Three monitor filters run in order (`shaders/glow.fs`, `shaders/crt.fs`, `shaders/soft.fs`, adapted from games/asteroids and turned down): a glow that only the brightest things clear (the fog itself outshines the oil, so the threshold is 0.8 there), a whisper of a tube screen (curvature 0.08, faint scanlines, light vignette, no flicker) and a small tent blur that rounds pixels' corners. F2 turns them all off. Isometric look from a manual projection in `Draw` (`screenX = (x-y)*tileW/2`, `screenY = (x+y)*tileH/2`), with painters-order drawing by depth. Note: Tiled maps cannot be isometric in GoLib (orthogonal only), so the region is drawn from sprites and shapes, not `DrawMap`; if Tiled is used, it is only as a layout editor whose data the game re-projects. First version: simple shapes and a small palette (cold ground, lilac veins, amber oil, pale fog); sprites later, tiny robots over larger tiles.

## Sounds
Made in code with `golib.NewSound` (recipes and `SoundSpec`): the fog's low loop outside bubbles, the repulsor hum, robot blips, a digestion crunch, a construction done chime. Fully playable muted.

## Tuning
Pinned as code lands, all at the top of the sim files with units in the name: `fogCycleTicks`, `repulsorOilPerCycle`, `robotChargeSeconds`, `robotMoveSpeed` (units/s), `oilPerPoolUnit`, `lilacPerVeinUnit`, blueprint costs. Pinned so far, in `region.go`: `regionCols`/`regionRows` (25x25 tiles), `tileW`/`tileH` (48x24 px at 2K/2), `unitsPerTile` 5 (the world's unit: a robot is 1 u, the core's pole 5 u across - one tile, a typical building 10 u - two by two, a vein 30 u - six tiles), `coreBubbleRadius` 4 tiles, `fogLineRadius` 10.5 tiles, `fogFadeTiles` 2.4 tiles. In `play.go`: `zoomOut`/`zoomIn` (whole-step zoom, 1 to 4; at 4 a robot's 1 u is about 40 screen px), `zoomGlide` 0.1 s (the zoom glides from step to step instead of jumping, keeping the point under the cursor under it; only at rest is the zoom a whole number), `panSpeed` 480 screen px/s, constant on the screen at every zoom; the right button drags the view, grab style. The camera is view, not state: it lives in the play scene and never serializes. In `draw.go`: `propZoom` 2, the zoom from which the rocks and bushes are drawn, so zooming in reveals detail. The fog's shape and speed are the feel of the game; they get their own section when the first region exists.

The region layout's placement rules, tested in `region_test.go`: oil, lilac and the core inside the fog line; the two seams crossing under the core generously inside the bubble; rocks and bushes on ground outside the bubble, so the comfort zone stays clear, sitting off their tile's middle by a stable jitter.

The world speaks SI: `unitMeters` 1 (one world unit is one meter, in `things.go`, so a tile is 5 m across, 25 m²), `oilPerPoolTile` 900 L, `lilacPerVeinTile` 300 kg, `coreHeight` 14 m; the core's card headlines its bubble radius (20 m). Each thing type gets its color from the catalog in `catalog.go`, which falls back to a color hashed from the type's name, stable forever, for types it has no entry for yet. Text colors itself with the `[name]...[/]` markup of `markup.go`; the palette holds one color per thing type plus `dim`, `light` and `fog`. In `inspect.go`: `tooltipWidth` 260 px, `titleSize`/`textSize` 12/10, the panel anchored to the tile's projected corner (it flips to the tile's left near the screen's right edge), `buttonWidth` 96 and `buttonRow` 22 for the cards' send/recall robot buttons.

The robots, in `sim_robots.go`: `startingRobots` 2 (the core's gift, they cost nothing yet), `robotSpeed` 3 u/s, `robotLoadTicks` 150 (2.5 s loading at a deposit), `robotCarryOil` 90 L and `robotCarryLilac` 60 kg per trip, `robotParkRadius` 3.5 u (the idle ring around the core, spots spread by the golden angle). A drained deposit leaves a scar on its tile and releases its robot. The stores live in the state (`State.Stock`) and show in the HUD and in the core's card; the robots' captions (`things.go`) read the state in the same priority order the rules do, so the words always say what the robot is doing.

Robots and building, as landed and as decided: the robots carry no plan - every tick the rules derive what one does from the state (`stepRobot`), so a save reproduces its future, and their positions are floats in units, the tile computed from them; the cell index that lookups will need stays a runtime structure, never state. Build jobs exist in the state (`State.Jobs`, with the work they ask for) and already pull every idle-armed robot first, but nothing marks them yet: blueprints, buildings and their costs are the next slice.

## Later
- **The arc (mid-game):** stabilizing the first region summons the ark, the mobile base - the game's own idea, arriving as a reward.
- **Server and multiplayer:** the architecture is already the protocol; a server runs `Apply`, clients send actions. Players join as avatars; robots do the routine, players the urgencies.
- **Outposts network:** Subnautica-style forward bases, beacon relays that extend the robots' range; star pattern with the ark as hub.
- **Region map and travel**, logistics between outposts, the fog's origin as the endgame: reclaim the planet region by region.
- Sprite art over shapes; music (a tracker module or OGG when provided; `golib.NewTune` before that).
- **Text and translations:** all in-game text is English. Strings move to `assets/text/<lang>.json` (one flat key-to-string file per language, read once with `golib.ReadAsset`) when the first text-heavy screens land; the language is a player setting, not part of the simulation state.

## Changelog
- 2026-09-20: created with `golib new` (skeleton square). Design brief written from the design talks: gray fog survival, hybrid nomad/sedentary, oil + lilac, robots, indestructible core, mark-to-build, serializable state with action reducers, 2K isometric.
- 2026-09-20: slice 1, the painted region: 25x25 hand-made layout in code (`region.go`), isometric projection, oil pools, lilac veins, the core with its bubble, a static fog line (`draw.go`). Skeleton square removed; nothing moves yet. All text English; JSON string files wait for the first text-heavy screen.
- 2026-09-20: slice 1 polished after review: screen at 2K/2 (1280x720, `PixelArt`); the white slivers at the fog border gone (the fog fades in per tile now, no band); two generous seams of oil and lilac cross under the core, inside the bubble; robot and building-cost policy written down for the next slice.
- 2026-09-20: a subtle monitor filter, `shaders/soft.fs`, rounds the pixels' corners (F2 turns it off); flat areas and the ground's checker stay untouched, and no grain is added.
- 2026-09-20: the filter becomes the asteroids pair, turned down: `glow.fs` with its threshold at 0.8 (the pale fog outshines the oil, so only the core, the bubble's edge and the text glow) at strength 0.9, and `crt.fs` without flicker, scanlines at 0.96, a lighter vignette and curvature 0.08. The soft rounding runs last.
- 2026-09-20: slice 2, the camera: WASD, arrows or left stick pan, and the wheel zooms in whole steps from 1 to 4, anchored on the cursor, the view bounded to the region. The world took its unit (`unitsPerTile` 5, with the proportions it pins: robot 1 u, pole 5 u, building 10 u, vein 30 u), `project` speaks units, and the core's pole now fills its whole tile. Rocks and bushes dot the ground from zoom 2 on, off their tile's middle by a stable jitter. The cell-and-offset model is decided and written down, waiting for the robots.
- 2026-09-20: first feel pass: the zoom glides between its whole steps instead of jumping (`zoomGlide`), still anchored on the cursor, and holding the right button drags the view like a grabbed map, both working together.
- 2026-09-20: slice 3, tile inspection: a click picks a tile (right-click cancels, a drag never does) and a panel lists what stands there, one card per thing, headlined by its amount, expandable on click to its details. The entity database lands in `catalog.go` (name, color, unit, card lines per type; a stable hashed color for types it has no entry for), the `[name]...[/]` color markup in `markup.go`, the panel in `inspect.go`. The world took its SI units (1 u = 1 m; oil in L, lilac in kg) in `things.go`; the two seams and the core have fichas. Technical notes moved to [README.md](./README.md).
- 2026-09-20: slice 4, the robots - the simulation lands (`state.go`, `actions.go`, `sim_robots.go`, `world_test.go`): the core starts with two idle robots; an expanded oil or lilac card has a send robot button (recall robot once the tile has its robot), and the nearest free robot takes the post, hauls loads home and is released when the deposit runs dry, leaving a scar. Build jobs sit in the state and already pull a robot home before its own post, though nothing marks them yet. Stores show in the HUD and the core's card; robots are drawn (body, lit top, glowing eye, cargo pack) and carded with what they are doing. Verified by tests and shots: determinism, JSON round trip, haul conservation, job priority, dry release.
