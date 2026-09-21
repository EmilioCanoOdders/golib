package main

import (
	"math"
	"sort"
)

// Tuning: what the buildings cost, what they hold and what the fog
// does, with units in the name. A building is 40 u across, a tenth of a
// tile, so it reads as an icon when the view is far out.
const (
	// Raising a building, paid from the stores the moment it is marked.
	// Every building asks lilac; those that plug into the oil grid also
	// ask a little oil.
	factoryCostLilac   = 200.0 // kg
	chargerCostLilac   = 120.0 // kg
	chargerCostOil     = 40.0  // L
	siloCostLilac      = 100.0 // kg
	warehouseCostLilac = 100.0 // kg
	protectorCostLilac = 180.0 // kg
	protectorCostOil   = 40.0  // L
	pumpCostLilac      = 150.0 // kg

	buildingWorkTicks = 600 // ticks of robot work to raise any building: 10 s

	// The factory's robots: lilac and oil apiece, 12 s of work each.
	robotCostLilac    = 40.0 // kg
	robotCostOil      = 30.0 // L
	factoryRobotTicks = 720  // ticks to build one robot

	// Storage: the core's own stores, and what each building adds.
	coreOilCap        = 1000.0 // L
	coreLilacCap      = 2500.0 // kg
	siloOilCap        = 1000.0 // L apiece
	warehouseLilacCap = 4000.0 // kg apiece

	// The tank of a built robot. It burns oil walking, hauling, loading
	// and idling alike; low, it walks to the nearest charger or the core
	// to refill from the stores; empty outside a bubble, the fog digests
	// it.
	robotTankLiters     = 120.0 // L
	robotBurnPerSecond  = 0.25  // L/s
	robotLowTankAt      = 0.25  // tank fraction that sends it to refuel
	chargerRefillPerSec = 20.0  // L/s drawn from the stores

	// The shadow protector's bubble: a small disc of safe ground, like
	// the core's but without the pole's upkeep.
	protectorBubbleTiles = 2.0 // tiles
)

// fogSpeedFactor is how much of a robot's speed is left deep in the fog:
// it slows every robot, core or built, and the bubbles cancel it.
const fogSpeedFactor = 0.5

// The ground divides past its tiles: a tile of 200 u holds 8 by 8 cells
// of `buildingCell` u on a side, and one cell is the footprint of the
// smallest building - the last subdivision of the casilla, four robots
// wide and a bit more. A building's Col, Row in the state are cell
// coordinates, and a tile may hold several buildings on different cells.
const (
	buildingCell = 25 // u on a side: 8 by 8 cells to a tile

	regionCellCols = regionCols * unitsPerTile / buildingCell // 200
	regionCellRows = regionRows * unitsPerTile / buildingCell //
)

// cellCenterUnits returns the middle of a cell, in world units.
func cellCenterUnits(col, row int) (x, y float64) {
	return float64(col)*buildingCell + buildingCell*0.5,
		float64(row)*buildingCell + buildingCell*0.5
}

// cellTile returns the tile a cell belongs to.
func cellTile(col, row int) (tcol, trow int) {
	return col * buildingCell / unitsPerTile,
		row * buildingCell / unitsPerTile
}

// cellAtWorld returns the cell under the world point at x, y units, and
// whether it is inside the region. It undoes project and floors onto the
// cell grid, the way tileAtWorld does onto tiles.
func cellAtWorld(x, y float64) (col, row int, inside bool) {
	worldX, worldY := unitsAtWorld(x, y)
	col, row = int(math.Floor(worldX/buildingCell)), int(math.Floor(worldY/buildingCell))
	return col, row, col >= 0 && row >= 0 && col < regionCellCols && row < regionCellRows
}

// buildingCost returns what raising a kind asks for, in lilac and oil.
func buildingCost(kind BuildingKind) (lilac, oil float64) {
	switch kind {
	case BuildingFactory:
		return factoryCostLilac, 0
	case BuildingCharger:
		return chargerCostLilac, chargerCostOil
	case BuildingSilo:
		return siloCostLilac, 0
	case BuildingWarehouse:
		return warehouseCostLilac, 0
	case BuildingProtector:
		return protectorCostLilac, protectorCostOil
	case BuildingPump:
		return pumpCostLilac, 0
	}
	return 0, 0
}

// canAfford reports whether the stores can pay a blueprint's cost.
func canAfford(s *State, kind BuildingKind) bool {
	lilac, oil := buildingCost(kind)
	return s.Stock.Lilac >= lilac && oilTotal(s) >= oil
}

// sortedBuildingIDs lists the buildings' IDs in order, so nothing ever
// depends on map iteration.
func sortedBuildingIDs(s *State) []int64 {
	ids := make([]int64, 0, len(s.Buildings))
	for id := range s.Buildings {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// buildingAt returns the building standing on a cell, if any.
func buildingAt(s *State, col, row int) (Building, bool) {
	for _, id := range sortedBuildingIDs(s) {
		if b := s.Buildings[id]; b.Col == col && b.Row == row {
			return b, true
		}
	}
	return Building{}, false
}

// buildingsOnTile returns every building standing on a tile, in ID
// order: cells subdivide the tile, and several buildings may share it.
func buildingsOnTile(s *State, tcol, trow int) []Building {
	var found []Building
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		if ccol, crow := cellTile(b.Col, b.Row); ccol == tcol && crow == trow {
			found = append(found, b)
		}
	}
	return found
}

// canPlace reports whether a kind may be marked on a cell: buildable
// ground, no building there yet, and — the fog's law — inside a safe
// bubble, which the protector alone is built to stand outside of. A pump
// is the one kind that stands on oil instead: on a pool with oil left in
// it and no pump yet.
func canPlace(s *State, kind BuildingKind, col, row int) bool {
	if col < 0 || row < 0 || col >= regionCellCols || row >= regionCellRows {
		return false
	}
	tcol, trow := cellTile(col, row)
	ground := byte(kindGround)
	if kind == BuildingPump {
		ground = kindOil
	}
	if tileAt(tcol, trow) != ground {
		return false
	}
	if kind == BuildingPump &&
		(patchPumped(s, tcol, trow) || remainingAt(s, tcol, trow) <= 0) {
		return false
	}
	if _, occupied := buildingAt(s, col, row); occupied {
		return false
	}
	for _, job := range s.Jobs {
		if job.Col == col && job.Row == row {
			return false // a job already raises something here
		}
	}
	if _, littered := pileAt(s, col, row); littered {
		return false // loose items hold the cell until they are hauled off
	}
	x, y := cellCenterUnits(col, row)
	return kind == BuildingProtector || inSafeZone(s, x, y)
}

// inSafeZone reports whether a world point stands inside a bubble: the
// core's, or a shadow protector's. Nothing is digested inside one, and
// the fog does not slow whoever walks there.
func inSafeZone(s *State, x, y float64) bool {
	return shelteredWithout(s, x, y, 0)
}

// shelteredWithout reports whether a world point would stand inside a
// bubble if one building were gone: what demolishing a protector asks.
func shelteredWithout(s *State, x, y float64, gone int64) bool {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	if math.Hypot(x-cx, y-cy) <= coreBubbleRadius*unitsPerTile {
		return true
	}
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		if b.Kind != BuildingProtector || b.ID == gone {
			continue
		}
		bx, by := cellCenterUnits(b.Col, b.Row)
		if math.Hypot(x-bx, y-by) <= protectorBubbleTiles*unitsPerTile {
			return true
		}
	}
	return false
}

// fogAt returns how much fog sits on a world point, from 0 to 1: none
// inside a bubble, then the same fade the view paints past the line -
// the line of now, pressed in while a swell is up.
func fogAt(s *State, x, y float64) float64 {
	if inSafeZone(s, x, y) {
		return 0
	}
	return float64(fogCover(fogDistanceAt(x, y), fogLineNow(s)))
}

// lilacCap returns what the colony's stores hold at most in lilac: the
// core's own room, plus every warehouse. Oil's tanks are in sim_oil.go.
func lilacCap(s *State) float64 {
	cap := coreLilacCap
	for _, id := range sortedBuildingIDs(s) {
		if s.Buildings[id].Kind == BuildingWarehouse {
			cap += warehouseLilacCap
		}
	}
	return cap
}

// refuelSpot returns where a robot goes to refill its tank: the middle
// of its refuelTank.
func refuelSpot(s *State, r Robot) (x, y float64) {
	spot, _ := tankSpot(s, refuelTank(s, r))
	return spot.X, spot.Y
}

// stepFactories moves every factory's robot build one tick forward; done,
// the new robot rolls out of the works with its tank full.
func stepFactories(s *State) {
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		if b.Kind != BuildingFactory || b.Work <= 0 {
			continue
		}
		b.Work--
		s.Buildings[id] = b
		if b.Work == 0 {
			x, y := cellCenterUnits(b.Col, b.Row)
			s.spawnRobot(RobotBuilt, x, y)
		}
	}
}
