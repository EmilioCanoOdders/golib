package main

import (
	"math"
	"testing"
)

func TestAHeldSwellStaysUpAndCountsForNothing(t *testing.T) {
	s := newGame()
	Apply(s, DevHoldSwell{On: true})
	runTicks(s, fogSwellTicksMax+fogCycleTicks)
	if s.Fog.SwellLeft <= 0 || fogLineNow(s) >= fogLineRadius {
		t.Fatal("the held swell drained, want it up until let go")
	}
	Apply(s, DevHoldSwell{On: false})
	if s.Fog.SwellLeft != 0 || s.Fog.Held {
		t.Fatalf("let go, the swell has %d ticks left, want calm", s.Fog.SwellLeft)
	}
	if s.Fog.Swells != 0 {
		t.Errorf("the held swell counted as %d, want the fog's own dials untouched",
			s.Fog.Swells)
	}
}

func TestDevSpawnRobotPutsAFreeBuiltRobotOnTheSpot(t *testing.T) {
	s := newGame()
	stock := s.Stock
	Apply(s, DevSpawnRobot{X: 500, Y: 700})
	if len(s.Robots) != startingRobots+1 || s.Stock != stock {
		t.Fatalf("%d robots and stores %+v, want one more robot for nothing",
			len(s.Robots), s.Stock)
	}
	ids := sortedRobotIDs(s)
	r := s.Robots[ids[len(ids)-1]]
	if r.Kind != RobotBuilt || r.Tank != robotTankLiters || r.X != 500 || r.Y != 700 {
		t.Errorf("the robot is %+v, want a built one, tank full, at 500, 700", r)
	}
	Apply(s, DevSpawnRobot{X: -1, Y: 700})
	if len(s.Robots) != startingRobots+1 {
		t.Error("a spot outside the region took a robot")
	}
}

func TestUnitsAtWorldUndoesProject(t *testing.T) {
	px, py := project(1234, 567)
	x, y := unitsAtWorld(float64(px), float64(py))
	if math.Abs(x-1234) > 0.5 || math.Abs(y-567) > 0.5 {
		t.Fatalf("1234, 567 came back as %v, %v", x, y)
	}
}
