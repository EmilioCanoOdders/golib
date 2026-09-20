package main

import (
	"fmt"
	"math"
)

// The simulation's state: one serializable value, the whole game. It
// holds no pointers, no channels, no functions, so it goes through
// golib.SaveData as it is, and loading it back loads the game, exactly.
// Everything the view adds for itself (the camera, the picked tile, the
// open cards) lives in the scenes, never here.

// State is the whole game as one value. Actions (actions.go) are the
// only thing that changes it.
type State struct {
	Ticks     int64              // game ticks, 60 to the second
	NextID    int64              // the ID the next robot or building gets
	Robots    map[int64]Robot    // the colony, by ID
	Buildings map[int64]Building // the colony's structures, by ID
	Stock     Stock              // what the core's stores hold
	Drain     map[string]float64 // what remains in each deposit patch, by key
	Jobs      []Job              // build jobs, oldest first
	Fog       Fog                // the region's weather: the cycles and the swell
}

// Fog is the region's weather, where the fog's breath has got to. The
// swell rises at a cycle's end and drains tick by tick; NextIn counts
// the cycles of calm left before the next one, and Swells remembers how
// many have passed, which is how the fog grows harder. SwellLeft > 0
// means a swell is up (sim_fog.go holds the law).
type Fog struct {
	Cycle     int64   // whole cycles since the region began
	CycleLeft int64   // ticks until the current cycle ends
	SwellLeft int64   // ticks left of the current swell; 0 is calm
	NextIn    float64 // cycles of calm left before the next swell
	Swells    int64   // swells that have passed: the difficulty's memory
}

// Stock is what the colony has stored at the core.
type Stock struct {
	Oil   float64 // liters
	Lilac float64 // kilograms
}

// RobotKind says where a robot comes from: the core's own slow workers,
// free and without a tank, or the ones the factory builds, which burn
// oil from a tank and are digested by the fog when it runs dry outside
// a bubble.
type RobotKind string

const (
	RobotCore  RobotKind = "core"
	RobotBuilt RobotKind = "built"
)

// Robot is one worker. It carries no plan: every tick the rules (see
// sim_robots.go) derive what it does from the state, so a save reproduces
// its future. Its post is one tile of a deposit patch, and working it
// drains the whole patch.
type Robot struct {
	ID        int64
	Kind      RobotKind // core or built
	X, Y      float64   // position, in units (1 u = 1 m)
	Tank      float64   // liters of oil left; core robots carry none
	PostCol   int       // the tile of the patch it was sent to; -1 when free
	PostRow   int       //
	WorkTicks int64     // ticks of loading left at its post
	Carry     float64   // what it carries, in the cargo's SI unit
	Cargo     ThingType // oil, lilac, or "" while empty
}

// BuildingKind names one of the structures the colony can raise. The
// blueprints' costs and the rules each kind obeys live in
// sim_buildings.go.
type BuildingKind string

const (
	BuildingFactory   BuildingKind = "factory"   // builds robots from lilac and oil
	BuildingCharger   BuildingKind = "charger"   // refills a built robot's tank
	BuildingSilo      BuildingKind = "silo"      // stores more oil
	BuildingWarehouse BuildingKind = "warehouse" // stores more lilac
	BuildingProtector BuildingKind = "protector" // a small bubble of safe ground
)

// Building is one raised structure. Its Col, Row are cell coordinates
// (the tile grid's last subdivision, a 40 u footprint — sim_buildings.go),
// so a tile may hold several buildings. Its Work counts down while a
// factory builds a robot; every other kind leaves it at zero.
type Building struct {
	ID       int64
	Kind     BuildingKind
	Col, Row int // the cell it stands on
	Work     int64
}

// Job is one build job: what to raise, on which cell, and the ticks of
// work it still asks for.
type Job struct {
	Kind     BuildingKind
	Col, Row int // the cell it stands on
	Left     int64
}

// hasPost reports whether the robot was sent to a deposit tile.
func (r Robot) hasPost() bool {
	return r.PostCol >= 0
}

func (r *Robot) clearPost() {
	r.PostCol, r.PostRow = -1, -1
}

// The core's gift: what its stores hold at the start, enough to mark
// the first buildings without a haul first.
const (
	startingStockOil   = 300.0 // L
	startingStockLilac = 600.0 // kg
)

// newGame deals the starting region: every deposit patch full, the
// starting robots idle by the core, and the core's gift in the stores.
func newGame() *State {
	s := &State{
		NextID:    1,
		Robots:    map[int64]Robot{},
		Buildings: map[int64]Building{},
		Drain:     map[string]float64{},
		Stock:     Stock{Oil: startingStockOil, Lilac: startingStockLilac},
		Fog:       Fog{CycleLeft: fogCycleTicks, NextIn: fogSwellPeriod},
	}
	for _, d := range regionDeposits {
		s.Drain[depositKey(d)] = depositFull(d)
	}
	for i := 0; i < startingRobots; i++ {
		x, y := parkSpot(s.NextID)
		s.spawnRobot(RobotCore, x, y)
	}
	return s
}

// spawnRobot adds one robot to the colony at a spot, with the tank full
// when it is a built one.
func (s *State) spawnRobot(kind RobotKind, x, y float64) {
	id := s.NextID
	s.NextID++
	tank := 0.0
	if kind == RobotBuilt {
		tank = robotTankLiters
	}
	s.Robots[id] = Robot{
		ID: id, Kind: kind, X: x, Y: y, Tank: tank,
		PostCol: -1, PostRow: -1,
	}
}

// raise turns a finished job into the building it asked for.
func (s *State) raise(kind BuildingKind, col, row int) {
	id := s.NextID
	s.NextID++
	s.Buildings[id] = Building{ID: id, Kind: kind, Col: col, Row: row}
}

// drainKey names a deposit tile inside State.Drain.
func drainKey(col, row int) string {
	return fmt.Sprintf("%d,%d", col, row)
}

// tileCenterUnits returns the middle of a tile, in world units.
func tileCenterUnits(col, row int) (x, y float64) {
	return float64(col)*unitsPerTile + unitsPerTile*0.5,
		float64(row)*unitsPerTile + unitsPerTile*0.5
}

// robotTile returns the tile a robot stands on, from its position.
func robotTile(r Robot) (col, row int) {
	return int(math.Floor(r.X / unitsPerTile)),
		int(math.Floor(r.Y / unitsPerTile))
}
