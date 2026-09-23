package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestFacingFromMovementUsesScreenOctants(t *testing.T) {
	for want := uint8(0); want < 8; want++ {
		angle := float64(want) * math.Pi / 4
		screenX, screenY := math.Cos(angle), math.Sin(angle)
		projectedX := 2 * screenX / float64(unitW)
		projectedY := 2 * screenY / float64(unitH)
		dx := (projectedX + projectedY) / 2
		dy := (projectedY - projectedX) / 2

		if got := facingFromMovement(dx, dy); got != want {
			t.Errorf("screen octant %d maps to %d", want, got)
		}
	}
}

func TestRobotFacingChangesWithMovementAndWaitsAtRest(t *testing.T) {
	s := newGame()
	r := Robot{X: 1000, Y: 1000, Facing: facingDown}

	if r.walkTowards(s, 1010, 990) {
		t.Fatal("robot arrived after one short step")
	}
	if r.Facing != facingRight {
		t.Fatalf("robot faces %d while moving right, want %d",
			r.Facing, facingRight)
	}

	if !r.walkTowards(s, r.X, r.Y) {
		t.Fatal("robot did not arrive at its current position")
	}
	if r.Facing != facingRight {
		t.Errorf("stationary robot turned to %d", r.Facing)
	}

	r.walkTowards(s, 1000, 1000)
	if r.Facing != facingLeft {
		t.Errorf("robot faces %d while moving left, want %d",
			r.Facing, facingLeft)
	}
}

func TestRivalFacingChangesWithMovementAndWaitsAtRest(t *testing.T) {
	s := &State{Enemies: map[int64]Enemy{}}
	e := Enemy{
		ID: 1, Kind: EnemyScout, X: 100, Y: 100,
		Facing: facingDown,
	}

	if s.driveParty([]Enemy{e}, 110, 90) {
		t.Fatal("vehicle arrived after one short step")
	}
	e = s.Enemies[e.ID]
	if e.Facing != facingRight {
		t.Fatalf("vehicle faces %d while moving right, want %d",
			e.Facing, facingRight)
	}

	s.driveParty([]Enemy{e}, e.X, e.Y)
	e = s.Enemies[e.ID]
	if e.Facing != facingRight {
		t.Errorf("stationary vehicle turned to %d", e.Facing)
	}

	s.driveParty([]Enemy{e}, 100, 100)
	e = s.Enemies[e.ID]
	if e.Facing != facingLeft {
		t.Errorf("vehicle faces %d while moving left, want %d",
			e.Facing, facingLeft)
	}
}

func TestFacingSavesAndDefaultsForLegacyStates(t *testing.T) {
	s := &State{
		Robots: map[int64]Robot{
			1: {ID: 1, Kind: RobotCore, Facing: facingUpLeft},
		},
		Enemies: map[int64]Enemy{
			2: {ID: 2, Kind: EnemyScout, Facing: facingDownRight},
		},
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var saved State
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Robots[1].Facing != facingUpLeft ||
		saved.Enemies[2].Facing != facingDownRight {
		t.Fatalf("facing changed after save: robot %d, vehicle %d",
			saved.Robots[1].Facing, saved.Enemies[2].Facing)
	}

	const legacy = `{"Robots":{"1":{"ID":1,"Kind":"core"}},` +
		`"Enemies":{"2":{"ID":2,"Kind":"scout"}}}`
	var old State
	if err := json.Unmarshal([]byte(legacy), &old); err != nil {
		t.Fatal(err)
	}
	if old.Robots[1].Facing != facingRight ||
		old.Enemies[2].Facing != facingRight {
		t.Errorf("old save defaults to robot %d and vehicle %d, want right",
			old.Robots[1].Facing, old.Enemies[2].Facing)
	}
}
