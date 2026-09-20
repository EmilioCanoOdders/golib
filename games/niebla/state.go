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
	Ticks  int64              // game ticks, 60 to the second
	NextID int64              // the ID the next robot gets
	Robots map[int64]Robot    // the colony, by ID
	Stock  Stock              // what the core's stores hold
	Drain  map[string]float64 // what remains in each deposit patch, by key
	Jobs   []Job              // build jobs, oldest first; nothing marks them yet
}

// Stock is what the colony has stored at the core.
type Stock struct {
	Oil   float64 // liters
	Lilac float64 // kilograms
}

// Robot is one worker. It carries no plan: every tick the rules (see
// sim_robots.go) derive what it does from the state, so a save reproduces
// its future. Its post is one tile of a deposit patch, and working it
// drains the whole patch.
type Robot struct {
	ID        int64
	X, Y      float64   // position, in units (1 u = 1 m)
	PostCol   int       // the tile of the patch it was sent to; -1 when free
	PostRow   int       //
	WorkTicks int64     // ticks of loading left at its post
	Carry     float64   // what it carries, in the cargo's SI unit
	Cargo     ThingType // oil, lilac, or "" while empty
}

// Job is one build job: a tile and the ticks of work it still asks for.
// Constructions don't exist yet, so nothing marks jobs, but the robots'
// priority over them is already law (sim_robots.go).
type Job struct {
	Col  int
	Row  int
	Left int64 // ticks of work left
}

// hasPost reports whether the robot was sent to a deposit tile.
func (r Robot) hasPost() bool {
	return r.PostCol >= 0
}

func (r *Robot) clearPost() {
	r.PostCol, r.PostRow = -1, -1
}

// newGame deals the starting region: every deposit patch full, and the
// starting robots idle by the core.
func newGame() *State {
	s := &State{
		NextID: 1,
		Robots: map[int64]Robot{},
		Drain:  map[string]float64{},
	}
	for _, d := range regionDeposits {
		s.Drain[depositKey(d)] = depositFull(d)
	}
	for i := 0; i < startingRobots; i++ {
		s.spawnRobot()
	}
	return s
}

func (s *State) spawnRobot() {
	id := s.NextID
	s.NextID++
	x, y := parkSpot(id)
	s.Robots[id] = Robot{ID: id, X: x, Y: y, PostCol: -1, PostRow: -1}
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
