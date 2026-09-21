package main

import (
	"math"
	"strings"
	"testing"
)

// pointAtTiles returns a world point standing d tiles from the core,
// straight east of it.
func pointAtTiles(d float64) (float64, float64) {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	return cx + d*unitsPerTile, cy
}

func TestTheFogKeepsItsCycles(t *testing.T) {
	s := newGame()
	if s.Fog.Cycle != 0 || s.Fog.CycleLeft != fogCycleTicks {
		t.Fatalf("the fog starts at cycle %d with %d ticks left, want 0 and %d",
			s.Fog.Cycle, s.Fog.CycleLeft, fogCycleTicks)
	}
	runTicks(s, fogCycleTicks)
	if s.Fog.Cycle != 1 || s.Fog.CycleLeft != fogCycleTicks {
		t.Errorf("after a cycle the fog reads cycle %d, %d ticks left, want 1 and %d",
			s.Fog.Cycle, s.Fog.CycleLeft, fogCycleTicks)
	}
}

func TestTheFirstSwellRisesOnSchedule(t *testing.T) {
	s := newGame()
	// The last calm cycle is the forecast's word: the swell rises at
	// its end, whole.
	runTicks(s, (int(fogSwellPeriod)-1)*fogCycleTicks)
	if s.Fog.NextIn != 1 || s.Fog.SwellLeft != 0 {
		t.Fatalf("a cycle before the swell the fog reads NextIn %v, SwellLeft %d, want 1 and 0",
			s.Fog.NextIn, s.Fog.SwellLeft)
	}
	runTicks(s, fogCycleTicks)
	if s.Fog.SwellLeft != fogSwellTicks || s.Fog.Swells != 1 {
		t.Fatalf("the first swell did not rise: %d ticks left, %d swells",
			s.Fog.SwellLeft, s.Fog.Swells)
	}
	if s.Fog.NextIn >= fogSwellPeriod {
		t.Errorf("the calm after the first swell is %v cycles, want it shortened",
			s.Fog.NextIn)
	}
	runTicks(s, fogSwellTicks)
	if s.Fog.SwellLeft != 0 {
		t.Errorf("the swell kept %d ticks past its length", s.Fog.SwellLeft)
	}
}

// swellUp puts a swell up and pressed in whole, the way it stands once
// its ramp is over.
func swellUp(s *State) {
	s.Fog.SwellLeft = fogSwellTicks
	s.Fog.Pressure = 1
}

func TestASwellPressesInAndLetsGoLittleByLittle(t *testing.T) {
	s := newGame()
	Apply(s, DevHoldSwell{On: true})
	last := fogLineNow(s)
	for i := 0; i < fogSwellRampTicks; i++ {
		Apply(s, Tick{})
		line := fogLineNow(s)
		if line >= last || last-line > 0.05 {
			t.Fatalf("tick %d moved the line from %v to %v, want a small step in",
				i, last, line)
		}
		last = line
	}
	if want := fogLineRadius - swellReach(s); math.Abs(float64(last-want)) > 1e-4 {
		t.Fatalf("the ramp over, the line stands at %v, want %v", last, want)
	}
	Apply(s, DevHoldSwell{On: false})
	runTicks(s, fogSwellRampTicks/2)
	if line := fogLineNow(s); line <= last || line >= fogLineRadius {
		t.Errorf("half way out the line stands at %v, want it between %v and %v",
			line, last, float32(fogLineRadius))
	}
	runTicks(s, fogSwellRampTicks/2+1)
	if line := fogLineNow(s); line != fogLineRadius {
		t.Errorf("let go, the line rests at %v, want %v", line, float32(fogLineRadius))
	}
}

func TestASwellPressesTheLineIn(t *testing.T) {
	s := newGame()
	band := fogLineRadius - 1.5 // a ring the calm fog leaves clear, the swell does not
	x, y := pointAtTiles(band)
	if fogAt(s, x, y) != 0 {
		t.Fatalf("the band at %v tiles already sits in the calm fog", band)
	}
	if line := fogLineNow(s); line != fogLineRadius {
		t.Fatalf("the calm line stands at %v, want %v", line, fogLineRadius)
	}
	swellUp(s)
	if line := fogLineNow(s); line >= fogLineRadius {
		t.Fatalf("the swell left the line at %v, want it pressed in", line)
	}
	if fogAt(s, x, y) <= 0 {
		t.Error("the swell left the band clear")
	}
	cx, cy := tileCenterUnits(coreCol, coreRow)
	if fogAt(s, cx, cy) != 0 {
		t.Error("the swell broke the core's bubble")
	}
}

func TestASwellNeverReachesTheBubble(t *testing.T) {
	s := newGame()
	s.Fog.Swells = 1000 // a fog grown as hard as it will go
	swellUp(s)
	x, y := pointAtTiles(float64(coreBubbleRadius) + 0.5/unitsPerTile)
	if fogAt(s, x, y) != 0 {
		t.Errorf("the fog sits at the bubble's rim during a swell, line at %v",
			fogLineNow(s))
	}
	if line := fogLineNow(s); line < coreBubbleRadius+fogSwellMargin {
		t.Errorf("the pushed line at %v ignores the bubble's margin", line)
	}
}

func TestThePushedBandDragsMore(t *testing.T) {
	s := newGame()
	band := fogLineRadius - 1.5 // inside the pushed band, clear of the calm fog
	x, y := pointAtTiles(band)
	moved := func() float64 {
		r := Robot{X: x, Y: y, PostCol: -1, PostRow: -1}
		before := r.X
		r.walkTowards(s, x+10*unitsPerTile, y)
		return r.X - before
	}
	calm := moved()
	if math.Abs(calm-robotSpeed/60) > 1e-9 {
		t.Errorf("the calm fog dragged at %v tiles: %v a tick, want the whole %v",
			band, calm, robotSpeed/60)
	}
	swellUp(s)
	swell := moved()
	if swell >= calm {
		t.Errorf("the pushed band let a robot keep %v of its step against the calm %v",
			swell, calm)
	}
}

func TestASwellBurnsTanksFasterOutside(t *testing.T) {
	s := newGame()
	x, y := pointAtTiles(4.5) // outside the bubble, clear of even the pushed fog
	id := s.NextID
	s.spawnRobot(RobotBuilt, x, y)
	runTicks(s, 60)
	r := s.Robots[id]
	calm := robotTankLiters - r.Tank
	r.Tank = robotTankLiters
	s.Robots[id] = r
	swellUp(s)
	runTicks(s, 60)
	swell := robotTankLiters - s.Robots[id].Tank
	if math.Abs(calm-robotBurnPerSecond) > 0.001 {
		t.Errorf("a minute of calm burned %v L, want %v", calm, robotBurnPerSecond)
	}
	if math.Abs(swell-calm*fogSwellBurn) > 0.001 {
		t.Errorf("a minute of swell burned %v L, want %v", swell, calm*fogSwellBurn)
	}
	// The bubble keeps its word: inside it, the swell burns nothing extra.
	id2 := s.NextID
	px, py := parkSpot(id2)
	s.spawnRobot(RobotBuilt, px, py)
	runTicks(s, 60)
	inside := robotTankLiters - s.Robots[id2].Tank
	if math.Abs(inside-robotBurnPerSecond) > 0.001 {
		t.Errorf("a minute inside the bubble burned %v L during a swell, want %v",
			inside, robotBurnPerSecond)
	}
}

func TestTheHUDNamesTheSwell(t *testing.T) {
	s := newGame()
	scene := &playScene{state: s}
	if strings.Contains(scene.hudLine(), "swell") {
		t.Error("the HUD cries swell in the calm first cycle")
	}
	runTicks(s, (int(fogSwellPeriod)-1)*fogCycleTicks)
	if line := scene.hudLine(); !strings.Contains(line, "swell next cycle") {
		t.Errorf("the forecast cycle reads %q, want it naming the swell", line)
	}
	runTicks(s, fogCycleTicks)
	line := scene.hudLine()
	if !strings.Contains(line, "swell") || strings.Contains(line, "next cycle") {
		t.Errorf("the HUD misses the swell: %q", line)
	}
}
