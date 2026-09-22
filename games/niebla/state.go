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
	Seed       int64              // the region's seed: its relief, cover and deposits (worldgen.go)
	Ticks      int64              // game ticks, 60 to the second
	NextID     int64              // the ID the next robot, building, pile or pipe gets
	Robots     map[int64]Robot    // the colony, by ID
	Buildings  map[int64]Building // the colony's structures, by ID
	Stock      Stock              // what the core's stores hold
	Drain      map[string]float64 // what remains in each deposit patch, by key
	Jobs       []Job              // build jobs, oldest first
	Piles      map[int64]Pile     // loose items on the ground, by ID
	Pipes      map[int64]Pipe     // oil pipes, laid or being laid, by ID
	Fog        Fog                // the region's weather: the cycles and the swell
	Enemies    map[int64]Enemy    // the rivals' vehicles, by ID (sim_enemies.go)
	Parties    map[int64]Party    // the rivals' visits under way, by ID
	Raids      Raids              // the rivals' clock: the visits that were, the next one
	Marks      map[int64]Mark     // what the scouts painted on the ground, by ID
	Reports    []Report           // the news the rivals made, oldest first
	Rolls      uint64             // random numbers drawn so far: the state's own PRNG
	Deliveries int64              // loads the robots have brought home, the first of which brings the first schematics in
	Squads     map[int64]Squad    // the squads' orders, by their war factory's ID
	Shots      map[int64]Shot     // bullets and shells in the air, by ID (sim_shots.go)
	Tech       map[string]bool    // the schematics that arrived: drop ID -> opened (sim_tech.go)
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
	Held      bool    // the dev tools hold the swell up: it doesn't drain
	Pressure  float64 // 0 calm to 1 pressed in whole: how far the swell has come
}

// Stock is what the colony has stored. Lilac is one stock under every
// roof; oil has a place, and this is the core's own tank (sim_oil.go).
type Stock struct {
	Oil   float64 // liters, in the core's tank
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
	// RobotCombat is a trooper, a war factory's: a built robot on a combat
	// chassis, which does no work and follows its squad (sim_squads.go).
	RobotCombat RobotKind = "combat"
)

// Robot is one worker. It carries no plan: every tick the rules (see
// sim_robots.go) derive what it does from the state, so a save reproduces
// its future. What it claims - a section of pipe to lay - is state too,
// since the other robots read it. Its post is one tile of a deposit patch, and working it
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
	Pile      int64     // the pile it is loading from; 0 while loading at its post
	Pipe      int64     // the pipe whose section it claimed to lay; 0 with no claim
	Section   int64     // the claimed section, from the pipe's source out
	Squad     int64     // troopers: the war factory whose squad it is in
	Health    float64   // troopers: what is left of it
	Reload    int64     // troopers: ticks until the next shot
	Aim       int64     // troopers: the vehicle the last shot went to
}

// tanked reports whether the robot runs on a tank of oil: every robot
// but the core's own.
func (r Robot) tanked() bool {
	return r.Kind != RobotCore
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
	BuildingPump      BuildingKind = "pump"      // draws a pool's oil into a pipe
	BuildingGuard     BuildingKind = "guard"     // shoots the rivals in its reach

	BuildingWarFactory BuildingKind = "warfactory" // builds troopers, its squad
	BuildingArtillery  BuildingKind = "artillery"  // shells the rivals the colony sees
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
	Oil      float64 // liters in its tank: silos and chargers (sim_oil.go)
	Reload   int64   // guard posts: ticks until the next shot (sim_enemies.go)
	Aim      int64   // guard posts: the vehicle the last shot went to
	Damage   float64 // what it has taken; at its health it falls (sim_shots.go)
}

// Job is one build job: what to raise, on which cell, and the ticks of
// work it still asks for.
type Job struct {
	Kind     BuildingKind
	Col, Row int // the cell it stands on
	Left     int64
}

// Pile is what a demolished building leaves on its cell: a container
// that stands for every loose item lying there. It has no mass, no
// health and no capacity, and it leaves the state with its last item
// (sim_piles.go holds the law).
type Pile struct {
	ID       int64
	Col, Row int     // the cell it lies on
	Oil      float64 // liters
	Lilac    float64 // kilograms
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

// newGame deals the starting region on the default seed.
func newGame() *State {
	return newGameOn(defaultSeed)
}

// newGameOn deals the starting region a seed generates: every deposit
// full, the starting robots idle by the core, and the core's gift in the
// stores.
func newGameOn(seed int64) *State {
	useRegion(seed)
	s := &State{
		Seed:      seed,
		NextID:    1,
		Robots:    map[int64]Robot{},
		Buildings: map[int64]Building{},
		Piles:     map[int64]Pile{},
		Pipes:     map[int64]Pipe{},
		Enemies:   map[int64]Enemy{},
		Parties:   map[int64]Party{},
		Marks:     map[int64]Mark{},
		Squads:    map[int64]Squad{},
		Shots:     map[int64]Shot{},
		Raids:     Raids{NextAt: raidFirstScoutTicks},
		Drain:     map[string]float64{},
		Stock:     Stock{Oil: startingStockOil, Lilac: startingStockLilac},
		Fog:       Fog{CycleLeft: fogCycleTicks, NextIn: fogSwellPeriod},
	}
	for _, d := range land.deposits {
		s.Drain[depositKey(d)] = depositFull(d)
	}
	for i := 0; i < startingRobots; i++ {
		x, y := parkSlot(i)
		s.spawnRobot(RobotCore, x, y)
	}
	return s
}

// enterRegion makes the ground the state's own, for a state that comes
// from a save. A save from before the generated regions names deposits
// that are gone: the ones it doesn't know wake up full.
func (s *State) enterRegion() {
	useRegion(s.Seed)
	if s.Drain == nil {
		s.Drain = map[string]float64{}
	}
	for _, d := range land.deposits {
		if _, known := s.Drain[depositKey(d)]; !known {
			s.Drain[depositKey(d)] = depositFull(d)
		}
	}
}

// spawnRobot adds one robot to the colony at a spot, with the tank full
// when it has one, and returns its ID.
func (s *State) spawnRobot(kind RobotKind, x, y float64) int64 {
	id := s.NextID
	s.NextID++
	r := Robot{ID: id, Kind: kind, X: x, Y: y, PostCol: -1, PostRow: -1}
	if r.tanked() {
		r.Tank = robotTankLiters
	}
	if kind == RobotCombat {
		r.Health = trooperHealth
	}
	s.Robots[id] = r
	return id
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

// robotCell returns the cell a robot stands on, from its position.
func robotCell(r Robot) (col, row int) {
	return int(math.Floor(r.X / buildingCell)),
		int(math.Floor(r.Y / buildingCell))
}
