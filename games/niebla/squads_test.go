package main

import (
	"math"
	"testing"

	"golib"
)

// squadOfTroopers raises a war factory by the core and puts n troopers
// in its squad, at its door.
func squadOfTroopers(t *testing.T, s *State, n int) Building {
	t.Helper()
	col, row := groundNearCore()
	home := raised(t, s, BuildingWarFactory, col, row)
	x, y := cellCenterUnits(col, row)
	for i := 0; i < n; i++ {
		r := s.Robots[s.spawnRobot(RobotCombat, x, y)]
		r.Squad = home.ID
		s.Robots[r.ID] = r
	}
	return home
}

func crawlerOf(s *State) Enemy {
	for _, id := range sortedEnemyIDs(s) {
		if e := s.Enemies[id]; e.Kind == EnemyCrawler {
			return e
		}
	}
	return Enemy{}
}

func TestAWarFactoryBuildsItsSquadAndNoMore(t *testing.T) {
	s := newGame()
	noRivals(s)
	s.Stock = Stock{Oil: 1000, Lilac: 2500}
	arriveAll(s)
	col, row := groundNearCore()
	home := raised(t, s, BuildingWarFactory, col, row)
	lilac, oil := s.Stock.Lilac, oilTotal(s)
	Apply(s, QueueRobot{Building: home.ID})
	if s.Stock.Lilac != lilac-trooperCostLilac || oilTotal(s) != oil-trooperCostOil {
		t.Errorf("a trooper cost %v kg and %v L, want %v and %v",
			lilac-s.Stock.Lilac, oil-oilTotal(s), trooperCostLilac, trooperCostOil)
	}
	runTicks(s, trooperBuildTicks)
	members := squadMembers(s, home.ID)
	if len(members) != 1 || members[0].Health != trooperHealth ||
		members[0].Tank <= 0 {
		t.Fatalf("the war factory's squad is %+v, want one whole trooper with oil", members)
	}
	for i := 1; i < squadSize+2; i++ {
		s.Stock = Stock{Oil: 1000, Lilac: 2500}
		Apply(s, QueueRobot{Building: home.ID})
		runTicks(s, trooperBuildTicks)
	}
	if got := len(squadMembers(s, home.ID)); got != squadSize {
		t.Errorf("the squad grew to %d, want it full at %d", got, squadSize)
	}
	// Troopers do no work: a job, a pile and a post are none of theirs.
	for _, id := range sortedRobotIDs(s) {
		if s.Robots[id].Kind != RobotCombat {
			delete(s.Robots, id)
		}
	}
	addWorker(s)
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: col + 3, Row: row})
	s.dropPile(col+5, row, 0, 50)
	oilCol, oilRow, _ := nearestTileOf(kindOil)
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	runTicks(s, 60*30)
	if len(s.Jobs) != 1 || len(s.Piles) != 1 {
		t.Errorf("%d jobs and %d piles left, want the troopers to touch neither",
			len(s.Jobs), len(s.Piles))
	}
	for _, robot := range postRobots(s, oilCol, oilRow) {
		if robot.Kind == RobotCombat {
			t.Errorf("trooper %d took a deposit post", robot.ID)
		}
	}
}

func TestWarFactoryBuildsOneMechanicSeparatelyFromItsSquad(t *testing.T) {
	s := newGame()
	noRivals(s)
	s.Stock = Stock{Oil: 1000, Lilac: 2500}
	col, row := groundNearCore()
	home := raised(t, s, BuildingWarFactory, col, row)
	Apply(s, QueueMechanic{Building: home.ID})
	if b := s.Buildings[home.ID]; b.WorkKind != RobotRepair ||
		b.Work != mechanicBuildTicks {
		t.Fatalf("the war factory has queued %+v, want a mechanic", b)
	}
	if !squadRoom(s, s.Buildings[home.ID]) {
		t.Fatal("building a mechanic used a slot in the trooper squad")
	}
	runTicks(s, mechanicBuildTicks)
	mechanic, found := mechanicForFactory(s, home.ID)
	if !found || mechanic.Kind != RobotRepair || mechanic.Factory != home.ID {
		t.Fatalf("the mechanic is %+v, found %v", mechanic, found)
	}
	if len(squadMembers(s, home.ID)) != 0 {
		t.Fatal("the mechanic joined its factory's combat squad")
	}
	if mechanicRoom(s, s.Buildings[home.ID]) {
		t.Fatal("the war factory offered a second mechanic")
	}

	stock := s.Stock
	Apply(s, QueueMechanic{Building: home.ID})
	if s.Stock != stock || s.Buildings[home.ID].Work > 0 {
		t.Fatal("a full mechanic slot accepted another mechanic")
	}
	Apply(s, QueueRobot{Building: home.ID})
	if b := s.Buildings[home.ID]; b.WorkKind != RobotCombat ||
		b.Work != trooperBuildTicks {
		t.Fatalf("the war factory could not queue a trooper: %+v", b)
	}
}

func TestRivalGunsCanChooseMechanicsButIgnoreWorkers(t *testing.T) {
	s := newGame()
	col, row := groundNearCore()
	home := raised(t, s, BuildingWarFactory, col, row)
	x, y := cellCenterUnits(col, row)
	mechanicID := s.spawnRobot(RobotRepair, x, y)
	mechanic := s.Robots[mechanicID]
	mechanic.Factory = home.ID
	s.Robots[mechanicID] = mechanic
	s.Enemies[900] = Enemy{
		ID: 900, Kind: EnemyRaider, X: x + 40, Y: y,
		Health: enemySpecOf(EnemyRaider).health,
	}
	stepEnemyGuns(s)
	shotFound := false
	for _, shot := range s.Shots {
		if shot.Robot == mechanicID {
			shotFound = true
		}
	}
	if !shotFound {
		t.Fatal("the raider did not target the exposed mechanic")
	}
	if target, found := nearestColonyUnit(s, x+40, y, 110); !found ||
		target.Kind != RobotRepair {
		t.Fatalf("nearest combat target is %+v, found %v", target, found)
	}
}

func TestASquadGuardsWhereItIsToldAndMendsUnderABubble(t *testing.T) {
	s := newGame()
	noRivals(s)
	home := squadOfTroopers(t, s, 3)
	x, y := cellCenterUnits(home.Col, home.Row)
	Apply(s, OrderSquad{Squad: home.ID, X: x + 300, Y: y + 100})
	for _, r := range squadMembers(s, home.ID) {
		r.Health = 10
		s.Robots[r.ID] = r
	}
	runTicks(s, 60*30)
	for _, r := range squadMembers(s, home.ID) {
		if gap := math.Hypot(r.X-(x+300), r.Y-(y+100)); gap > 40 {
			t.Errorf("trooper %d stands %v u from the spot it guards", r.ID, gap)
		}
		if r.Health <= 10 {
			t.Errorf("trooper %d didn't mend at its spot: %v health", r.ID, r.Health)
		}
	}
	// An order for a building that is no war factory is no order.
	Apply(s, OrderSquad{Squad: 9999, X: 10, Y: 10})
	if len(s.Squads) != 1 {
		t.Errorf("%d squads hold an order, want the one", len(s.Squads))
	}
}

func TestASquadAttacksTheCrawlerFirstAndTheFogTakesTheRest(t *testing.T) {
	s := newGame()
	home := squadOfTroopers(t, s, 6)
	s.Raids.Visits = 1
	visitNow(s)
	tickUntil(s, 60*600, func() bool { return lastReport(s).Kind == ReportCamp })
	crawler := crawlerOf(s)
	Apply(s, OrderSquad{Squad: home.ID, Enemy: crawler.ID})
	if sq := s.Squads[home.ID]; sq.Order != OrderAttack || sq.Focus != crawler.ID ||
		sq.Party != crawler.Party {
		t.Fatalf("the squad's order is %+v, want an attack on the crawler's party", sq)
	}
	if !tickUntil(s, 60*300, func() bool { return crawlerOf(s).ID == 0 }) {
		t.Fatalf("the squad never brought the crawler down")
	}
	if len(s.Enemies) != raidFirstRaiders {
		t.Errorf("%d raiders stand when the crawler falls, want all %d: it goes first",
			len(s.Enemies), raidFirstRaiders)
	}
	hurt := false
	for _, r := range squadMembers(s, home.ID) {
		hurt = hurt || r.Health < trooperHealth
		if r.Tank >= robotTankLiters-trooperShotOil {
			t.Errorf("trooper %d's tank is full after a fight", r.ID)
		}
	}
	if !hurt && len(squadMembers(s, home.ID)) == 6 {
		t.Errorf("the rivals never shot back")
	}
	if !tickUntil(s, 60*300, func() bool { return len(s.Parties) == 0 }) {
		t.Fatalf("the party never ended")
	}
	if lastReport(s).Kind != ReportDestroyed && lastReport(s).Kind != ReportLeft {
		t.Errorf("the last report is %q", lastReport(s).Kind)
	}
	// With nobody left to attack the squad goes back to its door.
	runTicks(s, 2)
	if _, ordered := s.Squads[home.ID]; ordered {
		t.Errorf("the squad keeps an order about a party that is gone")
	}
	if squadOf(s, home.ID).Order != OrderGuard {
		t.Errorf("the squad doesn't guard its war factory after the fight")
	}
}

func TestAFallenTrooperLeavesItsWreckAndADemolishedWarFactoryItsSquad(t *testing.T) {
	s := newGame()
	home := squadOfTroopers(t, s, 1)
	s.Raids.Visits = 1
	visitNow(s)
	tickUntil(s, 60*600, func() bool { return lastReport(s).Kind == ReportCamp })
	Apply(s, OrderSquad{Squad: home.ID, Enemy: crawlerOf(s).ID})
	if !tickUntil(s, 60*300, func() bool { return len(squadMembers(s, home.ID)) == 0 }) {
		t.Fatalf("one trooper against a whole raid never fell")
	}
	wreck := false
	for _, p := range s.Piles {
		wreck = wreck || p.Lilac == trooperCostLilac*unitWreckRefund
	}
	if !wreck {
		t.Errorf("the fallen trooper left no wreck: %+v", s.Piles)
	}
	survivor := s.Robots[s.spawnRobot(RobotCombat, 2500, 2500)]
	survivor.Squad = home.ID
	s.Robots[survivor.ID] = survivor
	demolishNow(t, s, home.ID)
	runTicks(s, 60*30)
	r := s.Robots[survivor.ID]
	px, py := parkCenter()
	if gap := math.Hypot(r.X-px, r.Y-py); gap > 40 {
		t.Errorf("a trooper with no war factory stands %v u from the core's ranks", gap)
	}
	if _, ordered := s.Squads[home.ID]; ordered {
		t.Errorf("a demolished war factory's squad keeps its order")
	}
}

func TestAFallenTrooperLeavesAQuarterOfItsResources(t *testing.T) {
	s := newGame()
	x, y := tileCenterUnits(2, 2)
	id := s.spawnRobot(RobotCombat, x, y)
	r := s.Robots[id]
	r.Tank = 80
	s.Robots[id] = r

	s.hurtColonyUnit(id, trooperHealth)
	col, row := robotCell(r)
	p, ok := pileAt(s, col, row)
	if !ok {
		t.Fatal("the fallen trooper left no wreck")
	}
	wantOil := (trooperCostOil + r.Tank) * unitWreckRefund
	wantLilac := trooperCostLilac * unitWreckRefund
	if math.Abs(p.Oil-wantOil) > 0.001 ||
		math.Abs(p.Lilac-wantLilac) > 0.001 {
		t.Errorf("the trooper's wreck holds %v L and %v kg, want %v and %v",
			p.Oil, p.Lilac, wantOil, wantLilac)
	}
}

func TestTheNumberKeysCallTheWarFactoriesOldestFirst(t *testing.T) {
	s := newGame()
	noRivals(s)
	if slots := squadSlots(s); len(slots) != 0 {
		t.Fatalf("%d squads stand before any war factory, want none", len(slots))
	}
	col, row := groundNearCore()
	first := raised(t, s, BuildingWarFactory, col, row)
	raised(t, s, BuildingSilo, col+1, row)
	second := raised(t, s, BuildingWarFactory, col+2, row)
	slots := squadSlots(s)
	if len(slots) != 2 || slots[0] != first.ID || slots[1] != second.ID {
		t.Errorf("the keys would call %v, want the two war factories oldest first",
			slots)
	}
}

func TestASquadsMarkPicksItsSquad(t *testing.T) {
	s := newGame()
	noRivals(s)
	home := squadOfTroopers(t, s, 2)
	scene := newPlayScene(s)
	x, y := cellCenterUnits(home.Col, home.Row)
	// The pennant of a squad nobody has ordered stands at its door.
	pole := dotRadius(14, scene.zoom, 10)
	gx, gy := project(float32(x), float32(y))
	at := scene.camera.ToScreen(golib.Vector2{X: gx, Y: gy - pole/2})
	if got, ok := scene.squadMarkAt(at.X, at.Y); !ok || got != home.ID {
		t.Errorf("the pennant picked squad %d, %v; want %d", got, ok, home.ID)
	}
	// The ring of an attack stands on the vehicle it is to shoot first.
	e := Enemy{ID: 900, Kind: EnemyScout, Party: 901, X: x + 400, Y: y + 200}
	s.Enemies[e.ID] = e
	s.Squads[home.ID] = Squad{
		Home: home.ID, Order: OrderAttack, Party: e.Party, Focus: e.ID,
	}
	rx, ry := project(float32(e.X), float32(e.Y))
	at = scene.camera.ToScreen(golib.Vector2{X: rx, Y: ry - 2*unitH})
	if got, ok := scene.squadMarkAt(at.X, at.Y); !ok || got != home.ID {
		t.Errorf("the ring picked squad %d, %v; want %d", got, ok, home.ID)
	}
	if _, ok := scene.squadMarkAt(4, 4); ok {
		t.Errorf("a mark answered at the screen's corner")
	}
}

func TestTheSquadsBoxesLieApartAndPickTheirSquad(t *testing.T) {
	for i := 0; i < 3; i++ {
		rect := squadBoxRect(i)
		if rect.X < 0 || rect.Y < 0 ||
			rect.X+rect.Width > screenWidth || rect.Y+rect.Height > screenHeight {
			t.Fatalf("box %d leaves the screen: %+v", i, rect)
		}
		for j := i + 1; j < 3; j++ {
			if rect.Overlaps(squadBoxRect(j)) {
				t.Errorf("boxes %d and %d lie on each other", i, j)
			}
		}
		mx, my := rect.X+rect.Width/2, rect.Y+rect.Height/2
		if got, ok := squadBoxAt(mx, my); !ok || got != i {
			t.Errorf("the middle of box %d picked %d, %v", i, got, ok)
		}
	}
	if _, ok := squadBoxAt(4, 4); ok {
		t.Errorf("a box answered at the screen's corner")
	}
}
