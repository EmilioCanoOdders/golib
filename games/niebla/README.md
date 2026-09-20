# niebla — technical notes

How the code is built. The game design lives in [DESIGN.md](DESIGN.md); this
file is for whoever works on the code, human or agent.

## Commands

From the GoLib repository root:

```text
./golib run niebla          # play it
./golib test niebla         # go vet + go test
./golib shot niebla 30 --input "Mouse@5:568,372 MouseLeft@6 Mouse@7:700,386 MouseLeft@8 Mouse@9:678,450 MouseLeft@10"
```

The shot above picks the safe oil pool at tile (11, 14), expands its card
and presses its send robot button: (568, 372) is the tile's center on
screen, (700, 386) the card's title row, (678, 450) the button. The frame
shows the card expanded, offering recall robot. The camera at rest centers
the view on world point (640, 372) — the middle of its bounds — so screen
and world differ by (0, -12) at rest zoom; click targets in scripted shots
aim at a tile's center (`projectTile` + half a tile, plus that offset),
never its corner, whose tile depends on float rounding.

## Files

| File | Holds |
| --- | --- |
| `main.go` | `main`, screen size, every color of the game |
| `play.go` | The play scene: input to actions plus one `Tick` per update; the camera, the selection, the marking blueprint and the open cards live here, never serialized |
| `radial.go` | The build menu: the radial of blueprints a click on empty ground opens |
| `state.go` | The simulation's state: robots (core or built), buildings, stock, what remains of each deposit, build jobs; `newGame`, which deals the starting region |
| `actions.go` | The actions (`Tick`, `SendRobot`, `RecallRobot`, `MarkBuilding`, `QueueRobot`) and `Apply`, the only door into the state |
| `sim_robots.go` | The robots' rules and tuning: what a robot does each tick — carry home, mind the tank, finish loading, oldest build job, own post, idle by the core |
| `sim_buildings.go` | The buildings' rules and tuning: blueprints' costs, placement and safe zones, storage caps, refuel spots, the factories' robot works |
| `region.go` | The hand-made 25x25 layout, the isometric `project`, tile helpers, the deposit patches flooded out of the layout; pure Go, no drawing |
| `things.go` | What a tile holds: `Thing` snapshots out of layout plus state (a deposit tile shows its whole patch), `tileAtWorld`, the SI quantities; pure Go, no drawing |
| `catalog.go` | The entity database: per thing type its name, color, unit and card lines, plus the stable-color fallback |
| `markup.go` | The `[name]...[/]` colored-text markup: parser and drawer |
| `inspect.go` | The inspection panel: layout, hit testing, painting, the cards' buttons; tile highlights |
| `draw.go` | The region painter: ground, buildings, robots, fog, bubbles, build-site wireframes, the marking ghost |
| `region_test.go` | Layout, projection, things, SI formatting, catalog tests |
| `markup_test.go` | Markup parser and tooltip layout/button tests |
| `world_test.go` | The simulation driven directly: starting robots, hauling, picking, priority, recall, dry deposits, determinism, JSON round trip |
| `buildings_test.go` | The buildings driven directly: marking pays and raises, the fog refuses ground, the factory's robots, refueling, digestion, the fog's drag, full stores and silos |

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
  tick what one does, in priority order (carry home, mind the tank,
  finish loading, oldest build job, own post, idle by the core).
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

Five blueprints (`BuildingKind` in `state.go`, rules and tuning in
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
the core trying again each tick (its card says waiting for storage), and
deposits what fits when a silo opens room.

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
pole is 10 m across and 14 m tall, and its bubble radius is 800 m. Oil is
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
hit-tests it (`contains`, `cardAt`, `buttonAt`) while `Draw` paints it.
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
silo opens room. Visual checks are shots with scripted clicks; see the
command above.
