package main

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

// nearestTileOf returns the deposit tile of a kind closest to the core.
func nearestTileOf(kind byte) (col, row int, ok bool) {
	best := math.MaxFloat64
	for r := 0; r < regionRows; r++ {
		for c := 0; c < regionCols; c++ {
			if tileAt(c, r) != kind {
				continue
			}
			if d := float64(tileDistance(c, r)); d < best {
				best, col, row, ok = d, c, r, true
			}
		}
	}
	return col, row, ok
}

func runTicks(s *State, n int) {
	for i := 0; i < n; i++ {
		Apply(s, Tick{})
	}
}

func TestNewGameStartsTwoIdleRobotsByTheCore(t *testing.T) {
	s := newGame()
	if len(s.Robots) != startingRobots {
		t.Fatalf("the core starts with %d robots, want %d",
			len(s.Robots), startingRobots)
	}
	cx, cy := tileCenterUnits(coreCol, coreRow)
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		if r.hasPost() {
			t.Errorf("robot %d starts with a post, want it idle", id)
		}
		if r.Carry != 0 || r.Cargo != "" {
			t.Errorf("robot %d starts carrying %v %v, want empty arms",
				id, r.Carry, r.Cargo)
		}
		if d := math.Hypot(r.X-cx, r.Y-cy); d > robotParkRadius+0.01 {
			t.Errorf("robot %d idles %v units from the core, want it on the %v unit ring",
				id, d, robotParkRadius)
		}
	}
	if s.Stock.Oil != 0 || s.Stock.Lilac != 0 {
		t.Errorf("the stores start with %v L and %v kg, want both empty",
			s.Stock.Oil, s.Stock.Lilac)
	}
}

func TestSentRobotHaulsOilHome(t *testing.T) {
	s := newGame()
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	Apply(s, SendRobot{Col: col, Row: row})
	if _, owned := postOwner(s, col, row); !owned {
		t.Fatal("nobody took the oil post")
	}
	// The nearest pool is the safe one inside the bubble: half a minute
	// out at robotSpeed, a little more back with the load.
	for i := 0; i < 60*400 && s.Stock.Oil <= 0; i++ {
		Apply(s, Tick{})
	}
	if s.Stock.Oil <= 0 {
		t.Fatalf("after 400 s of hauling the stores hold %v L of oil", s.Stock.Oil)
	}
	patch, _ := depositAt(col, row)
	total := depositFull(patch)
	left := remainingAt(s, col, row)
	if left >= total {
		t.Errorf("the pool still holds %v L, want it drained by the hauling", left)
	}
	if sum := s.Stock.Oil + left; math.Abs(sum-total) > 0.0001 {
		t.Errorf("the stores hold %v L and the pool %v L, want them to sum to %v",
			s.Stock.Oil, left, total)
	}
}

// TestDepositPatchIsOneUnit pins the patch law: sending a robot to any
// tile of a vein claims the whole vein, a second send on another of its
// tiles changes nothing, and recalling from any tile frees it.
func TestDepositPatchIsOneUnit(t *testing.T) {
	s := newGame()
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	patch, _ := depositAt(col, row)
	other, ok := otherPatchTile(patch, [2]int{col, row})
	if !ok {
		t.Fatal("the oil patch has no second tile to test with")
	}
	Apply(s, SendRobot{Col: col, Row: row})
	first, owned := postOwner(s, col, row)
	if !owned {
		t.Fatal("nobody took the oil patch")
	}
	Apply(s, SendRobot{Col: other[0], Row: other[1]})
	again, owned := postOwner(s, other[0], other[1])
	if !owned || again.ID != first.ID {
		t.Errorf("the patch's second tile answered %v, want the same robot %d",
			again, first.ID)
	}
	if len(s.Robots) != startingRobots {
		t.Fatalf("the colony grew to %d robots, want %d",
			len(s.Robots), startingRobots)
	}
	Apply(s, RecallRobot{Col: other[0], Row: other[1]})
	if _, owned := postOwner(s, col, row); owned {
		t.Error("the patch survived a recall from its other tile")
	}
}

func TestSendRobotPicksAndKeepsItsRobots(t *testing.T) {
	s := newGame()
	oilCol, oilRow, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	veinCol, veinRow, ok := nearestTileOf(kindLilac)
	if !ok {
		t.Fatal("the region has no lilac to test with")
	}
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	first, ok := postOwner(s, oilCol, oilRow)
	if !ok {
		t.Fatal("nobody took the oil post")
	}
	// A tile with its robot already asks nobody else.
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	again, _ := postOwner(s, oilCol, oilRow)
	if again.ID != first.ID {
		t.Errorf("robot %d took an occupied post from robot %d", again.ID, first.ID)
	}
	// The next post goes to the robot still free.
	Apply(s, SendRobot{Col: veinCol, Row: veinRow})
	second, ok := postOwner(s, veinCol, veinRow)
	if !ok {
		t.Fatal("nobody took the lilac post")
	}
	if second.ID == first.ID {
		t.Errorf("robot %d took both posts", first.ID)
	}
	// Recall frees a robot: the post stands empty, nobody replaces it.
	Apply(s, RecallRobot{Col: oilCol, Row: oilRow})
	if _, owned := postOwner(s, oilCol, oilRow); owned {
		t.Error("the oil post survived its recall")
	}
	if r := s.Robots[first.ID]; r.hasPost() {
		t.Errorf("robot %d still holds a post after the recall", first.ID)
	}
}

func TestDryDepositReleasesItsRobot(t *testing.T) {
	s := newGame()
	col, row, ok := nearestTileOf(kindLilac)
	if !ok {
		t.Fatal("the region has no lilac to test with")
	}
	patch, _ := depositAt(col, row)
	// Shrink the vein to five loads, so the test stays quick: the robot
	// must still empty the whole patch, and be released when the last
	// load is picked.
	full := robotCarryLilac * 5
	s.Drain[depositKey(patch)] = full
	Apply(s, SendRobot{Col: col, Row: row})
	for i := 0; i < 60*400 && s.Stock.Lilac < full; i++ {
		Apply(s, Tick{})
	}
	if _, owned := postOwner(s, col, row); owned {
		t.Fatal("the robot keeps a dry patch")
	}
	if left := remainingAt(s, col, row); left > 0 {
		t.Errorf("the drained vein still holds %v kg", left)
	}
	if s.Stock.Lilac != full {
		t.Errorf("the stores hold %v kg, want the whole vein, %v",
			s.Stock.Lilac, full)
	}
}

func TestBuildJobsComeFirst(t *testing.T) {
	s := newGame()
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	Apply(s, SendRobot{Col: col, Row: row})
	owner, _ := postOwner(s, col, row)
	runTicks(s, 60) // the robot sets out towards its post
	// A build job springs up on open ground across the way.
	jobCol, jobRow := coreCol+3, coreRow-3
	s.Jobs = []Job{{Col: jobCol, Row: jobRow, Left: 30}}
	jobX, jobY := tileCenterUnits(jobCol, jobRow)
	r := s.Robots[owner.ID]
	d0 := math.Hypot(r.X-jobX, r.Y-jobY)
	Apply(s, Tick{})
	r = s.Robots[owner.ID]
	d1 := math.Hypot(r.X-jobX, r.Y-jobY)
	if d1 >= d0 {
		t.Errorf("the robot ignored the build job: its distance to it went %v to %v",
			d0, d1)
	}
	// It raises the job before it goes back to its own post: the walk
	// there and back is a few minutes at robotSpeed.
	for i := 0; i < 60*400 && len(s.Jobs) > 0; i++ {
		Apply(s, Tick{})
	}
	if len(s.Jobs) != 0 {
		t.Fatal("the build job never finished")
	}
	if s.Stock.Oil > 0 {
		t.Errorf("the stores hold %v L before the job is done, want none", s.Stock.Oil)
	}
	for i := 0; i < 60*500 && s.Stock.Oil == 0; i++ {
		Apply(s, Tick{})
	}
	if s.Stock.Oil == 0 {
		t.Fatal("the robot never went back to its post after the job")
	}
}

func TestTheSimulationReplaysTheSame(t *testing.T) {
	play := func() *State {
		s := newGame()
		oilCol, oilRow, _ := nearestTileOf(kindOil)
		veinCol, veinRow, _ := nearestTileOf(kindLilac)
		Apply(s, SendRobot{Col: oilCol, Row: oilRow})
		Apply(s, SendRobot{Col: veinCol, Row: veinRow})
		runTicks(s, 3600)
		return s
	}
	if !reflect.DeepEqual(play(), play()) {
		t.Errorf("two runs of the same actions ended in different states")
	}
}

func TestTheStateSerializesRound(t *testing.T) {
	s := newGame()
	col, row, _ := nearestTileOf(kindOil)
	Apply(s, SendRobot{Col: col, Row: row})
	runTicks(s, 600)
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("the state does not marshal: %v", err)
	}
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("the state does not unmarshal: %v", err)
	}
	if !reflect.DeepEqual(&back, s) {
		t.Errorf("the state changed across a JSON round trip")
	}
}
