package main

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

// safePool returns the oil pool inside the core's bubble.
func safePool(t *testing.T) Deposit {
	t.Helper()
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil pool")
	}
	d, _ := depositAt(col, row)
	return d
}

// pumpOn puts a finished pump on a pool's middle, skipping the robots'
// work.
func pumpOn(t *testing.T, s *State, d Deposit) Building {
	t.Helper()
	col, row := pumpCell(d)
	return raised(t, s, BuildingPump, col, row)
}

// pipeOut returns the first pipe that leaves an end.
func pipeOut(s *State, from int64) (Pipe, bool) {
	for _, p := range pipesOf(s, from) {
		if p.From == from {
			return p, true
		}
	}
	return Pipe{}, false
}

// laid runs the simulation until the robots have laid every pipe.
func laid(t *testing.T, s *State) {
	t.Helper()
	done := func() bool {
		_, owed := unlaidPipe(s)
		return !owed
	}
	if !tickUntil(s, 60*600, done) {
		t.Fatalf("the robots never laid the pipes: %v", s.Pipes)
	}
}

// layNow marks a pipe and lays it at once, skipping the robots' work.
func layNow(t *testing.T, s *State, from, to int64) Pipe {
	t.Helper()
	before := len(s.Pipes)
	Apply(s, LayPipe{From: from, To: to})
	if len(s.Pipes) != before+1 {
		t.Fatalf("no pipe was laid from %d to %d", from, to)
	}
	for _, p := range pipesOf(s, from) {
		if p.To == to {
			p.Left = 0
			s.Pipes[p.ID] = p
			return p
		}
	}
	t.Fatal("the pipe just laid is gone")
	return Pipe{}
}

func TestAPumpStandsOnAPoolAndAPoolTakesOne(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundNearCore()
	if canPlace(s, BuildingPump, col, row) {
		t.Error("a pump was taken on plain ground")
	}
	d := safePool(t)
	pc, pr := pumpCell(d)
	if tcol, trow := cellTile(pc, pr); tileAt(tcol, trow) != kindOil {
		t.Fatalf("the pump's cell %d, %d stands off its pool", pc, pr)
	}
	if canPlace(s, BuildingSilo, pc, pr) {
		t.Error("a silo was taken on oil")
	}
	if !canPlace(s, BuildingPump, pc, pr) {
		t.Fatal("the safe pool refuses its pump")
	}
	Apply(s, MarkBuilding{Kind: BuildingPump, Col: pc, Row: pr})
	if len(s.Jobs) != 1 || s.Stock.Lilac != 1200-pumpCostLilac {
		t.Fatalf("marking a pump left %d jobs and %v kg", len(s.Jobs), s.Stock.Lilac)
	}
	if canPlace(s, BuildingPump, pc+1, pr) {
		t.Error("a pool with a pump rising took a second one")
	}
	for _, far := range regionDeposits {
		if far.Kind != kindOil || far == d {
			continue
		}
		if fc, fr := pumpCell(far); canPlace(s, BuildingPump, fc, fr) {
			t.Errorf("the pool at %d, %d took a pump outside every bubble",
				far.Col, far.Row)
		}
	}
}

func TestAPipeIsPaidBySectionAndLaidByTheRobots(t *testing.T) {
	s := newGame()
	seedStock(s)
	pump := pumpOn(t, s, safePool(t))
	from, _ := pipeEndSpot(s, pump.ID)
	bends := []PipePoint{{from.X + 120, from.Y - 60}}
	sections, ok := canLayPipe(s, pump.ID, 0, bends)
	if !ok || sections < 2 {
		t.Fatalf("the pump can't take a pipe to the core: %d sections, %v", sections, ok)
	}
	Apply(s, LayPipe{From: pump.ID, To: 0, Bends: bends})
	p, piped := pipeOut(s, pump.ID)
	if !piped {
		t.Fatal("laying left no pipe")
	}
	if want := 1200 - float64(sections)*pipeSectionLilac; s.Stock.Lilac != want {
		t.Errorf("the stores hold %v kg, want %v: %d sections paid",
			s.Stock.Lilac, want, sections)
	}
	if p.Left != sections*pipeSectionWorkTicks {
		t.Errorf("the pipe asks %d ticks, want %d", p.Left, sections*pipeSectionWorkTicks)
	}
	oil := s.Stock.Oil
	runTicks(s, 120)
	if s.Stock.Oil != oil {
		t.Error("oil ran down a pipe nobody has laid yet")
	}
	if pumpStatus(s, pump) != pumpLaying {
		t.Errorf("the pump says %q while its pipe is laid", pumpStatus(s, pump))
	}
	laid(t, s)
	if _, again := canLayPipe(s, pump.ID, 0, nil); again {
		t.Error("two ends took a second pipe between them")
	}
}

func TestALaidPipeCarriesThePoolIntoItsTank(t *testing.T) {
	s := newGame()
	seedStock(s)
	d := safePool(t)
	pump := pumpOn(t, s, d)
	Apply(s, LayPipe{From: pump.ID, To: coreTank})
	laid(t, s)
	oil, pool := s.Stock.Oil, s.Drain[depositKey(d)]
	runTicks(s, 600)
	gained := s.Stock.Oil - oil
	if want := pumpLitersPerSecond * 10; math.Abs(gained-want) > 0.001 {
		t.Errorf("ten seconds of pumping brought %v L, want %v", gained, want)
	}
	if lost := pool - s.Drain[depositKey(d)]; math.Abs(lost-gained) > 0.001 {
		t.Errorf("the pool lost %v L and the core gained %v", lost, gained)
	}
	for _, r := range s.Robots {
		if r.Carry > 0 {
			t.Errorf("robot %d carries %v: the pipe's oil rides in nobody's arms",
				r.ID, r.Carry)
		}
	}
	// A full tank at the pipe's end stops the pump, and nothing is lost.
	s.Stock.Oil = coreOilCap - 1
	pool = s.Drain[depositKey(d)]
	runTicks(s, 600)
	if s.Stock.Oil != coreOilCap {
		t.Errorf("the core holds %v L in a tank of %v", s.Stock.Oil, coreOilCap)
	}
	if lost := pool - s.Drain[depositKey(d)]; math.Abs(lost-1) > 0.001 {
		t.Errorf("the pool lost %v L for 1 L of room", lost)
	}
	if pumpStatus(s, pump) != pumpBlocked {
		t.Errorf("the pump says %q into a full tank", pumpStatus(s, pump))
	}
	// A dry pool stops it too.
	s.Stock.Oil = 0
	s.Drain[depositKey(d)] = 0.01
	runTicks(s, 60)
	if s.Drain[depositKey(d)] != 0 || pumpStatus(s, pump) != pumpDry {
		t.Errorf("the pool holds %v L and the pump says %q",
			s.Drain[depositKey(d)], pumpStatus(s, pump))
	}
}

func TestOilHasAPlaceAndPipesMoveItBetweenTanks(t *testing.T) {
	s := newGame()
	seedStock(s)
	delete(s.Robots, 1) // nobody hauls or refuels: the pipes alone move oil
	delete(s.Robots, 2)
	col, row := groundNearCore()
	first := raised(t, s, BuildingSilo, col, row)
	second := raised(t, s, BuildingSilo, col+2, row)
	charger := raised(t, s, BuildingCharger, col+4, row)
	if got, want := oilCap(s), coreOilCap+2*siloOilCap+chargerOilCap; got != want {
		t.Fatalf("the colony's tanks hold %v L at most, want %v", got, want)
	}
	// The core feeds a silo, which feeds another silo and a charger.
	layNow(t, s, coreTank, first.ID)
	layNow(t, s, first.ID, second.ID)
	layNow(t, s, first.ID, charger.ID)
	total := oilTotal(s)
	runTicks(s, 60*400)
	if math.Abs(oilTotal(s)-total) > 0.001 {
		t.Errorf("the colony holds %v L after piping, had %v: pipes lose nothing",
			oilTotal(s), total)
	}
	if s.Stock.Oil > 0.001 || s.Buildings[first.ID].Oil > 0.001 {
		t.Errorf("the core holds %v L and the first silo %v: the oil ran on",
			s.Stock.Oil, s.Buildings[first.ID].Oil)
	}
	if got := s.Buildings[charger.ID].Oil; math.Abs(got-chargerOilCap) > 0.001 {
		t.Errorf("the charger holds %v L, want its tank full at %v", got, chargerOilCap)
	}
	if got := s.Buildings[second.ID].Oil; math.Abs(got-(600-chargerOilCap)) > 0.001 {
		t.Errorf("the second silo holds %v L, want the rest, %v", got, 600-chargerOilCap)
	}
	// What the colony pays comes out of any tank.
	Apply(s, MarkBuilding{Kind: BuildingProtector, Col: col, Row: row + 3})
	if got := total - oilTotal(s); math.Abs(got-protectorCostOil) > 0.001 {
		t.Errorf("a protector took %v L off the tanks, want %v", got, protectorCostOil)
	}
	// A building takes so many pipes and no more, and two ends take one.
	third := raised(t, s, BuildingSilo, col+6, row)
	if canJoin(s, first.ID, third.ID) {
		t.Errorf("a silo with %d pipes took one more", pipePorts)
	}
	if canJoin(s, second.ID, first.ID) {
		t.Error("two silos took a second pipe between them, the other way")
	}
	if canJoin(s, second.ID, second.ID) {
		t.Error("a silo took a pipe to itself")
	}
	// A demolished tank drops its own oil, and its pipes with it.
	held := s.Buildings[second.ID].Oil
	Apply(s, Demolish{Building: second.ID})
	pile, _ := pileAt(s, col+2, row)
	if math.Abs(pile.Oil-held) > 0.001 {
		t.Errorf("the pile holds %v L, want the silo's %v", pile.Oil, held)
	}
	if len(pipesOf(s, first.ID)) != 2 {
		t.Errorf("the first silo keeps %d pipes, want 2", len(pipesOf(s, first.ID)))
	}
}

func TestRobotsCarryOilToATankWithRoomAndRefillFromTheirPost(t *testing.T) {
	s := newGame()
	col, row := groundNearCore()
	silo := raised(t, s, BuildingSilo, col, row)
	s.Stock.Oil = coreOilCap
	x, y := parkSpot(1)
	r := s.Robots[1]
	r.X, r.Y, r.Carry, r.Cargo = x, y, 20, TypeOil
	s.Robots[1] = r
	if got := haulTank(s, r); got != silo.ID {
		t.Fatalf("a robot by a full core carries its oil to tank %d, want the silo", got)
	}
	if !tickUntil(s, 60*120, func() bool { return s.Robots[1].Carry == 0 }) {
		t.Fatal("the robot never unloaded")
	}
	if s.Buildings[silo.ID].Oil != 20 || s.Stock.Oil != coreOilCap {
		t.Errorf("the silo holds %v L and the core %v, want the load in the silo",
			s.Buildings[silo.ID].Oil, s.Stock.Oil)
	}
	// A dry charger sends a thirsty robot on to a post with oil.
	charger := raised(t, s, BuildingCharger, col+3, row)
	cx, cy := cellCenterUnits(charger.Col, charger.Row)
	thirsty := Robot{Kind: RobotBuilt, X: cx + 5, Y: cy, Tank: 10}
	if got := refuelTank(s, thirsty); got != coreTank {
		t.Errorf("a robot by a dry charger refills at tank %d, want the core", got)
	}
	b := s.Buildings[charger.ID]
	b.Oil = 50
	s.Buildings[charger.ID] = b
	if got := refuelTank(s, thirsty); got != charger.ID {
		t.Errorf("a robot by a charger with oil refills at tank %d, want it", got)
	}
}

func TestLayPipeRefusesWhatItCantJoin(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundNearCore()
	charger := raised(t, s, BuildingCharger, col, row)
	pump := pumpOn(t, s, safePool(t))
	refused := []LayPipe{
		{From: pump.ID, To: pump.ID},
		{From: charger.ID, To: pump.ID},
		{From: 998, To: coreTank},
		{From: pump.ID, To: 999},
		{From: pump.ID, To: 0, Bends: []PipePoint{{-5, 100}}},
		{From: pump.ID, To: 0, Bends: make([]PipePoint, pipeMaxBends+1)},
	}
	for _, a := range refused {
		Apply(s, a)
		if len(s.Pipes) != 0 || s.Stock.Lilac != 1200 {
			t.Fatalf("%+v was taken: %v, %v kg", a, s.Pipes, s.Stock.Lilac)
		}
	}
	s.Stock.Lilac = pipeSectionLilac - 1
	Apply(s, LayPipe{From: pump.ID, To: 0})
	if len(s.Pipes) != 0 {
		t.Error("a pipe was laid on stores that can't pay a section")
	}
}

func TestAPipeLeavesWithItsEndsAndItsCostFallsAsAPile(t *testing.T) {
	s := newGame()
	seedStock(s)
	pump := pumpOn(t, s, safePool(t))
	col, row := groundNearCore()
	silo := raised(t, s, BuildingSilo, col, row)
	Apply(s, LayPipe{From: pump.ID, To: silo.ID})
	p, _ := pipeOut(s, pump.ID)
	Apply(s, Demolish{Building: silo.ID})
	if len(s.Pipes) != 0 {
		t.Fatal("the pipe outlived the silo it ended at")
	}
	pile, _ := pileAt(s, col, row)
	if want := siloCostLilac + pipeCost(p.Sections); pile.Lilac != want {
		t.Errorf("the pile holds %v kg, want the silo's and the pipe's %v", pile.Lilac, want)
	}

	Apply(s, LayPipe{From: pump.ID, To: 0})
	p, _ = pipeOut(s, pump.ID)
	Apply(s, RemovePipe{Pipe: p.ID})
	if len(s.Pipes) != 0 {
		t.Fatal("the removed pipe is still there")
	}
	pile, _ = pileAt(s, pump.Col, pump.Row)
	if pile.Lilac != pipeCost(p.Sections) {
		t.Errorf("the pile by the pump holds %v kg, want %v", pile.Lilac, pipeCost(p.Sections))
	}

	Apply(s, LayPipe{From: pump.ID, To: 0})
	Apply(s, Demolish{Building: pump.ID})
	if len(s.Pipes) != 0 {
		t.Error("the pipe outlived its pump")
	}
}

func TestAPipesCurvePassesThroughItsBends(t *testing.T) {
	from, to := PipePoint{100, 100}, PipePoint{900, 300}
	bends := []PipePoint{{300, 400}, {320, 410}, {700, 50}}
	path := pipePath(from, bends, to)
	if path[0] != from || pointGap(path[len(path)-1], to) > 1e-9 {
		t.Fatalf("the curve runs from %v to %v", path[0], path[len(path)-1])
	}
	for _, bend := range bends {
		nearest := math.Inf(1)
		for _, p := range path {
			nearest = math.Min(nearest, pointGap(p, bend))
		}
		if nearest > 1e-6 {
			t.Errorf("the curve misses the bend %v by %v u", bend, nearest)
		}
	}
	for _, p := range path {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) {
			t.Fatal("the curve holds a point that is no number")
		}
	}
	if !reflect.DeepEqual(path, pipePath(from, bends, to)) {
		t.Error("the same clicks drew two curves")
	}
	// No bends: a sag, not a ruler's line, and longer than one.
	sag := pipePath(from, nil, to)
	if pathLength(sag) <= pointGap(from, to) {
		t.Error("a pipe with no bends runs straight")
	}
	// Two clicks on one spot, or on an end, break nothing.
	twice := pipePath(from, []PipePoint{{300, 400}, {300, 400}, to}, to)
	for _, p := range twice {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) {
			t.Fatal("a doubled click broke the curve")
		}
	}
	half := pathPointAt(path, pathLength(path)/2)
	if half == from || half == to {
		t.Errorf("the curve's middle is its end: %v", half)
	}
	if pathPointAt(path, -10) != from || pathPointAt(path, 1e9) != path[len(path)-1] {
		t.Error("a point past the curve's ends isn't held at them")
	}
}

func TestPipesSurviveASave(t *testing.T) {
	s := newGame()
	seedStock(s)
	pump := pumpOn(t, s, safePool(t))
	from, _ := pipeEndSpot(s, pump.ID)
	Apply(s, LayPipe{
		From: pump.ID, To: 0, Bends: []PipePoint{{from.X + 80, from.Y - 40}},
	})
	runTicks(s, 300)
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("the state does not marshal: %v", err)
	}
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("the state does not unmarshal: %v", err)
	}
	if !reflect.DeepEqual(&back, s) {
		t.Error("the pipes changed across a JSON round trip")
	}
	// A save from before the pipes has no table for them, and takes one.
	old := newGame()
	seedStock(old)
	old.Pipes = nil
	oldPump := pumpOn(t, old, safePool(t))
	Apply(old, LayPipe{From: oldPump.ID, To: 0})
	if len(old.Pipes) != 1 {
		t.Error("a save with no pipe table refused a pipe")
	}
}

func TestAPoolsCardsCarryThePumpAndItsPipe(t *testing.T) {
	s := newGame()
	seedStock(s)
	d := safePool(t)
	camera := newPlayScene(s).camera
	panel := tooltipLayout(s, camera, d.Col, d.Row, map[string]bool{})
	if panel.findButton(buttonBuildPump) == nil {
		t.Fatal("a pool with no pump offers none")
	}
	s.Stock.Lilac = 0
	panel = tooltipLayout(s, camera, d.Col, d.Row, map[string]bool{})
	if row := panel.findButton(buttonBuildPump); row == nil || !row.dim {
		t.Error("empty stores don't dim the pump's button")
	}
	seedStock(s)
	pump := pumpOn(t, s, d)
	// The pool is one thing: every tile of it shows its pump.
	for row := d.Row; row < d.Row+d.Rows; row++ {
		for col := d.Col; col < d.Col+d.Cols; col++ {
			panel = tooltipLayout(s, camera, col, row, map[string]bool{})
			if panel.findButton(buttonLayPipe) == nil {
				t.Errorf("tile %d, %d of the pool doesn't show its pump", col, row)
			}
			if panel.findButton(buttonBuildPump) != nil {
				t.Errorf("tile %d, %d offers a second pump", col, row)
			}
		}
	}
	Apply(s, LayPipe{From: pump.ID, To: 0})
	panel = tooltipLayout(s, camera, d.Col, d.Row, map[string]bool{})
	if panel.findButton(buttonRemovePipe) == nil {
		t.Error("a piped pump doesn't offer to remove its pipe")
	}
}
