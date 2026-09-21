# niebla — technical notes

How the code is built. The game design lives in [DESIGN.md](DESIGN.md); this
file is for whoever works on the code, human or agent.

## Commands

From the GoLib repository root:

```text
./golib run niebla          # play it
./golib test niebla         # go vet + go test
./golib shot niebla 30      # the menu, with the player's number
./golib shot niebla 120 --input "Enter@1"   # reach the region: Play is Enter
```

A shot starts on the menu; `Enter@1` presses Play (or click it: `Mouse@5:640,390
MouseLeft@6`). Inside the region:

```text
./golib shot niebla 120 --input "Enter@1 Mouse@5:568,372 MouseLeft@6 Mouse@7:700,386 MouseLeft@8 Mouse@9:678,450 MouseLeft@10"
```

The clicks pick the safe oil pool at tile (11, 14), expand its card
and press its send robot button: (568, 372) is the tile's center on
screen, (700, 386) the card's title row, (678, 450) the button. The frame
shows the card expanded, offering recall robot. The camera at rest centers
the view on world point (640, 372) — the middle of its bounds — so screen
and world differ by (0, -12) at rest zoom; click targets in scripted shots
aim at a tile's center (`projectTile` + half a tile, plus that offset),
never its corner, whose tile depends on float rounding.

## Dev tools

In the region, hold Control and click the game's name twice: a strip
opens under the HUD with **hold a swell** (the fog presses in and stays
until the button lets it go) and **place robots** (every click on the
region drops a free built robot there; a right click disarms). In a
scripted shot:

```text
./golib shot niebla 400 --input "Enter@1 Mouse@2:40,24 \
  LeftControl@3-12 MouseLeft@4 MouseLeft@8 \
  Mouse@14:76,77 MouseLeft@15 \
  Mouse@18:204,77 MouseLeft@19 Mouse@22:250,330 MouseLeft@23"
```

## Files

| File | Holds |
| --- | --- |
| `main.go` | `main` — who is playing, then the menu scene —, screen size, every color of the game, the monitor filters (made once, shared by the scenes) |
| `menu.go` | The title screen: the game's name, the player's number, Play and Quit; the `menuButton` hit-testing both scenes' menus use |
| `identity.go` | Who is playing: the machine's ID (registry value, platform UUID or `/etc/machine-id`), hashed with the game's salt into `player`, the number the menu shows and a later server hands tokens out by |
| `store.go` | The local database (SQLite): players, saves and the machine table; `saveBase`/`resumeState`, the scenes' door into it; the DB path, `:memory:` under `golib shot` |
| `play.go` | The play scene: input to actions plus one `Tick` per update; the camera, the selection, the marking blueprint and the open cards live here, never serialized; Esc saves and returns to the menu, autosave every `autosaveTicks` |
| `radial.go` | The build menu: the radial of blueprints a click on empty ground opens |
| `state.go` | The simulation's state: robots (core or built), buildings, stock, what remains of each deposit, build jobs; `newGame`, which deals the starting region |
| `actions.go` | The actions (`Tick`, `SendRobot`, `RecallRobot`, `MarkBuilding`, `QueueRobot`, `Demolish`, `CancelJob`, `LayPipe`, `RemovePipe`, and the dev tools' `DevHoldSwell` and `DevSpawnRobot`) and `Apply`, the only door into the state |
| `sim_robots.go` | The robots' rules and tuning: `robotDay`, the lines of a robot's day in priority order — carry home, mind the tank, finish loading, oldest build job, pick up loose items, own post, idle by the core |
| `sim_piles.go` | Demolition and loose items: `canDemolish`, the piles (`dropPile`, `pileOffer`, `nearestPile`, `takeFromPile`), the stores' free room and `storeSpot`, where a load is unloaded |
| `sim_buildings.go` | The buildings' rules and tuning: blueprints' costs, placement and safe zones, storage caps, refuel spots, the factories' robot works |
| `sim_oil.go` | Oil's tanks: the core's, the silos' and the chargers'; `oilTotal`, `oilCap`, `payOil`, and `haulTank` and `refuelTank`, where a robot carries oil to and refills from |
| `sim_pipes.go` | Pumps and pipes: the `Pipe`, its curve (`pipePath`, a centripetal Catmull-Rom spline through the bends), sections and cost, `canJoin` and the ports, the robots' work on it, `stepPipes` and `pumpStatus` |
| `sim_fog.go` | The fog's law and tuning: cycles, swells, where the line stands now (`fogLineNow`), the drag a walker keeps (`fogDrag`) |
| `swell.go` | How a pressing swell looks, by its pressure: waves of shade rolling in to the line and one-pixel static over the mist; a pure picture of the state |
| `region.go` | The hand-made 25x25 layout, the isometric `project`, tile helpers, the deposit patches flooded out of the layout; pure Go, no drawing |
| `things.go` | What a tile holds: `Thing` snapshots out of layout plus state (a deposit tile shows its whole patch), `tileAtWorld`, the SI quantities; pure Go, no drawing |
| `catalog.go` | The entity database: per thing type its name, color, unit and card lines, plus the stable-color fallback |
| `markup.go` | The `[name]...[/]` colored-text markup: parser and drawer |
| `inspect.go` | The inspection panel: layout, hit testing, painting, the cards' buttons; tile highlights |
| `mites.go` | The fog's wear, for looks only: mites of darkness orbiting whatever stands in the mist, by its volume, trailing walkers and closing in on what stands still; view, never state |
| `pipes.go` | Pipes on the screen (`drawPipes`: casing, body, the ghost of the unlaid part, the blobs of oil by the state's tick) and the pointer's mode that lays one (`pipeLaying`, `updateLaying`, the curve in hand and its price) |
| `dev.go` | The dev tools: Control and two clicks on the game's name open a strip of buttons — hold a swell, place free robots —; view only, acting through the `Dev*` actions; `unitsAtWorld`, the inverse of `project` |
| `draw.go` | The region painter: ground, the core's monolith, buildings, robots, fog, bubbles, build-site wireframes, the marking ghost |
| `region_test.go` | Layout, projection, things, SI formatting, catalog tests |
| `markup_test.go` | Markup parser and tooltip layout/button tests |
| `world_test.go` | The simulation driven directly: starting robots, hauling, picking, priority, recall, dry deposits, determinism, JSON round trip |
| `buildings_test.go` | The buildings driven directly: marking pays and raises, the fog refuses ground, the factory's robots, refueling, digestion, the fog's drag, full stores and silos, the protector's bubble on its cell |
| `pipes_test.go` | Pumps and pipes driven directly: a pump stands on a pool and a pool takes one, a pipe is paid by the section and laid by the robots, a laid pipe carries the pool into its tank and stops at a full one or a dry pool, oil has a place and pipes move it between tanks (shares, ports, payments, a demolished tank's oil), robots carry oil to a tank with room and refill where there is oil, what `LayPipe` refuses, a pipe leaves with its ends and its cost falls as a pile, the curve passes through its bends, pipes survive a save, the pool's cards carry the pump and its pipe |
| `mites_test.go` | The mites driven with no window: counted by volume and only in the fog, tight on what stands still and trailing a walker, fading over what is gone, the falloff's layers |
| `dev_test.go` | The dev actions: a held swell stays up and doesn't count, a placed robot is built, full and free, `unitsAtWorld` undoes `project` |
| `fog_test.go` | The fog driven directly: cycles, the first swell on schedule, the pressed line, the bubble's margin, the pushed band's drag, the swell's burn, the HUD's forecast |
| `identity_test.go` | The identity derived from a machine ID: stable, distinct, and the parsers of what `reg query`, `ioreg` and the machine-id files say |
| `store_test.go` | The database driven directly: an identity kept across runs, the fallback one too, the token column waiting empty, a base saved and loaded back whole, a second save replacing the first, one player's save invisible to another, the DB path's rules |

## Architecture

DESIGN.md's four laws (one serializable state, actions in/state out, the
game is a visualization, determinism) hold since slice 4, in a first,
robot-sized form:

- `State` (`state.go`) is the whole game: robots by ID, the stores, what
  remains of each deposit patch (one key per patch, flooded once out of
  the static layout in `region.go` — deposits become entities when
  buildings need neighbors), and the build jobs, which nothing marks yet
  but every robot obeys. It has no pointers, channels or functions, so
  it serializes as it is.
- Actions (`actions.go`) are structs (`Tick`, `SendRobot`, `RecallRobot`);
  `Apply` mutates the state it is given — one owner, no copies — and is
  total and deterministic, so a seed plus an action log replays a game.
- The robots carry no plan: `stepRobot` (`sim_robots.go`) derives each
  tick what one does from `robotDay`, a list of tasks in priority order
  (carry home, mind the tank, finish loading, oldest build job, pick up
  loose items, own post, idle by the core): the first task that claims
  the robot owns its tick, and the robot's caption reads the same list
  (`Robot.taskNow`). A per-robot task list, when it comes, is a filter
  over it.
  Anything that iterates entities iterates them in sorted ID order.
- The play scene sends input actions and one `Tick` per update; `Draw`
  only reads. View state — camera, picked tile, marked blueprint, open
  cards, pointer — lives in the scene and never serializes.
- The catalog and the markup palette are view-side metadata, not state:
  package-level tables, like a schema. Card lines read the state, so
  they can say what remains in a deposit and what a robot is doing.

### Buildings and built robots

The ground divides past its tiles: a tile of 200 u holds 8 by 8 **cells**
of `buildingCell` 25 u on a side (sim_buildings.go), and one cell is the
footprint of the smallest building — about four robots across — the
tile grid's last subdivision, so a tile may hold several buildings. A
building's `Col, Row` in the state are cell coordinates; `cellAtWorld`
undoes the projection onto the cell grid the way `tileAtWorld` does onto
tiles.

Five blueprints in the radial menu, and the pump off it (see [Pumps and
pipes](#pumps-and-pipes)) (`BuildingKind` in `state.go`, rules and tuning in
`sim_buildings.go`): the **robot factory** builds robots from lilac and
oil, the **charger** refills a built robot's tank from the stores, the
**silo** and the **warehouse** add oil and lilac storage room, and the
**shadow protector** holds a small bubble of safe ground of its own.
Marking is building, and it takes two clicks: a click on a free cell of
ground opens the **radial build menu** (`radial.go`) right on that cell —
the options lay out around the cell's projected center every frame, so
the menu follows the view — and picking a blueprint pays its cost from
the stores and marks it on that very cell. Options read their own
validity (`canPlace` plus `canAfford`): the ones the ground, the fog or
the stores refuse sit dimmed and ignore clicks. The job joins the queue;
the robots raise the oldest job first, standing on the cell's edge
(spread by ID) where the rising body can't swallow them, and the site
shows the part already built in solid colors inside a **wireframe** of
the whole body, with the work's progress bar under the cell, drawn over
the fog so a site in the mist stays visible.

The fog's law, in `canPlace` and `inSafeZone`: nothing but a protector
may be marked outside a bubble, so expansion is protector first, then
the infrastructure it shelters. The bubbles also cancel the fog's drag,
which slows every robot to half its pace deep in the mist.

### Oil's tanks

`sim_oil.go`: oil has a place. `State.Stock.Oil` is the core's own tank
and `Building.Oil` a silo's or a charger's (`tankCapOf`); a tank is named
by its building's ID, the core's by `coreTank`, 0. `oilTotal` and
`oilCap` sum them for the HUD and for `canAfford`, and `payOil` takes
what the colony spends out of any tank, the core's first. Robots choose
with `nearestTank`: `haulTank` is the nearest tank with room (where
`storeSpot` walks an oil load and `deposit` pours it), `refuelTank` the
nearest charger or core with oil (where `refuelSpot` walks and `refill`
draws). `Demolish` drops a tank's oil in its pile. Lilac is still one
stock under `lilacCap`. A save from before the tanks loads with all its
oil in the core, over its cap if need be: it takes no more until it is
used or piped away.

### Pumps and pipes

The **pump** (`BuildingPump`) is the one kind `canPlace` takes on oil
instead of ground: on a pool with oil left and no pump yet
(`patchPumped`), inside a bubble. It isn't in the radial menu - a click on
a pool inspects it - so the pool's card carries `build pump`, which marks
it on `pumpCell`, the patch's middle; `sameGround` makes `thingsAt` show
a pool's pump and site on every tile of the pool.

A **pipe** (`State.Pipes`, by ID) carries oil one way, `From` a pump or
a tank `To` a tank, through the player's `Bends`, in units. `canJoin` is
the network's law: two different ends, a port free on each
(`freePorts`: `pipePorts` 3, `corePipePorts` 6), no pipe between them
already. The curve is never stored: `pipeSpine` rebuilds it from the
ends and the bends with `pipePath` - a centripetal Catmull-Rom spline
cut into `pipeSpanSamples` pieces a span, with a `sagPoint` when there
are no bends - so the sim (length, sections, where the robots stand) and
the view (the drawing, the blobs) read the same line. `Sections` is what
was paid and `Left` the robot work owed: `building` claims a robot for
the oldest unlaid pipe once no site is left, and `stepLayPipe` stands it
at the pipe's head (`pipeLaidPart` along the curve), working a tick per
arrival. `stepPipes` runs after the factories: every `pipeFlowing` pipe
(laid, `pipeSupply` in its source, `tankRoom` at its end) moves up to
`pipeLitersPerSecond`, the pipes of one source sharing what it gives -
for a pump, the `pumpLitersPerSecond` it draws from `State.Drain`.
`Demolish` calls `takePipesOf`, so a pipe never outlives an end.

The laying mode is view (`pipeLaying` in the scene): it collects bends
and sends one `LayPipe` (`sendPipe`) on the click that lands on a tank
the pipe may end at (`layTargets`) - `layTarget` tests the pointer
against the tank's body on the screen, foot to top, a building winning
over the core - or on `connect` in the last node's menu
(`layMenuLayout`, `pickLayMenu`), which ends it at `nearestTarget`. The
panel lists an end's pipes as button rows (`pipeEndOf`, `pipeNote`; the
row's `ref` is the pipe `RemovePipe` takes). `drawPipes` runs twice,
like the piles: clear stretches under the buildings and robots with
their shadow, fogged ones and the unlaid ghost over the fog; the pipe is
drawn `lift` above its ground line, on posts.

### The fog breathes

The weather is state (`State.Fog` in `state.go`, law and tuning in
`sim_fog.go`): `Cycle` counts whole cycles of `fogCycleTicks` (30 s);
`NextIn` counts the cycles of calm left before the next **swell**;
`SwellLeft` is the ticks the current swell has left; `Swells` remembers
how many have passed, and every dial grows with that count — each swell
comes `fogSwellQuickener` times sooner (floor `fogSwellMinPeriod`),
lasts `fogSwellTicksGrowth` ticks longer (roof `fogSwellTicksMax`) and
presses `fogSwellGrowth` tiles deeper (`fogSwellReach` the first time),
but never past `fogSwellMargin` of the bubble: `swellReach` caps it, so
the core's ground is not negotiable whatever the swell count.

A swell starts and ends on its tick, but what the ground feels is its
`Pressure`, 0 to 1, which `stepPressure` walks up while a swell is up
and back down in the calm, `fogSwellRampTicks` either way: `fogLineNow`
is the calm line minus the reach times the pressure, and the extra burn
scales with it. `swell.go` draws the pressing - crests of shade rolling
in (`drawSwellWaves`, in the world, under the fog line) and static over
the mist (`drawSwellStatic`, in screen pixels) - from `State.Ticks`
alone, through `hashUnit`, with no randomness and no memory.

A swell starts whole at a cycle's end — which is what makes the HUD's
forecast exact: while it says `swell next cycle` (the ghost line stands
where the fog will press in), the swell rises at that very boundary.
While it is up, `fogLineNow` returns the pressed line, the view paints
the band and the line there, and the sim gets meaner in the pushed
band: `fogDrag` keeps a quarter of the step where the calm fog would
leave clear ground (`fogSwellSpeedFactor`), and built robots outside a
bubble burn their tanks 1.5x (`fogSwellBurn`). Nothing else changes:
placement (`canPlace`) and the bubbles never read the swell.

Two kinds of robot (`RobotKind`): the core's own, free and tankless, and
the factory's, paid in lilac and oil. A built one burns oil as it walks,
hauls or idles; under the low line (`robotLowTankAt`) the tank claims its
day and walks it to the nearest charger or the core, where it stands
until the tank is full — even past the low line, so it doesn't dance
between post and work — and outside a bubble, a tank at zero means the
fog digests the robot. Both the burn and the drag are dials at the top
of `sim_buildings.go`.

The stores have a roof: `oilCap`/`lilacCap` is the core's own room plus
every silo and warehouse. A robot hauling into a full store stands at
the store trying again each tick (its card says waiting for storage), and
deposits what fits when a silo opens room. The stores are one stock, but
a load's walk ends at the nearest store of its kind (`storeSpot`): a
warehouse or the core for lilac, a silo or the core for oil.

Demolition (`sim_piles.go`): `Demolish` takes a building out of the
state at once and `CancelJob` a site out of the queue. What it was made
of falls on its cell as one `Pile` (`State.Piles`, by ID): the
blueprint's cost times `demolishRefund`, the cost of the robot a factory
was building, and what the stores lose the roof for (`spillOverflow`).
A pile holds its cell against `canPlace` until its last item leaves,
which deletes it. A protector can't go while it alone shelters another
building or a site (`canDemolish`, on `shelteredWithout`). Robots pick
piles up after build jobs and before their posts: the nearest pile that
holds something the stores have free room for (`freeRoom` counts what
is already on its way home, so nobody loads what won't fit), one kind
per trip, lilac first, loading for `robotLoadTicks`; `Robot.Pile` says
which pile a loading robot stands at, 0 at its post.

### The lifecycle, identity and the local database

The game boots on the **menu** (`menu.go`): the game's name, the player's
number, Play and Quit. Play carries the player to their base as they left
it — `resumeState` loads the last save, or deals a new region when there
is none. Esc in the region saves and returns to the menu; the region also
saves itself every `autosaveTicks` (900, 15 s), so a window closed without
ceremony loses less than that. The menu is the only screen where Esc
quits.

**Identity** (`identity.go`): the machine says who is playing. Its
system ID — Windows' `MachineGuid`, macOS' `IOPlatformUUID`, Linux'
`/etc/machine-id` — hashed with `playerIDSalt`, is `player`: 64 hex
characters, stable across runs, and never shown or sent in raw form. The
menu shows its first eight as `#30E99076`; the database and a later
server use the whole thing. A machine that won't say who it is gets a
random identity, kept in the database's `machine` table, so it is still
stable from then on. Identity is per machine, not per human: two players
on one computer share a number, and an avatar picker is the later answer.

**The database** (`store.go`) is SQLite through `modernc.org/sqlite`
(pure Go — this project has no C compiler), one file at the player's
settings folder, in `GoLib games/niebla/`, the same place a GoLib dist
build keeps its saves, so a debug build and a dist one share the base.
The schema is the schema a server keeps, on one machine for now:

- `players` — one row per identity, with `source` (`machine` or
  `random`), `created_at`, and a `token` column that stays empty until a
  server hands one out at first contact. The client is already shaped
  for that moment: resolve, register, then authenticate by identity.
- `saves` — the whole `State` as one JSON value per player and slot
  (`region` for now), with the tick and the time it was written. The
  server will hold one authoritative region per player the same way.
- `machine` — key-value for what belongs to this machine alone (the
  fallback identity lives here).

Under `golib shot` and `go test` the database is `:memory:`: shots and
tests never touch the player's base, and `golib shot --save` still
starts a game deep in a state — `resumeState` takes the seeded `state`
value over whatever the database has. Two caveats: the driver doesn't
build for the browser (`js/wasm`), so a web build of this game will get
its store from a server or the browser's own, not this file; and the
simulation itself never reads the clock — only the saves' timestamps do.

### Entity catalog

`catalog.go` holds one `ThingInfo` per `ThingType`: name, color, the SI unit
of its headline amount, an optional `Summary` (default: `si(amount, unit)`)
and `Details` for the expanded card. A type with no entry gets its name from
the type itself and a color from `stableColor`, a hash of the name through
HSV — the same type always prints in the same color, so new entities are
readable the moment they exist. To dress a type up later, add its catalog
entry; the markup palette picks the color up automatically.

### Text markup

`"[oil]900 L[/]"` prints `900 L` in the oil color. `[name]` switches to the
palette color `name`, `[/]` returns to the color before it, colors nest, and
unknown tags print as they are. The palette holds one entry per thing type
plus `dim`, `light` and `fog`. `drawMarkup` draws span by span, advancing by
`TextWidth` (which counts one letter gap per character, gaps included after
the last), so spans land where one `DrawText` call would put them.

### Units

The world speaks SI: one world unit is one meter (`unitMeters`), so a tile
is 200 m across (4 ha) and the region 5 km from side to side, the core's
monolith is 16 by 4 m and 36 m tall, and its bubble radius is 800 m. Oil is
liters, lilac is kilograms (`si` turns 12000 kg into `12.0 t`, so nobody
ever reads `kkg`). The per-tile amounts (900 L, 3000 kg) are the deposits'
density; a patch of four tiles holds four of them (a whole vein: 12 t).
Robots are fast rovers with small arms (30 m/s, 30 L or 20 kg a trip), so
a worked deposit shows a constant coming and going. Amounts live at the
top of `things.go`.

### Deposit patches

A vein is one thing however many tiles it spans. `findDeposits`
(`region.go`) floods the layout once, at startup, into `regionDeposits`
(static data, never state) and `depositAt` maps any tile to its patch.
`State.Drain` holds one entry per patch, keyed by the patch's top corner
tile; a robot's post is one tile of the patch, and working it drains the
whole patch — one robot per patch, one card per patch, one big scar when
it runs dry. `draw.go` paints each patch as one continuous body that
shrinks with what remains of it.

### Tile picking

`tileAtWorld` undoes `project`: a tile's diamond on the screen is the square
`[col, col+1) x [row, row+1)` in tiles, so the inverse is exact, and a point
maps to its tile with two `math.Floor` calls. The inspection panel anchors
on the tile's projected corner through `camera.ToScreen`, so it follows the
tile while the view pans or zooms, and flips to the tile's left when it
would leave the screen.

### Inspection panel

One geometry, two users: `tooltipLayout` builds the row list, and `Update`
hit-tests it (`contains`, `trashAt`, `cardAt`, `buttonAt`) while `Draw`
paints it. A building's or a site's title row ends in a trash can
(`trashFor`; none on the core, dimmed on a protector that can't go): the
first press arms it — the scene's `armed` holds the card's ID and
`tooltip.arm` paints it red under `demolish?` —, the second applies
`Demolish` or `CancelJob`, and any other click disarms. A site and a
pile have cards of their own (`siteThing`, `pileThing`).
Cards start open on their own: a tile's primary thing (deposits, the
core, buildings — `Primary` in the catalog) and, on a tile with a single
thing, that thing. A click on a title folds or opens from where the card
stands (`cardOpen` gives the default, the scene's `expanded` map stores
the click). Clicking a tile selects it — unless the cell under the
pointer is free buildable ground, which opens the build menu instead —
an expanded deposit card carries a send robot / recall robot button, a
factory card a build robot button; the panel wins over what sits under
it, so its buttons work even where it covers buildable ground; a right
click that never moved more than 4 px deselects (a drag is a pan, not a
cancel). The
camera rests centered on world point (640, 372) — screen and world
differ by (0, -12) at rest zoom; click targets in scripted shots aim
at a tile's center (`projectTile` + half a tile), never its corner, whose
tile depends on float rounding.

## Testing

`region_test.go` pins the layout's placement rules, the deposit patches
(shape and spread), the projection's round trip, the things a tile holds,
the SI formatter and the catalog's stability. `markup_test.go` covers the
parser (nesting, unknown tags, unclosed color), the tooltip layout's rows
and hit testing, and the cards' robot buttons. `world_test.go` drives the
simulation with no window: the starting robots, a haul's conservation
(store + patch remaining = patch full), the patch law (one robot per
vein, sends on its other tiles change nothing), who takes a post,
build-job priority, recall, dry patches, replay determinism and the JSON
round trip. `buildings_test.go` does the same for the buildings slice: a
marking pays and its building rises, the fog refuses ground but not a
protector, one tile fits several buildings on its cells, a cell with a
job pending takes no second job, the factory queues and rolls out tanked
robots, a built robot refuels before it runs dry, the fog digests a dry
one outside the bubbles, deep fog halves every walker's pace and a
protector's pocket cancels it, and full stores hold the cargo until a
silo opens room. `piles_test.go` pins the demolition: the cost falls as
a pile and comes home whole, a factory's robot is cancelled and refunded,
a silo spills what loses its roof and the oil waits for room with nobody
holding it, a cancelled site drops its cost, a protector stays while it
alone shelters a building, a load goes to the nearest store of its kind,
piles survive a save (and a save from before them takes one), and the
cards carry their trash cans. `fog_test.go` does the same for the fog slice: the
cycles tick, the first swell rises on schedule and drains whole, the
line presses in and never reaches the bubble, the pushed band drags
more, a swell burns outside but not inside, and the HUD forecasts.
`mites_test.go` pins the fog's mites, which are view but need no
window: their count follows the body's volume and the fog on it, none
under a bubble; they sit on a robot that stands still and trail one that
walks; they fade over a host that left the state; and the layers they
are drawn with stack into `miteFalloff`.
`identity_test.go` pins the identity: stable for a machine, distinct
between machines, 64 hex characters, and the three parsers of what the
systems report. `store_test.go` pins the database: an identity (and a
fallback one) kept across runs, a fresh player's token waiting empty, a
base saved and loaded back whole, a second save replacing the first,
one player's save invisible to another, and the DB path's rules
(`:memory:` under `golib shot`, the settings folder otherwise, a
missing one is an error).
Visual checks are shots with scripted clicks; see the
command above.
