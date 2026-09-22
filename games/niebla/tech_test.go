package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"golib"
)

// arriveAll grants every drop of the ladder, opened, so a test can
// build what it needs without waiting for the clock or the rivals.
func arriveAll(s *State) {
	s.Tech = map[string]bool{}
	for i := range techLadder {
		s.Tech[techLadder[i].id] = true
	}
}

func TestSchematicsArriveWithTheFirstDelivery(t *testing.T) {
	s := newGame()
	if techPending(s) != "" {
		t.Fatalf("a new game has schematics waiting before their time: %q",
			techPending(s))
	}
	for _, kind := range (kinds{
		BuildingSilo, BuildingWarehouse, BuildingCharger, BuildingGuard,
		BuildingProtector, BuildingPump, BuildingFactory,
		BuildingWarFactory, BuildingArtillery,
	}) {
		if kindUnlocked(s, kind) {
			t.Fatalf("a new game can raise %s", kind)
		}
	}
	// Time alone brings nothing: the first load home does.
	runTicks(s, 60*120)
	if techPending(s) != "" {
		t.Fatalf("the schematics came on the clock, with no delivery yet: %q",
			techPending(s))
	}
	col, row, _ := nearestTileOf(kindOil)
	Apply(s, SendRobot{Col: col, Row: row})
	if !tickUntil(s, 60*120, func() bool { return s.Deliveries > 0 }) {
		t.Fatal("the robot never delivered its first load")
	}
	if techPending(s) != techInfraID {
		t.Fatalf("the first delivery brought %q, want %q",
			techPending(s), techInfraID)
	}
	for _, kind := range (kinds{BuildingSilo, BuildingWarehouse, BuildingCharger}) {
		if !kindUnlocked(s, kind) {
			t.Errorf("%s did not arrive with infrastructure", kind)
		}
	}
	if kindUnlocked(s, BuildingGuard) || kindUnlocked(s, BuildingPump) {
		t.Error("infrastructure brought the guard post or the pump with it")
	}
}

// kinds is a list of building kinds, so a test can loop over one.
type kinds []BuildingKind

func TestTheGuardPostAnswersTheMark(t *testing.T) {
	s := newGame()
	if kindUnlocked(s, BuildingGuard) {
		t.Fatal("the guard post stood ready before anyone came")
	}
	visitNow(s)
	if !tickUntil(s, 60*60, func() bool { return len(s.Parties) > 0 }) {
		t.Fatal("the scout never drove in")
	}
	// Driving in is not enough: the drawing must be inevitable first.
	runTicks(s, 60)
	if kindUnlocked(s, BuildingGuard) {
		t.Fatal("the guard post came while the scout was still driving in")
	}
	// A rival at the tanks, drinking, makes it inevitable: there the
	// guard comes, while the thief steals and before the mark dries.
	s.Enemies[7] = Enemy{ID: 7, Party: 1, Kind: EnemyScout, X: 100, Y: 100, Oil: 3}
	runTicks(s, 1)
	if !kindUnlocked(s, BuildingGuard) {
		t.Fatal("a rival drinking at the tanks brought no guard post")
	}
	if kindUnlocked(s, BuildingProtector) || kindUnlocked(s, BuildingWarFactory) {
		t.Error("more than the guard post came with the mark")
	}
	// And a mark already on the ground brings it too, should the moment
	// have passed while the state stood still.
	marked := newGame()
	marked.Marks[1] = Mark{ID: 1, X: 100, Y: 100}
	runTicks(marked, 1)
	if !kindUnlocked(marked, BuildingGuard) {
		t.Fatal("a mark on the ground brought no guard post")
	}
}

func TestTheFrontierKitAndTheFactoryComeOnTheirClock(t *testing.T) {
	s := newGame()
	runTicks(s, techFrontierTicks)
	if !kindUnlocked(s, BuildingProtector) || !kindUnlocked(s, BuildingPump) {
		t.Fatal("the frontier kit never arrived at its tick")
	}
	if kindUnlocked(s, BuildingFactory) {
		t.Fatal("the factory came with the frontier kit")
	}
	runTicks(s, techIndustryTicks-techFrontierTicks)
	if !kindUnlocked(s, BuildingFactory) {
		t.Fatal("the factory never arrived at its tick")
	}
}

func TestMobileUnitsWaitForTheFirstRaidToLeave(t *testing.T) {
	s := newGame()
	s.Raids.Visits = 1
	runTicks(s, 1)
	if kindUnlocked(s, BuildingWarFactory) {
		t.Fatal("the war factory came after the scout, not after the first raid")
	}
	s.Raids.Visits = 2
	runTicks(s, 1)
	if !kindUnlocked(s, BuildingWarFactory) {
		t.Fatal("the war factory never came after the first raid")
	}
}

func TestArtilleryWaitsForABase(t *testing.T) {
	s := newGame()
	if dropArrived(s, techArtilleryID) {
		t.Fatal("artillery came to a region with no base")
	}
	s.Parties[9] = Party{ID: 9, Stage: StageSettled}
	stepTech(s) // the wiring is pinned by the clock test; this is the trigger
	if !dropArrived(s, techArtilleryID) || !kindUnlocked(s, BuildingArtillery) {
		t.Fatal("a base settled and artillery never came")
	}
}

func TestAMarkedBlueprintRefusesWhatIsLocked(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundNearCore()
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: col, Row: row})
	if len(s.Jobs) != 0 {
		t.Fatalf("a locked silo was marked: %d jobs", len(s.Jobs))
	}
	if s.Stock.Lilac != 1200 {
		t.Fatalf("a locked silo was paid for: %v lilac left", s.Stock.Lilac)
	}
	arriveAll(s)
	Apply(s, MarkBuilding{Kind: BuildingSilo, Col: col, Row: row})
	if len(s.Jobs) != 1 || s.Jobs[0].Kind != BuildingSilo {
		t.Fatalf("an arrived silo was not marked: %d jobs", len(s.Jobs))
	}
}

func TestAPipeWaitsForTheFrontierKit(t *testing.T) {
	s := newGame()
	seedStock(s)
	Apply(s, LayPipe{From: coreTank, To: 0})
	if len(s.Pipes) != 0 {
		t.Fatalf("a pipe was marked with no frontier kit: %d pipes", len(s.Pipes))
	}
	arriveAll(s)
	col, row := groundNearCore()
	silo := raised(t, s, BuildingSilo, col, row)
	Apply(s, LayPipe{From: coreTank, To: silo.ID})
	if len(s.Pipes) != 1 {
		t.Fatalf("a pipe was refused once the frontier kit came: %d pipes",
			len(s.Pipes))
	}
}

func TestTheBadgeOpensAndAStaleAckDoesNothing(t *testing.T) {
	s := newGame()
	Apply(s, DevNextTech{})
	runTicks(s, 1)
	if got := techPending(s); got != techInfraID {
		t.Fatalf("pending %q, want %q", got, techInfraID)
	}
	Apply(s, AckTech{ID: techArtilleryID})
	if _, ok := s.Tech[techArtilleryID]; ok {
		t.Fatal("an ack opened schematics that never arrived")
	}
	Apply(s, AckTech{ID: techInfraID})
	if opened := s.Tech[techInfraID]; !opened {
		t.Fatal("the ack did not open the drop")
	}
	if techPending(s) != "" {
		t.Fatalf("the badge still waits after its ack: %q", techPending(s))
	}
}

func TestASaveFromBeforeTheSchematicsOpensWhatItEarned(t *testing.T) {
	s := newGame()
	s.Deliveries = 3
	s.Ticks = techIndustryTicks + 60
	s.Raids.Visits = 2
	s.Tech = nil
	Apply(s, Tick{})
	for _, id := range []string{techInfraID, techFrontierID, techIndustryID, techMobileID} {
		if s.Tech[id] != true {
			t.Errorf("the old save's %s is %v, want opened", id, s.Tech[id])
		}
	}
	if _, ok := s.Tech[techGuardID]; ok {
		t.Error("the guard post came to an old save with nobody driving in")
	}
	if _, ok := s.Tech[techArtilleryID]; ok {
		t.Error("artillery came to an old save with no base")
	}
	// No badge waits: the guard post comes when a scout paints its mark,
	// and this region has none on the ground yet.
	if techPending(s) != "" {
		t.Errorf("an old save owes a click: pending %q", techPending(s))
	}
	s.Marks[1] = Mark{ID: 1, X: 100, Y: 100}
	Apply(s, Tick{})
	if opened := s.Tech[techGuardID]; opened {
		t.Error("the mark brought the guard post already opened")
	}
	if techPending(s) != techGuardID {
		t.Errorf("the mark's badge is %q, want the guard post's", techPending(s))
	}
}

func TestDevNextTechBringsTheLadderInOrder(t *testing.T) {
	s := newGame()
	for i := range techLadder {
		Apply(s, DevNextTech{})
		runTicks(s, 1)
		if got := techPending(s); got != techLadder[i].id {
			t.Fatalf("drop %d: pending %q, want %q", i, got, techLadder[i].id)
		}
		Apply(s, AckTech{ID: techLadder[i].id})
		if techPending(s) != "" {
			t.Fatalf("drop %d: the badge waits after its ack", i)
		}
	}
	Apply(s, DevNextTech{})
	for i := range techLadder {
		if _, ok := s.Tech[techLadder[i].id]; !ok {
			t.Errorf("drop %d never arrived through the dev tool", i)
		}
	}
}

func TestCardButtonsWaitForTheirSchematics(t *testing.T) {
	s := newGame()
	seedStock(s)
	camera := golib.NewCamera(screenWidth, screenHeight)
	col, row := groundNearCore()
	raised(t, s, BuildingSilo, col, row)
	// The silo's lay pipe waits for the frontier kit, the pool's build
	// pump for its own drop; neither is offered before theirs arrives.
	panel := tooltipLayout(s, camera, col, row, map[string]bool{})
	if panel.findButton(buttonLayPipe) != nil {
		t.Error("lay pipe is offered with no frontier kit")
	}
	d := safePool(t)
	dc, dr := tileCell(d.Col, d.Row)
	panel = tooltipLayout(s, camera, dc, dr, map[string]bool{})
	if panel.findButton(buttonBuildPump) != nil {
		t.Error("build pump is offered with no schematics")
	}
	arriveAll(s)
	panel = tooltipLayout(s, camera, col, row, map[string]bool{})
	if panel.findButton(buttonLayPipe) == nil {
		t.Error("lay pipe is still missing once the frontier kit arrived")
	}
	// An unaffordable option is not offered either.
	lilac := s.Stock.Lilac
	s.Stock.Lilac = 0
	panel = tooltipLayout(s, camera, dc, dr, map[string]bool{})
	if panel.findButton(buttonBuildPump) != nil {
		t.Error("build pump is offered on stores that can't pay it")
	}
	s.Stock.Lilac = lilac
}

func TestTheTechSurvivesARoundTrip(t *testing.T) {
	s := newGame()
	Apply(s, DevNextTech{})
	runTicks(s, 1)
	Apply(s, AckTech{ID: techInfraID})
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("the state does not marshal: %v", err)
	}
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("the state does not unmarshal: %v", err)
	}
	if !reflect.DeepEqual(back.Tech, s.Tech) {
		t.Errorf("the tech ledger changed across a JSON round trip: %v vs %v",
			back.Tech, s.Tech)
	}
}
