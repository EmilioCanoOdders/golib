package main

import (
	"fmt"
	"math"
)

// The world speaks SI: one unit of the world is one meter, so a tile is
// 200 m across, oil is measured in liters and lilac in kilograms. The
// per-tile amounts are the deposits' density: a patch of four tiles holds
// four of them (a whole vein, 12 t of lilac).
const (
	unitMeters = 1 // one world unit, in m

	oilPerPoolTile   = 900  // liters of oil in one pool tile
	lilacPerVeinTile = 3000 // kilograms of lilac in one vein tile

	corePoleAcross = 10 // m across the core's pole
	coreHeight     = 14 // m of pole above the ground
)

// coreBubbleMeters returns the core's bubble radius in meters.
func coreBubbleMeters() float64 {
	return float64(coreBubbleRadius * unitsPerTile * unitMeters)
}

// protectorBubbleMeters returns a shadow protector's bubble radius in
// meters.
func protectorBubbleMeters() float64 {
	return float64(protectorBubbleTiles * unitsPerTile * unitMeters)
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
	TypeRobot ThingType = "robot"

	TypeFactory   ThingType = "factory"
	TypeCharger   ThingType = "charger"
	TypeSilo      ThingType = "silo"
	TypeWarehouse ThingType = "warehouse"
	TypeProtector ThingType = "protector"
)

// buildingType maps a building kind to the thing type its cards read.
func buildingType(kind BuildingKind) ThingType {
	switch kind {
	case BuildingFactory:
		return TypeFactory
	case BuildingCharger:
		return TypeCharger
	case BuildingSilo:
		return TypeSilo
	case BuildingWarehouse:
		return TypeWarehouse
	case BuildingProtector:
		return TypeProtector
	}
	return TypeRobot
}

// Detail is one line of a thing's expanded card: a label and a value. The
// value may carry markup, such as "[oil]900 L[/]".
type Detail struct {
	Label string
	Value string
}

// Thing is one thing the world holds, seen from a tile: the view-side
// snapshot the cards render. It carries no behavior; what it means and
// how it reads live in the catalog. Robots come from the state; a
// deposit's amount is what remains of its whole patch in it.
type Thing struct {
	Type    ThingType
	ID      string  // stable in the region, so views can remember it
	Amount  float64 // the type's headline quantity, in its SI unit
	Caption string  // a headline that replaces the summary, a robot's task
	Ref     int64   // the entity's ID in the state, for robots
	Cols    int     // deposits: the patch's extent, in tiles
	Rows    int     //
}

// thingsAt returns the things standing on a tile: its building or deposit
// first, then the robots on it by ID. A deposit tile shows its whole
// patch's card: the vein is one thing, however many tiles it spans.
// Rocks and bushes are decoration, so they have no card yet.
func thingsAt(s *State, col, row int) []Thing {
	var things []Thing
	switch tileAt(col, row) {
	case kindOil, kindLilac:
		d, _ := depositAt(col, row)
		thing := depositThing(d)
		thing.Amount = remainingAt(s, col, row)
		things = append(things, thing)
	case kindCore:
		things = append(things, coreAt(col, row))
	}
	for _, b := range buildingsOnTile(s, col, row) {
		things = append(things, buildingThing(b))
	}
	for _, r := range robotsOnTile(s, col, row) {
		things = append(things, robotThing(s, r))
	}
	return things
}

// robotsOnTile returns the robots standing on a tile, in ID order.
func robotsOnTile(s *State, col, row int) []Robot {
	var found []Robot
	for _, id := range sortedRobotIDs(s) {
		if r := s.Robots[id]; r.standsOn(col, row) {
			found = append(found, r)
		}
	}
	return found
}

// depositThing is a patch's card: named for its kind, ID'd by its top
// corner tile, carrying its extent so the card can say how much ground
// the patch covers.
func depositThing(d Deposit) Thing {
	kind := TypeOil
	if d.Kind == kindLilac {
		kind = TypeLilac
	}
	return Thing{
		Type: kind,
		ID:   fmt.Sprintf("%s@%d,%d", kind, d.Col, d.Row),
		Cols: d.Cols,
		Rows: d.Rows,
	}
}

func coreAt(col, row int) Thing {
	return Thing{
		Type:   TypeCore,
		ID:     fmt.Sprintf("core@%d,%d", col, row),
		Amount: coreBubbleMeters(),
	}
}

// buildingThing is a raised building's card, ID'd by its entity ID so it
// stays the same card across saves. The catalog says what each kind's
// card reads; the factory's caption counts down its robot when busy.
func buildingThing(b Building) Thing {
	kind := buildingType(b.Kind)
	thing := Thing{
		Type: kind,
		ID:   fmt.Sprintf("%s-%d", kind, b.ID),
		Ref:  b.ID,
	}
	if b.Kind == BuildingFactory && b.Work > 0 {
		thing.Caption = fmt.Sprintf("building robot, %d s", b.Work/60)
	}
	return thing
}

// standsOn reports whether the robot's position falls on this tile.
func (r Robot) standsOn(col, row int) bool {
	c, rw := robotTile(r)
	return c == col && rw == row
}

func robotThing(s *State, r Robot) Thing {
	return Thing{
		Type:    TypeRobot,
		ID:      fmt.Sprintf("robot-%d", r.ID),
		Ref:     r.ID,
		Caption: robotCaption(s, r),
	}
}

// robotCaption is the robot card's headline: what it is up to. It reads
// the state in the same priority order the rules do (sim_robots.go), so
// the words always say what the robot is doing.
func robotCaption(s *State, r Robot) string {
	_, hasJob := oldestJob(s)
	switch {
	case r.Carry > 0:
		if word := storageWord(s, r); word != "" {
			return word
		}
		return "hauling " + cargoWord(r.Cargo)
	case chargeStatus(s, r) != "":
		return chargeStatus(s, r)
	case r.WorkTicks > 0:
		return "loading " + postWord(r)
	case hasJob:
		return "building"
	case r.hasPost():
		return postWord(r) + " run"
	default:
		return "idle"
	}
}

// storageWord names a robot standing at the core with a full store and
// cargo in its arms.
func storageWord(s *State, r Robot) string {
	if r.Carry <= 0 {
		return ""
	}
	x, y := parkSpot(r.ID)
	if math.Hypot(r.X-x, r.Y-y) >= 0.5 {
		return ""
	}
	full := oilCap(s) - s.Stock.Oil
	if r.Cargo == TypeLilac {
		full = lilacCap(s) - s.Stock.Lilac
	}
	if full < 0.0001 {
		return "waiting for storage"
	}
	return ""
}

func postWord(r Robot) string {
	if tileAt(r.PostCol, r.PostRow) == kindOil {
		return "oil"
	}
	return "lilac"
}

func cargoWord(cargo ThingType) string {
	if cargo == TypeOil {
		return "oil"
	}
	return "lilac"
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
