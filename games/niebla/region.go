package main

import "math"

// Tuning: the numbers that define the region, with units in the name.
const (
	regionCols = 25 // tiles across
	regionRows = 25 // tiles down

	tileW = 48 // tile width, in screen pixels
	tileH = 24 // tile height, in screen pixels

	coreCol = 12 // where the core sits
	coreRow = 12

	coreBubbleRadius = 4.0 // tiles; nothing is digested inside it
	fogLineRadius    = 10.5 // tiles; the fog's front stands here, for now
	fogFadeTiles     = 2.4  // tiles; how far the fog fades in past the line
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
)

// regionLayout is the hand-made region, Tiled-informed, drawn in code for
// now: one rune per tile, ground everywhere but for the oil pools, the lilac
// veins and the core. Every row is regionCols runes wide.
var regionLayout = []string{
	".........................",
	".........................",
	".........................",
	".........................",
	".........................",
	"................ooo......",
	"......L.........ooo......",
	".......L.................",
	"........L...L............",
	"............L............",
	"............L............",
	"............L............",
	"...LL...ooooCoooo........",
	"............L............",
	"............L............",
	".....ooo....L............",
	".....ooo....L.....oo.....",
	"................L.o......",
	".................L.......",
	".........................",
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

// project returns where the top corner of a tile's diamond lands on the
// screen, as DESIGN.md says: screenX = (x-y)*tileW/2, screenY = (x+y)*tileH/2.
// The coordinates may be fractional, for circles around the core.
func project(col, row float32) (x, y float32) {
	return (col-row)*tileW/2 + regionOriginX, (col+row)*tileH/2 + regionOriginY
}

// ellipseSemiAxes returns the half width and half height on the screen of a
// world circle of a radius in tiles, which the projection flattens.
func ellipseSemiAxes(radius float32) (halfW, halfH float32) {
	k := radius * float32(math.Sqrt2) / 2
	return k * tileW, k * tileH
}
