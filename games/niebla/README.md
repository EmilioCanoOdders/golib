# niebla — technical notes

How the code is built. The game design lives in [DESIGN.md](DESIGN.md); this
file is for whoever works on the code, human or agent.

## Commands

From the GoLib repository root:

```text
./golib run niebla          # play it
./golib test niebla         # go vet + go test
./golib shot niebla 10 --input "Mouse@5:640,360 MouseLeft@6"
```

The shot above clicks the core tile at rest zoom: the screen center is
(640, 360), and the camera at rest centers the view on world point
(640, 372) — the two differ, `camera.ToWorld` does the mapping.

## Files

| File | Holds |
| --- | --- |
| `main.go` | `main`, screen size, every color of the game |
| `play.go` | The play scene: input to camera moves and inspection picks; the camera and the selection live here, never serialized |
| `region.go` | The hand-made 25x25 layout, the isometric `project`, tile helpers; pure Go, no drawing |
| `things.go` | What a tile holds: `Thing` snapshots from the layout, `tileAtWorld`, the SI quantities; pure Go, no drawing |
| `catalog.go` | The entity database: per thing type its name, color, unit and card lines, plus the stable-color fallback |
| `markup.go` | The `[name]...[/]` colored-text markup: parser and drawer |
| `inspect.go` | The inspection panel: layout, hit testing, painting; tile highlights |
| `draw.go` | The region painter: ground, fog, core, bubble, props |
| `region_test.go` | Layout, projection, things, SI formatting, catalog tests |
| `markup_test.go` | Markup parser and tooltip layout tests |

## Architecture

DESIGN.md's four laws (one serializable state, actions in/state out, the
game is a visualization, determinism) are the target; the simulation has not
landed yet. What exists today, and where it will plug in:

- The region is static data (`regionLayout`), read by everyone. When the
  simulation arrives, `thingsAt` stops reading the layout and reads the
  state; nothing else in the view layer changes.
- View state, never serialized: the camera, the picked tile, which cards
  stand open (`playScene.expanded`, keyed by `Thing.ID`).
- The catalog and the markup palette are view-side metadata, not state:
  package-level tables, like a schema.

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
is 5 m across (25 m²), the core's pole is 5 m across and 14 m tall, its
bubble radius is 20 m. Oil is liters, lilac is kilograms (`si` turns 1500 kg
into `1.5 t`, so nobody ever reads `kkg`). Amounts live at the top of
`things.go`.

### Tile picking

`tileAtWorld` undoes `project`: a tile's diamond on the screen is the square
`[col, col+1) x [row, row+1)` in tiles, so the inverse is exact, and a point
maps to its tile with two `math.Floor` calls. The inspection panel anchors
on the tile's projected corner through `camera.ToScreen`, so it follows the
tile while the view pans or zooms, and flips to the tile's left when it
would leave the screen.

### Inspection panel

One geometry, two users: `tooltipLayout` builds the row list, and `Update`
hit-tests it (`contains`, `cardAt`) while `Draw` paints it. Clicking a tile
selects it; clicking a card's title toggles its details; a right click that
never moved more than 4 px deselects (a drag is a pan, not a cancel).

## Testing

`region_test.go` pins the layout's placement rules, the projection's round
trip, the things a tile holds, the SI formatter and the catalog's stability.
`markup_test.go` covers the parser (nesting, unknown tags, unclosed color)
and the tooltip layout's rows and hit testing. Visual checks are shots with
scripted clicks; see the command above.
