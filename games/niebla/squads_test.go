package main

import (
	"math"
	"testing"
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
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: col + 3, Row: row})
	s.dropPile(col+5, row, 0, 50)
	oilCol, oilRow, _ := nearestTileOf(kindOil)
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	runTicks(s, 60*30)
	if len(s.Jobs) != 1 || len(s.Piles) != 1 {
		t.Errorf("%d jobs and %d piles left, want the troopers to touch neither",
			len(s.Jobs), len(s.Piles))
	}
	if _, owned := postOwner(s, oilCol, oilRow); owned {
		t.Errorf("a trooper took a post")
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
		wreck = wreck || p.Lilac == trooperWreckLilac
	}
	if !wreck {
		t.Errorf("the fallen trooper left no wreck: %+v", s.Piles)
	}
	survivor := s.Robots[s.spawnRobot(RobotCombat, 2500, 2500)]
	survivor.Squad = home.ID
	s.Robots[survivor.ID] = survivor
	Apply(s, Demolish{Building: home.ID})
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
