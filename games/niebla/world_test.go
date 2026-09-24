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

// tileCell returns a cell of a tile, the one at its top corner: what a
// click on the tile's ground would pick.
func tileCell(tcol, trow int) (col, row int) {
	const cellsPerTile = unitsPerTile / buildingCell
	return tcol * cellsPerTile, trow * cellsPerTile
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
		if d := math.Hypot(r.X-cx, r.Y-cy); d > parkFromCore+parkSlots*parkSpacing {
			t.Errorf("robot %d idles %v units from the core, want it in the ranks beside it",
				id, d)
		}
	}
	if s.Stock.Oil != startingStockOil || s.Stock.Lilac != startingStockLilac {
		t.Errorf("the stores start with %v L and %v kg, want the core's gift",
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
	if len(postRobots(s, col, row)) == 0 {
		t.Fatal("nobody took the oil post")
	}
	// The nearest pool is the safe one inside the bubble: half a minute
	// out at robotSpeed, a little more back with the load. Hauling is
	// proven by what it adds to the gift the stores start with.
	for i := 0; i < 60*400 && s.Stock.Oil <= startingStockOil; i++ {
		Apply(s, Tick{})
	}
	if s.Stock.Oil <= startingStockOil {
		t.Fatalf("after 400 s of hauling the stores hold %v L of oil", s.Stock.Oil)
	}
	patch, _ := depositAt(col, row)
	total := depositFull(patch)
	left := remainingAt(s, col, row)
	if left >= total {
		t.Errorf("the pool still holds %v L, want it drained by the hauling", left)
	}
	if sum := s.Stock.Oil + left; math.Abs(sum-total-startingStockOil) > 0.0001 {
		t.Errorf("the stores hold %v L and the pool %v L, want them to sum to the gift plus %v",
			s.Stock.Oil, left, total)
	}
}

// TestDepositPatchSharesItsWorkersAcrossEveryTile pins the patch law:
// sending to any tile assigns another worker to the whole deposit, and
// recalling one worker leaves the others assigned.
func TestDepositPatchSharesItsWorkersAcrossEveryTile(t *testing.T) {
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
	workers := postRobots(s, col, row)
	if len(workers) == 0 {
		t.Fatal("nobody took the oil patch")
	}
	first := workers[0]
	Apply(s, SendRobot{Col: other[0], Row: other[1]})
	workers = postRobots(s, other[0], other[1])
	if len(workers) != 2 {
		t.Fatalf("the patch has %d workers after two sends, want 2", len(workers))
	}
	if workers[0].ID == workers[1].ID ||
		(workers[0].ID != first.ID && workers[1].ID != first.ID) {
		t.Errorf("the patch workers are %v, want distinct IDs including %d",
			workers, first.ID)
	}
	if len(s.Robots) != startingRobots {
		t.Fatalf("the colony grew to %d robots, want %d",
			len(s.Robots), startingRobots)
	}
	Apply(s, RecallRobot{ID: first.ID})
	workers = postRobots(s, col, row)
	if len(workers) != 1 || workers[0].ID == first.ID {
		t.Errorf("recalling robot %d left patch workers %v, want the other one",
			first.ID, workers)
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
	if len(postRobots(s, oilCol, oilRow)) == 0 {
		t.Fatal("nobody took the oil post")
	}
	// A second send adds a different worker to the same patch.
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	oilWorkers := postRobots(s, oilCol, oilRow)
	if len(oilWorkers) != 2 || oilWorkers[0].ID == oilWorkers[1].ID {
		t.Fatalf("two sends assigned %v to the oil patch, want two distinct robots",
			oilWorkers)
	}
	// No robot already at this patch is assigned twice.
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	if got := len(postRobots(s, oilCol, oilRow)); got != 2 {
		t.Errorf("a third send made %d workers at the oil patch, want 2", got)
	}
	// With no free robots, the next deposit retasks one from another patch.
	Apply(s, SendRobot{Col: veinCol, Row: veinRow})
	veinWorkers := postRobots(s, veinCol, veinRow)
	if len(veinWorkers) == 0 {
		t.Fatal("nobody took the lilac post")
	}
	second := veinWorkers[0]
	oilWorkers = postRobots(s, oilCol, oilRow)
	if len(oilWorkers) != 1 || oilWorkers[0].ID == second.ID {
		t.Errorf("retasking to lilac left oil workers %v and lilac worker %d",
			oilWorkers, second.ID)
	}
	// Recall frees exactly that robot's post.
	recalled := oilWorkers[0]
	Apply(s, RecallRobot{ID: recalled.ID})
	if workers := postRobots(s, oilCol, oilRow); len(workers) != 0 {
		t.Errorf("the oil patch kept workers after recalling one: %v", workers)
	}
	if r := s.Robots[recalled.ID]; r.hasPost() {
		t.Errorf("robot %d still holds a post after the recall", recalled.ID)
	}
	if workers := postRobots(s, veinCol, veinRow); len(workers) != 1 {
		t.Errorf("recalling from oil changed the lilac workers: %v", workers)
	}
}

func TestWorkersShareAndExhaustOneDeposit(t *testing.T) {
	s := newGame()
	col, row, ok := nearestTileOf(kindLilac)
	if !ok {
		t.Fatal("the region has no lilac to test with")
	}
	patch, _ := depositAt(col, row)
	amount := robotCarryLilac*3 + 7
	s.Drain[depositKey(patch)] = amount
	Apply(s, SendRobot{Col: col, Row: row})
	Apply(s, SendRobot{Col: col, Row: row})
	if workers := postRobots(s, col, row); len(workers) != 2 {
		t.Fatalf("the vein has %d workers, want 2", len(workers))
	}
	for i := 0; i < 60*400 && s.Stock.Lilac < startingStockLilac+amount; i++ {
		Apply(s, Tick{})
	}
	if got := s.Stock.Lilac; got != startingStockLilac+amount {
		t.Fatalf("the stores received %v kg, want %v", got, startingStockLilac+amount)
	}
	if left := remainingAt(s, col, row); left != 0 {
		t.Errorf("the vein still has %v kg after both workers finished", left)
	}
	if workers := postRobots(s, col, row); len(workers) != 0 {
		t.Errorf("the dry vein kept workers assigned: %v", workers)
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
	for i := 0; i < 60*400 && s.Stock.Lilac < startingStockLilac+full; i++ {
		Apply(s, Tick{})
	}
	if len(postRobots(s, col, row)) > 0 {
		t.Fatal("the robot keeps a dry patch")
	}
	if left := remainingAt(s, col, row); left > 0 {
		t.Errorf("the drained vein still holds %v kg", left)
	}
	if s.Stock.Lilac != startingStockLilac+full {
		t.Errorf("the stores hold %v kg, want the gift plus the whole vein, %v",
			s.Stock.Lilac, startingStockLilac+full)
	}
}

func TestBuildJobsComeFirst(t *testing.T) {
	s := newGame()
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	Apply(s, SendRobot{Col: col, Row: row})
	owner := postRobots(s, col, row)[0]
	runTicks(s, 60) // the robot sets out towards its post
	// A build job springs up on open ground across the way, five cells
	// north-east of the core.
	jobCol, jobRow := coreCol*5+5, coreRow*5-5
	s.Jobs = []Job{{Col: jobCol, Row: jobRow, Left: 30}}
	jobX, jobY := cellCenterUnits(jobCol, jobRow)
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
	if s.Stock.Oil > startingStockOil {
		t.Errorf("the stores hold %v L before the job is done, want just the gift",
			s.Stock.Oil)
	}
	for i := 0; i < 60*500 && s.Stock.Oil <= startingStockOil; i++ {
		Apply(s, Tick{})
	}
	if s.Stock.Oil <= startingStockOil {
		t.Fatal("the robot never went back to its post after the job")
	}
}

func TestRobotsBuildProtectorsBeforeOlderJobs(t *testing.T) {
	s := newGame()
	siloCol, siloRow := groundNearCore()
	protectorCol, protectorRow := siloCol+1, siloRow
	s.Jobs = []Job{
		{Kind: BuildingSilo, Col: siloCol, Row: siloRow, Left: 20},
		{Kind: BuildingProtector, Col: protectorCol, Row: protectorRow, Left: 1},
	}

	r := s.Robots[1]
	x, y := cellCenterUnits(protectorCol, protectorRow)
	angle := float64(r.ID) * goldenAngle
	r.X = x + math.Cos(angle)*11
	r.Y = y + math.Sin(angle)*11
	s.Robots[r.ID] = r
	delete(s.Robots, 2)

	Apply(s, Tick{})

	protector, built := buildingAt(s, protectorCol, protectorRow)
	if !built || protector.Kind != BuildingProtector {
		t.Fatal("the robot worked on the older silo instead of the protector")
	}
	if len(s.Jobs) != 1 || s.Jobs[0].Kind != BuildingSilo ||
		s.Jobs[0].Left != 20 {
		t.Errorf("the older silo job changed while the protector was built: %v",
			s.Jobs)
	}
}

func TestTheSimulationReplaysTheSame(t *testing.T) {
	play := func() *State {
		s := newGame()
		arriveAll(s)
		oilCol, oilRow, _ := nearestTileOf(kindOil)
		veinCol, veinRow, _ := nearestTileOf(kindLilac)
		Apply(s, SendRobot{Col: oilCol, Row: oilRow})
		Apply(s, SendRobot{Col: veinCol, Row: veinRow})
		s.Stock = Stock{Oil: 400, Lilac: 800}
		Apply(s, MarkBuilding{Kind: BuildingCharger, Col: 67, Row: 60})
		runTicks(s, 1800) // the charger rises in the first half
		Apply(s, QueueRobot{Building: 3})
		runTicks(s, 1800)  // the factory's robot rolls out in the second
		runTicks(s, 33000) // the run crosses the first swell, at cycle 18
		return s
	}
	if !reflect.DeepEqual(play(), play()) {
		t.Errorf("two runs of the same actions ended in different states")
	}
}

func TestTheStateSerializesRound(t *testing.T) {
	s := newGame()
	arriveAll(s)
	col, row, _ := nearestTileOf(kindOil)
	Apply(s, SendRobot{Col: col, Row: row})
	s.Stock = Stock{Oil: 400, Lilac: 800}
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: 67, Row: 60})
	runTicks(s, 1200) // the silo rises, and joins the oil room
	Apply(s, QueueRobot{Building: 3})
	runTicks(s, 1200) // the factory's robot joins the colony
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

func TestIdleRobotsRestInRanksByTheCore(t *testing.T) {
	s := newGame()
	cx, cy := tileCenterUnits(coreCol, coreRow)
	for len(s.Robots) < parkSlots+2 {
		s.spawnRobot(RobotCore, cx+60, cy+60)
	}
	runTicks(s, 60*10)
	if idle := idleRobots(s); idle != parkSlots+2 {
		t.Fatalf("%d robots idle, want all %d", idle, parkSlots+2)
	}
	taken := map[[2]float64]int{}
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		taken[[2]float64{r.X, r.Y}]++
		if d := math.Hypot(r.X-cx, r.Y-cy); d > parkFromCore+parkSlots*parkSpacing {
			t.Errorf("robot %d rests %v u from the core, want it in the ranks", id, d)
		}
	}
	if len(taken) != parkSlots {
		t.Errorf("the idle robots stand on %d spots, want the ranks' %d", len(taken), parkSlots)
	}
	// One leaves for a post, and the ranks close up behind it.
	col, row, _ := nearestTileOf(kindLilac)
	first := s.Robots[1]
	first.PostCol, first.PostRow = col, row
	s.Robots[1] = first
	runTicks(s, 60*5)
	x, y := parkSlot(0)
	if r := s.Robots[2]; r.X != x || r.Y != y {
		t.Errorf("robot 2 rests at %v, %v, want the first place %v, %v", r.X, r.Y, x, y)
	}
}
