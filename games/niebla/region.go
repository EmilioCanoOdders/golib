package main

import "math"

// Tuning: the numbers that define the region, with units in the name.
const (
	regionCols = 25 // tiles across
	regionRows = 25 // tiles down

	tileW = 48 // tile width, in screen pixels
	tileH = 24 // tile height, in screen pixels

	// The world measures itself in units, and one unit is one meter
	// (things.go). A tile is 200 u across, so the region is 5 km from
	// side to side and a deposit patch of four tiles is 400 m aside.
	// Zoomed out the whole region fits on the screen as icons, zoomed in
	// a tile fills it. A robot is 6 u across, the core's pole 10 u, a
	// future building 40 u.
	unitsPerTile = 200

	coreCol = 12 // where the core sits
	coreRow = 12

	coreBubbleRadius = 4.0  // tiles; nothing is digested inside it
	fogLineRadius    = 10.5 // tiles; the fog's front stands here, for now
	fogFadeTiles     = 2.4  // tiles; how far the fog fades in past the line
)

// The size of one unit on the screen, before the camera zooms.
const (
	unitW = float32(tileW) / unitsPerTile
	unitH = float32(tileH) / unitsPerTile
)

// Where the region's top corner lands on the screen, so the whole diamond
// fits with a margin.
const (
	regionOriginX = screenWidth / 2
	regionOriginY = 72
)

// The tile kinds of the hand-made layout.
const (
	kindGround = '.'
	kindOil    = 'o'
	kindLilac  = 'L'
	kindCore   = 'C'
	kindRock   = 'r' // a decoration, drawn from zoom propZoom on
	kindBush   = 'b' // a decoration, drawn from zoom propZoom on
)

// regionLayout is the hand-made region, Tiled-informed, drawn in code for
// now: one rune per tile, ground everywhere but for the oil pools, the
// lilac veins and the core. Every row is regionCols runes wide.
//
// Deposits come in patches of four tiles, two by two. One oil pool and
// one lilac vein sit inside the bubble, off the core, so the colony can
// mine both in comfort whatever the fog does; the other four patches
// stand in the empty band near the region's edge, 1.5 to 2 km from the
// core, where the zoomed-out view shows them as icons and the zoomed-in
// view makes the walk between them long. Rocks and bushes dot the ground
// between bubble and edge, never inside the bubble.
var regionLayout = []string{
	".........................",
	".........................",
	".........................",
	"...........b.............",
	".........b.....b.........",
	".................r.......",
	".....oo..........oo......",
	".....oo..........oo......",
	"....r....................",
	"....................b....",
	"..............LL.........",
	"..............LL.........",
	"............C............",
	"...b.....................",
	"..........oo.........r...",
	".....b....oo.............",
	"................r........",
	"........b.........LL.....",
	".....LL...........LL.....",
	".....LL...r..............",
	"........r....b...........",
	"..............r..........",
	".........................",
	".........................",
	".........................",
}

// tileAt returns the kind of the tile at a column and row. Outside the
// region there is only ground, which the fog covers anyway.
func tileAt(col, row int) byte {
	if col < 0 || row < 0 || col >= regionCols || row >= regionRows {
		return kindGround
	}
	return regionLayout[row][col]
}

// tileDistance returns how far a tile is from the core, in tiles.
func tileDistance(col, row int) float32 {
	return float32(math.Hypot(float64(col-coreCol), float64(row-coreRow)))
}

// project returns where the world point at x, y units lands on the screen,
// as DESIGN.md says: screenX = (x-y)*unitW/2, screenY = (x+y)*unitH/2. The
// coordinates may be fractional, for circles around the core.
func project(x, y float32) (sx, sy float32) {
	return (x-y)*unitW/2 + regionOriginX, (x+y)*unitH/2 + regionOriginY
}

// projectTile returns where a tile's top corner lands on the screen.
func projectTile(col, row float32) (x, y float32) {
	return project(col*unitsPerTile, row*unitsPerTile)
}

// jitter returns a stable number from a tile's place and a salt, from -0.5
// to 0.5, so props sit off the middle of their tile without anything stored:
// the same tile always draws the same way.
func jitter(col, row, salt int) float32 {
	h := col*374761393 + row*668265263 + salt*1442695041
	h ^= h >> 13
	h *= 1274126177
	h ^= h >> 16
	return float32(h&0xffff)/float32(0xffff) - 0.5
}

// ellipseSemiAxes returns the half width and half height on the screen of a
// world circle of a radius in tiles, which the projection flattens.
func ellipseSemiAxes(radius float32) (halfW, halfH float32) {
	k := radius * float32(math.Sqrt2) / 2
	return k * tileW, k * tileH
}

// Deposit is one vein or pool: a patch of deposit tiles the colony treats
// as a single thing, however many tiles it spans. One robot works a whole
// patch, one card describes it, and what remains of it lives in the state
// under one key, the bounding box's top corner tile.
type Deposit struct {
	Kind       byte // kindOil or kindLilac
	Col, Row   int  // the bounding box's top corner tile
	Cols, Rows int  // its extent, in tiles
} // regionDeposits is every patch of the layout, found once by flooding
// connected deposit tiles. Like the layout itself, it is static data the
// view and the rules read freely; it never enters the state.
var regionDeposits = findDeposits()

// depositIndex maps each deposit tile to its patch's place in
// regionDeposits: the runtime lookup the tile grid can't give in O(1).
var depositIndex = buildDepositIndex()

// findDeposits floods the layout for connected patches of oil or lilac.
func findDeposits() []Deposit {
	seen := map[[2]int]bool{}
	var deposits []Deposit
	for row := 0; row < regionRows; row++ {
		for col := 0; col < regionCols; col++ {
			kind := tileAt(col, row)
			if kind != kindOil && kind != kindLilac || seen[[2]int{col, row}] {
				continue
			}
			d := Deposit{Kind: kind, Col: col, Row: row, Cols: 1, Rows: 1}
			queue := [][2]int{{col, row}}
			seen[[2]int{col, row}] = true
			for len(queue) > 0 {
				t := queue[0]
				queue = queue[1:]
				if t[0] < d.Col {
					d.Cols += d.Col - t[0]
					d.Col = t[0]
				}
				if t[0] > d.Col+d.Cols-1 {
					d.Cols += t[0] - (d.Col + d.Cols - 1)
				}
				if t[1] < d.Row {
					d.Rows += d.Row - t[1]
					d.Row = t[1]
				}
				if t[1] > d.Row+d.Rows-1 {
					d.Rows += t[1] - (d.Row + d.Rows - 1)
				}
				for _, n := range [4][2]int{
					{t[0] + 1, t[1]},
					{t[0] - 1, t[1]},
					{t[0], t[1] + 1},
					{t[0], t[1] - 1},
				} {
					if tileAt(n[0], n[1]) == kind && !seen[n] {
						seen[n] = true
						queue = append(queue, n)
					}
				}
			}
			deposits = append(deposits, d)
		}
	}
	return deposits
}

func buildDepositIndex() map[[2]int]int {
	index := make(map[[2]int]int, regionCols*regionRows)
	for i, d := range regionDeposits {
		for row := d.Row; row < d.Row+d.Rows; row++ {
			for col := d.Col; col < d.Col+d.Cols; col++ {
				index[[2]int{col, row}] = i
			}
		}
	}
	return index
}

// depositAt returns the patch a tile belongs to, and whether it holds one.
func depositAt(col, row int) (Deposit, bool) {
	i, ok := depositIndex[[2]int{col, row}]
	if !ok {
		return Deposit{}, false
	}
	return regionDeposits[i], true
}

// depositKey names a patch's remaining amount inside State.Drain.
func depositKey(d Deposit) string {
	return drainKey(d.Col, d.Row)
}

// depositFull returns what a patch holds at first, in its SI unit: its
// per-tile density times its tiles.
func depositFull(d Deposit) float64 {
	tiles := float64(d.Cols * d.Rows)
	if d.Kind == kindOil {
		return tiles * oilPerPoolTile
	}
	return tiles * lilacPerVeinTile
}
