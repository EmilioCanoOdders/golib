package main

import "math"

// Tuning: the numbers that define the region, with units in the name.
const (
	regionCols = 25 // tiles across
	regionRows = 25 // tiles down

	tileW = 48 // tile width, in screen pixels
	tileH = 24 // tile height, in screen pixels

	// The world measures itself in units: a robot is 1 u, the core's pole
	// is 5 u across, a typical building covers 10 u, and a vein reaches
	// 30 u. Five units to the tile lands the pole on one tile and a
	// building on two by two.
	unitsPerTile = 5

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
// now: one rune per tile, ground everywhere but for the oil pools, the lilac
// veins and the core. Every row is regionCols runes wide.
var regionLayout = []string{
	".........................",
	".........................",
	".........................",
	".........................",
	".................r.......",
	"......b.........ooo......",
	"......L.........ooo......",
	".....r.L.........r.......",
	"....b...L...L............",
	".......b....L....b.......",
	"...r........L........r...",
	".......b....L....b.......",
	"...LL...ooooCoooo........",
	".......b....L.....b......",
	"...r........L........r...",
	".....ooo....L...b........",
	"....rooo....L.....oob....",
	".......b........L.ob.....",
	"........rb.......L.......",
	"......b.....r.....b......",
	".........................",
	".........................",
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
