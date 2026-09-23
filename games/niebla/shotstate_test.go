package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestWriteShotSquadState writes a region with two war factories and
// their troopers, the state a golib shot --save starts from: set
// NIEBLA_SHOT_STATE to the file to write and run the tests once, and
// the shot opens straight on the squads. Skipped otherwise, the way
// tests write nothing.
func TestWriteShotSquadState(t *testing.T) {
	path := os.Getenv("NIEBLA_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_SHOT_STATE to a file to write the squads' shot state to")
	}
	s := newGame()
	noRivals(s)
	home := squadOfTroopers(t, s, 4)
	col, row := groundNearCore()
	other := raised(t, s, BuildingWarFactory, col+2, row)
	x, y := cellCenterUnits(col+2, row)
	for i := 0; i < 2; i++ {
		r := s.Robots[s.spawnRobot(RobotCombat, x, y)]
		r.Squad = other.ID
		s.Robots[r.ID] = r
	}
	s.Squads[home.ID] = Squad{
		Home: home.ID, Order: OrderGuard,
		X: x - 400, Y: y + 300,
	}
	// The shot file is the store's shape: one saved name per key, and
	// resumeState reads the name "state".
	data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
	if err != nil {
		t.Fatalf("the state doesn't marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestWriteShotUnitState(t *testing.T) {
	path := os.Getenv("NIEBLA_UNIT_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_UNIT_SHOT_STATE to write the units' shot state")
	}
	s := newGame()
	noRivals(s)
	x, y := parkCenter()
	s.spawnRobot(RobotBuilt, x+20, y+12)
	s.spawnRobot(RobotCombat, x+40, y+24)
	data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestWriteShotRadialState writes the states the build menu's shots
// start from: NIEBLA_RADIAL_SHOT_STATE for the washed ring (the stores
// can pay nothing) and NIEBLA_RADIAL_FULL_SHOT_STATE for the paid one.
// Skipped otherwise, the way tests write nothing.
func TestWriteShotRadialState(t *testing.T) {
	washed := os.Getenv("NIEBLA_RADIAL_SHOT_STATE")
	full := os.Getenv("NIEBLA_RADIAL_FULL_SHOT_STATE")
	if washed == "" && full == "" {
		t.Skip("set NIEBLA_RADIAL_SHOT_STATE / NIEBLA_RADIAL_FULL_SHOT_STATE" +
			" to write the build menu's shot states")
	}
	write := func(path string, stock Stock) {
		s := newGame()
		noRivals(s)
		arriveAll(s)
		s.Stock = stock
		data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if washed != "" {
		write(washed, Stock{Oil: 300, Lilac: 40})
	}
	if full != "" {
		write(full, Stock{Oil: 600, Lilac: 1200})
	}
}
