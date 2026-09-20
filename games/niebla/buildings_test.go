package main

import (
	"math"
	"testing"
)

// seedStock fills the stores, the way a played game would have them
// before the player starts marking buildings.
func seedStock(s *State) {
	s.Stock = Stock{Oil: 600, Lilac: 1200}
}

// groundNearCore returns a buildable cell inside the core's bubble, off
// the core itself: tile 13, 12.
func groundNearCore() (int, int) {
	return 104, 96
}

// groundInTheFog returns a buildable cell far outside every bubble,
// deep past the fog line: tile 2, 2.
func groundInTheFog() (int, int) {
	return 20, 20
}

func TestMarkingPaysAndRaisesTheBuilding(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundNearCore()
	Apply(s, MarkBuilding{Kind: BuildingFactory, Col: col, Row: row})
	lilac, oil := buildingCost(BuildingFactory)
	if s.Stock.Lilac != 1200-lilac || s.Stock.Oil != 600-oil {
		t.Fatalf("the stores hold %v kg and %v L after marking, want the cost paid",
			s.Stock.Lilac, s.Stock.Oil)
	}
	if len(s.Jobs) != 1 || s.Jobs[0].Kind != BuildingFactory {
		t.Fatalf("marking left %d jobs, want one factory job", len(s.Jobs))
	}
	// The robots walk over and raise it, and the building takes the
	// cell.
	for i := 0; i < 60*120 && len(s.Buildings) == 0; i++ {
		Apply(s, Tick{})
	}
	b, ok := buildingAt(s, col, row)
	if !ok {
		t.Fatal("the factory never rose from the job")
	}
	if b.Kind != BuildingFactory {
		t.Errorf("cell %d, %d took a %s, want a factory", col, row, b.Kind)
	}
	if len(s.Jobs) != 0 {
		t.Errorf("%d jobs survive their building", len(s.Jobs))
	}
}

func TestCellsFitSeveralBuildingsToATile(t *testing.T) {
	s := newGame()
	seedStock(s)
	// Two buildings on one tile: neighbour cells, both marked.
	Apply(s, MarkBuilding{Kind: BuildingFactory, Col: 104, Row: 96})
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: 105, Row: 96})
	if len(s.Jobs) != 2 {
		t.Fatalf("marking left %d jobs, want two", len(s.Jobs))
	}
	// A cell with a job pending takes no second job.
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: 104, Row: 96})
	if len(s.Jobs) != 2 {
		t.Error("a cell took two jobs")
	}
	// A cell past the region's edge is no cell at all.
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: regionCellCols, Row: 60})
	if len(s.Jobs) != 2 {
		t.Error("the fog's edge took a silo")
	}
	for i := 0; i < 60*180 && len(s.Buildings) < 2; i++ {
		Apply(s, Tick{})
	}
	if built := buildingsOnTile(s, 13, 12); len(built) != 2 {
		t.Errorf("tile 13, 12 holds %d buildings, want the factory and the silo",
			len(built))
	}
}

func TestBuildingsCantStandInTheFog(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundInTheFog()
	Apply(s, MarkBuilding{Kind: BuildingCharger, Col: col, Row: row})
	if len(s.Jobs) != 0 {
		t.Fatal("the fog took a charger")
	}
	if s.Stock.Lilac != 1200 || s.Stock.Oil != 600 {
		t.Errorf("a refused marking still cost %v kg and %v L",
			1200-s.Stock.Lilac, 600-s.Stock.Oil)
	}
	// The protector is the one kind raised to stand out there.
	Apply(s, MarkBuilding{Kind: BuildingProtector, Col: col, Row: row})
	if len(s.Jobs) != 1 {
		t.Fatal("the fog refused the protector")
	}
}

func TestFactoryBuildsRobotsWithATank(t *testing.T) {
	s := newGame()
	seedStock(s)
	id := s.NextID
	s.NextID++
	col, row := groundNearCore()
	s.Buildings[id] = Building{
		ID: id, Kind: BuildingFactory, Col: col, Row: row,
	}
	Apply(s, QueueRobot{Building: id})
	if s.Buildings[id].Work != factoryRobotTicks {
		t.Fatalf("the factory stands at %d ticks of work, want %d",
			s.Buildings[id].Work, factoryRobotTicks)
	}
	if paid := 1200 - s.Stock.Lilac; paid != robotCostLilac {
		t.Errorf("the robot cost %v kg, want %v", paid, robotCostLilac)
	}
	// A factory already building takes no second order.
	Apply(s, QueueRobot{Building: id})
	if s.Buildings[id].Work != factoryRobotTicks {
		t.Error("the factory took a second robot while busy")
	}
	before := len(s.Robots)
	for i := 0; i < 60*30 && len(s.Robots) == before; i++ {
		Apply(s, Tick{})
	}
	if len(s.Robots) != before+1 {
		t.Fatal("the factory never rolled a robot out")
	}
	var fresh Robot
	found := false
	for _, r := range s.Robots {
		if r.Kind == RobotBuilt {
			fresh, found = r, true
		}
	}
	if !found {
		t.Fatal("the new robot is not a built one")
	}
	if fresh.Tank < robotTankLiters-0.1 {
		t.Errorf("the new robot rolls out with %v L in the tank, want %v",
			fresh.Tank, robotTankLiters)
	}
	fx, fy := cellCenterUnits(col, row)
	if d := math.Hypot(fresh.X-fx, fresh.Y-fy); d > unitsPerTile {
		t.Errorf("the new robot rolled out %v u from its factory", d)
	}
}

func TestABuiltRobotRefuelsBeforeItRunsDry(t *testing.T) {
	s := newGame()
	s.Stock = Stock{Oil: 200}
	id := s.NextID
	s.NextID++
	cx, cy := tileCenterUnits(coreCol, coreRow)
	s.Robots[id] = Robot{
		ID: id, Kind: RobotBuilt,
		X: cx + 100, Y: cy, Tank: 10,
		PostCol: -1, PostRow: -1,
	}
	// A tank under the low line sends the robot to the core, and the
	// refill drinks the stores at the charger's pace.
	for i := 0; i < 60*60 && s.Robots[id].Tank < robotTankLiters-0.001; i++ {
		Apply(s, Tick{})
	}
	r := s.Robots[id]
	if r.Tank < robotTankLiters-0.001 {
		t.Fatalf("after a minute by the core the tank holds %v L, want %v",
			r.Tank, robotTankLiters)
	}
	// The refill fills the tank's 110 missing liters; the walk over and
	// the seconds at the pump burn a little of the robot's own oil.
	if spent := 200 - s.Stock.Oil; spent < robotTankLiters-10 || spent > robotTankLiters-10+5 {
		t.Errorf("the refill drank %v L, want %v plus a little burn",
			spent, robotTankLiters-10)
	}
}

func TestFogDigestsARobotRunDryOutsideTheBubbles(t *testing.T) {
	s := newGame()
	id := s.NextID
	s.NextID++
	fx, fy := tileCenterUnits(2, 2)
	s.Robots[id] = Robot{
		ID: id, Kind: RobotBuilt, X: fx, Y: fy, Tank: 0.001,
		PostCol: -1, PostRow: -1,
	}
	Apply(s, Tick{})
	if _, ok := s.Robots[id]; ok {
		t.Fatal("the fog left a dry robot standing")
	}
	if len(s.Robots) != startingRobots {
		t.Errorf("the colony holds %d robots, want %d",
			len(s.Robots), startingRobots)
	}
	// A core robot never runs dry, so the fog never takes one.
	cid := s.NextID
	s.NextID++
	s.Robots[cid] = Robot{
		ID: cid, Kind: RobotCore, X: fx, Y: fy, PostCol: -1, PostRow: -1,
	}
	for i := 0; i < 60; i++ {
		Apply(s, Tick{})
	}
	if _, ok := s.Robots[cid]; !ok {
		t.Error("the fog digested a core robot")
	}
	// On the robot's own cell, a shadow protector's bubble shelters even
	// a dry built robot.
	pid := s.NextID
	s.NextID++
	s.Buildings[pid] = Building{
		ID: pid, Kind: BuildingProtector, Col: 20, Row: 20,
	}
	sid := s.NextID
	s.NextID++
	s.Robots[sid] = Robot{
		ID: sid, Kind: RobotBuilt, X: fx, Y: fy, Tank: 0.001,
		PostCol: -1, PostRow: -1,
	}
	for i := 0; i < 60; i++ {
		Apply(s, Tick{})
	}
	if _, ok := s.Robots[sid]; !ok {
		t.Error("the fog reached a robot under a protector")
	}
}

func TestTheFogSlowsWhoeverWalksIt(t *testing.T) {
	s := newGame()
	cx, cy := tileCenterUnits(coreCol, coreRow)
	clear := Robot{Kind: RobotCore, X: cx, Y: cy, PostCol: -1, PostRow: -1}
	fogged := Robot{Kind: RobotCore, X: 0, Y: 0, PostCol: -1, PostRow: -1}
	for i := 0; i < 120; i++ {
		clear.walkTowards(s, cx+100, cy)
		fogged.walkTowards(s, 100, 0)
	}
	if clear.X-cx <= fogged.X {
		t.Fatalf("the fogged walker kept pace: %v u against %v u",
			fogged.X, clear.X-cx)
	}
	ratio := fogged.X / (clear.X - cx)
	if math.Abs(ratio-fogSpeedFactor) > 0.01 {
		t.Errorf("deep fog left %v of the speed, want %v",
			ratio, fogSpeedFactor)
	}
	// A protector's pocket clears the air: full pace under it. Its cell
	// holds the walker's whole stroll, (0,0) to (60,0), inside the
	// bubble's 400 u.
	pid := s.NextID
	s.NextID++
	s.Buildings[pid] = Building{
		ID: pid, Kind: BuildingProtector, Col: 0, Row: 0,
	}
	sheltered := Robot{Kind: RobotCore, X: 0, Y: 0, PostCol: -1, PostRow: -1}
	for i := 0; i < 120; i++ {
		sheltered.walkTowards(s, 100, 0)
	}
	if sheltered.X < clear.X-cx {
		t.Errorf("a robot under a protector walked %v u against %v u",
			sheltered.X, clear.X-cx)
	}
}

func TestFullStoresHoldTheCargoUntilASiloOpens(t *testing.T) {
	s := newGame()
	s.Stock.Oil = oilCap(s)
	id := s.NextID
	s.NextID++
	x, y := parkSpot(id)
	s.Robots[id] = Robot{
		ID: id, Kind: RobotCore, X: x, Y: y,
		Carry: 30, Cargo: TypeOil, PostCol: -1, PostRow: -1,
	}
	runTicks(s, 10)
	if r := s.Robots[id]; r.Carry != 30 {
		t.Errorf("the robot left %v L in its arms against a full store, want 30",
			r.Carry)
	}
	if s.Stock.Oil != coreOilCap {
		t.Errorf("the full store took %v L more", s.Stock.Oil-coreOilCap)
	}
	// A silo opens room, and the waiting load lands in it.
	sid := s.NextID
	s.NextID++
	s.Buildings[sid] = Building{
		ID: sid, Kind: BuildingSilo, Col: coreCol + 2, Row: coreRow + 2,
	}
	Apply(s, Tick{})
	if r := s.Robots[id]; r.Carry != 0 {
		t.Errorf("the robot still holds %v L with room in the silo", r.Carry)
	}
	if want := coreOilCap + 30; s.Stock.Oil != want {
		t.Errorf("the stores hold %v L, want %v", s.Stock.Oil, want)
	}
}

// TestAProtectorsBubbleStandsOnItsCell pins the drawn bubble to the cell
// the protector was raised on. It once drew at the cell's index read as
// tiles - eight times too far, off the region - so the post rose and no
// protected radius ever showed around it.
func TestAProtectorsBubbleStandsOnItsCell(t *testing.T) {
	b := Building{Kind: BuildingProtector, Col: 147, Row: 100}
	cx, cy := protectorBubbleCenter(b)
	gx, gy := projectBuilding(b)
	if cx != gx || cy != gy {
		t.Errorf("the protector's bubble centers at %v, %v, want the building's ground %v, %v",
			cx, cy, gx, gy)
	}
}
