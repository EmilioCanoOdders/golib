package main

import (
	"fmt"
	"math"
)

// The world speaks SI: one unit of the world is one meter, so a tile is
// 5 m across, oil is measured in liters and lilac in kilograms.
const (
	unitMeters = 1 // one world unit, in m

	oilPerPoolTile   = 900 // liters of oil in one pool tile
	lilacPerVeinTile = 300 // kilograms of lilac in one vein tile

	corePoleAcross = unitsPerTile * unitMeters // m across the core's pole
	coreHeight     = 14                        // m of pole above the ground
)

// tileAreaLabel says how much ground one tile is, in SI.
var tileAreaLabel = fmt.Sprintf("%d by %d m",
	unitsPerTile*unitMeters, unitsPerTile*unitMeters)

// coreBubbleMeters returns the core's bubble radius in meters.
func coreBubbleMeters() float64 {
	return float64(coreBubbleRadius * unitsPerTile * unitMeters)
}

// ThingType names a kind of thing the world can hold: a resource in the
// ground, a building, a robot. The catalog (catalog.go) holds what each
// type is called and the color it is written in.
type ThingType string

// The thing types of the region so far.
const (
	TypeOil   ThingType = "oil"
	TypeLilac ThingType = "lilac"
	TypeCore  ThingType = "core"
)

// Detail is one line of a thing's expanded card: a label and a value. The
// value may carry markup, such as "[oil]900 L[/]".
type Detail struct {
	Label string
	Value string
}

// Thing is one thing the world holds, seen from a tile: the view-side
// snapshot the cards render. It carries no behavior; what it means and how
// it reads live in the catalog. When the simulation arrives, this comes
// from the state instead of the layout.
type Thing struct {
	Type   ThingType
	ID     string  // stable in the region, so views can remember it
	Amount float64 // the type's headline quantity, in its SI unit
}

// thingsAt returns the things standing on a tile, in a stable order. Rocks
// and bushes are decoration, so they have no card yet.
func thingsAt(col, row int) []Thing {
	switch tileAt(col, row) {
	case kindOil:
		return []Thing{oilAt(col, row)}
	case kindLilac:
		return []Thing{lilacAt(col, row)}
	case kindCore:
		return []Thing{coreAt(col, row)}
	}
	return nil
}

func oilAt(col, row int) Thing {
	return Thing{
		Type:   TypeOil,
		ID:     fmt.Sprintf("oil@%d,%d", col, row),
		Amount: oilPerPoolTile,
	}
}

func lilacAt(col, row int) Thing {
	return Thing{
		Type:   TypeLilac,
		ID:     fmt.Sprintf("lilac@%d,%d", col, row),
		Amount: lilacPerVeinTile,
	}
}

func coreAt(col, row int) Thing {
	return Thing{
		Type:   TypeCore,
		ID:     fmt.Sprintf("core@%d,%d", col, row),
		Amount: coreBubbleMeters(),
	}
}

// tileAtWorld returns the tile under the world point at x, y units, and
// whether the point is inside the region. It undoes project: a tile's
// diamond on the screen is the square [col, col+1) by [row, row+1) in tiles.
func tileAtWorld(x, y float32) (col, row int, inside bool) {
	a := (x - regionOriginX) / (unitW / 2)
	b := (y - regionOriginY) / (unitH / 2)
	worldX := (a + b) / 2 / unitsPerTile
	worldY := (b - a) / 2 / unitsPerTile
	col, row = int(math.Floor(float64(worldX))), int(math.Floor(float64(worldY)))
	return col, row, col >= 0 && row >= 0 && col < regionCols && row < regionRows
}

// si writes a quantity in its SI unit: whole numbers plainly, fractions
// with one decimal, thousands with the k prefix. Kilograms that grow past
// a thousand become tonnes, so no one reads "kkg".
func si(value float64, unit string) string {
	switch {
	case unit == "kg" && value >= 1000:
		return fmt.Sprintf("%.1f t", value/1000)
	case value >= 1000:
		return fmt.Sprintf("%.1f k%s", value/1000, unit)
	case value == math.Trunc(value):
		return fmt.Sprintf("%.0f %s", value, unit)
	default:
		return fmt.Sprintf("%.1f %s", value, unit)
	}
}
