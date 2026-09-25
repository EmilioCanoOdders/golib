package main

import (
	"encoding/json"
	"os"
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

func TestFactorySchematicsArriveWithTheFirstDelivery(t *testing.T) {
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
	if techPending(s) != techIndustryID {
		t.Fatalf("the first delivery brought %q, want %q",
			techPending(s), techIndustryID)
	}
	if !kindUnlocked(s, BuildingFactory) {
		t.Fatal("the first delivery did not bring the robot factory")
	}
	for _, kind := range (kinds{
		BuildingSilo, BuildingWarehouse, BuildingCharger, BuildingGuard,
		BuildingProtector, BuildingPump, BuildingWarFactory,
		BuildingArtillery,
	}) {
		if kindUnlocked(s, kind) {
			t.Errorf("the first delivery also unlocked %s", kind)
		}
	}
	if hasBuiltWorker(s) {
		t.Fatal("a new colony already has a factory-built worker")
	}
}

// kinds is a list of building kinds, so a test can loop over one.
type kinds []BuildingKind

func TestTheFirstFactoryWorkerBringsInfrastructureBeforeTheScout(t *testing.T) {
	s := newGame()
	oilCol, oilRow, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the starting region has no oil pool")
	}
	lilacCol, lilacRow, ok := nearestTileOf(kindLilac)
	if !ok {
		t.Fatal("the starting region has no lilac vein")
	}
	Apply(s, SendRobot{Col: oilCol, Row: oilRow})
	Apply(s, SendRobot{Col: lilacCol, Row: lilacRow})
	factoryCol, factoryRow := groundNearCore()

	var deliveryAt, factoryAt, workerAt, infrastructureAt int64
	factoryMarked, workerQueued := false, false
	for s.Ticks < raidFirstScoutTicks {
		if s.Deliveries > 0 && deliveryAt == 0 {
			deliveryAt = s.Ticks
		}
		if kindUnlocked(s, BuildingFactory) && !factoryMarked {
			Apply(s, MarkBuilding{
				Kind: BuildingFactory, Col: factoryCol, Row: factoryRow,
			})
			if len(s.Jobs) == 0 {
				t.Fatal("the factory could not be marked after its schematics")
			}
			factoryAt, factoryMarked = s.Ticks, true
			Apply(s, AckTech{ID: techIndustryID})
		}
		if factoryMarked && !workerQueued {
			if factory, raised := buildingAt(s, factoryCol, factoryRow); raised {
				Apply(s, QueueRobot{Building: factory.ID})
				if s.Buildings[factory.ID].Work > 0 {
					workerQueued = true
				}
			}
		}

		Apply(s, Tick{})
		if hasBuiltWorker(s) && workerAt == 0 {
			workerAt = s.Ticks
		}
		if kindUnlocked(s, BuildingCharger) && infrastructureAt == 0 {
			infrastructureAt = s.Ticks
		}
		if len(s.Enemies) > 0 {
			break
		}
	}

	if deliveryAt == 0 {
		t.Fatal("the opening plan made no first delivery")
	}
	if factoryAt == 0 {
		t.Fatal("the opening plan did not mark the unlocked factory")
	}
	if !workerQueued {
		t.Fatal("the opening plan could not order a factory worker")
	}
	if workerAt == 0 {
		t.Fatal("the factory did not finish its first worker")
	}
	if infrastructureAt == 0 {
		t.Fatal("the first factory worker did not unlock infrastructure")
	}
	if infrastructureAt >= raidFirstScoutTicks {
		t.Fatalf("infrastructure arrived at tick %d, not before the scout at %d",
			infrastructureAt, raidFirstScoutTicks)
	}
	t.Logf(
		"opening milestones (seconds): delivery %.1f, factory %.1f, " +
			"worker %.1f, infrastructure %.1f, scout %.1f",
		float64(deliveryAt)/60,
		float64(factoryAt)/60,
		float64(workerAt)/60,
		float64(infrastructureAt)/60,
		float64(raidFirstScoutTicks)/60,
	)
}

func TestOpeningCalloutsExplainTheNextStep(t *testing.T) {
	title, body := techWords(techIndustryID)
	if title != "robot factory" ||
		body != "Build it on clear ground, then use its card to build a robot." {
		t.Fatalf("factory callout is %q, %q", title, body)
	}
	title, body = techWords(techInfraID)
	if title != "infrastructure" ||
		body != "Silos hold oil, warehouses hold lilac, and " +
			"chargers refill worker tanks." {
		t.Fatalf("infrastructure callout is %q, %q", title, body)
	}
}

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

func TestTheFrontierKitComesOnItsClock(t *testing.T) {
	s := newGame()
	runTicks(s, techFrontierTicks)
	if !kindUnlocked(s, BuildingProtector) || !kindUnlocked(s, BuildingPump) {
		t.Fatal("the frontier kit never arrived at its tick")
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

func TestArtilleryWaitsForARivalFactory(t *testing.T) {
	s := newGame()
	if dropArrived(s, techArtilleryID) {
		t.Fatal("artillery came to a region with no rival city")
	}
	s.Cities[9] = City{ID: 9}
	s.Enemies[10] = Enemy{ID: 10, Kind: EnemyBase, City: 9}
	s.Cities[9] = City{ID: 9, Stage: 1, BuildingIDs: []int64{10}}
	stepTech(s)
	if dropArrived(s, techArtilleryID) {
		t.Fatal("artillery came before the rival factory was built")
	}
	s.Enemies[11] = Enemy{ID: 11, Kind: EnemyCityFactory, City: 9}
	city := s.Cities[9]
	city.Stage = len(cityBuildOrder)
	city.BuildingIDs = append(city.BuildingIDs, 11)
	s.Cities[9] = city
	stepTech(s)
	if !dropArrived(s, techArtilleryID) || !kindUnlocked(s, BuildingArtillery) {
		t.Fatal("the rival factory completed but artillery never arrived")
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
	if got := techPending(s); got != techIndustryID {
		t.Fatalf("pending %q, want %q", got, techIndustryID)
	}
	Apply(s, AckTech{ID: techArtilleryID})
	if _, ok := s.Tech[techArtilleryID]; ok {
		t.Fatal("an ack opened schematics that never arrived")
	}
	Apply(s, AckTech{ID: techIndustryID})
	if opened := s.Tech[techIndustryID]; !opened {
		t.Fatal("the ack did not open the drop")
	}
	if techPending(s) != "" {
		t.Fatalf("the badge still waits after its ack: %q", techPending(s))
	}
}

func TestDismissingTechCalloutPassesOutsideClicksThrough(t *testing.T) {
	s := newPlayScene(newGame())
	s.techCallout = techArtilleryID
	box := techCalloutBounds(s)
	inside := box.Center()
	if !s.dismissTechCallout(inside.X, inside.Y) {
		t.Fatal("a click on the callout was not consumed")
	}
	if s.techCallout != "" {
		t.Fatal("a click on the callout did not close it")
	}

	s.techCallout = techArtilleryID
	outsideX := box.X + box.Width + 1
	if s.dismissTechCallout(outsideX, inside.Y) {
		t.Fatal("a click outside the callout was consumed")
	}
	if s.techCallout != "" {
		t.Fatal("a click outside the callout did not close it")
	}
}

func TestEveryDropSquaresWhatItBrings(t *testing.T) {
	want := map[string][]techItem{
		techInfraID: {
			{name: "Oil silo", kind: BuildingSilo},
			{name: "Mineral warehouse", kind: BuildingWarehouse},
			{name: "Robot charger", kind: BuildingCharger},
		},
		techGuardID: {
			{name: "Guard post", kind: BuildingGuard},
		},
		techFrontierID: {
			{name: "Shadow protector", kind: BuildingProtector},
			{name: "Oil pump", kind: BuildingPump},
			{name: "Pipes", pipes: true},
		},
		techIndustryID: {
			{name: "Robot factory", kind: BuildingFactory},
		},
		techMobileID: {
			{name: "War factory", kind: BuildingWarFactory},
		},
		techArtilleryID: {
			{name: "Artillery", kind: BuildingArtillery},
		},
	}
	if len(want) != len(techLadder) {
		t.Fatalf("%d drops are squared about, the ladder has %d",
			len(want), len(techLadder))
	}
	for i := range techLadder {
		id := techLadder[i].id
		got := techBrings(id)
		if !reflect.DeepEqual(got, want[id]) {
			t.Errorf("drop %q brings %v, want %v", id, got, want[id])
		}
	}
	if items := techBrings("no such drop"); items != nil {
		t.Errorf("an unknown drop brings %v, want nothing", items)
	}
}

func TestTechCalloutHitboxCoversItsSquares(t *testing.T) {
	s := newPlayScene(newGame())
	s.techCallout = techInfraID
	box := techCalloutBounds(s)
	want := float32(2*techCalloutPad + techCalloutHead +
		techCalloutMaxBodyRows*techCalloutRow +
		techSquareGap + techSquareH)
	if box.Height != want {
		t.Fatalf("the hitbox is %v tall, want %v", box.Height, want)
	}
	bottomX, bottomY := box.X+4, box.Y+box.Height-1
	if !s.dismissTechCallout(bottomX, bottomY) {
		t.Fatal("a click on the squares' last row was not consumed")
	}
	if s.techCallout != "" {
		t.Fatal("a click on the squares did not close the callout")
	}
	s.techCallout = techInfraID
	if s.dismissTechCallout(bottomX, bottomY+1) {
		t.Fatal("a click under the callout was consumed")
	}
}

func TestASaveFromBeforeTheSchematicsOpensWhatItEarned(t *testing.T) {
	s := newGame()
	s.Deliveries = 3
	s.Ticks = legacyTechIndustryTicks + 60
	s.Raids.Visits = 2
	s.Tech = nil
	Apply(s, Tick{})
	for _, id := range []string{
		techIndustryID, techInfraID, techFrontierID, techMobileID,
	} {
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

func TestAnExistingInfrastructureUnlockIsNotRevoked(t *testing.T) {
	for _, opened := range []bool{false, true} {
		s := newGame()
		s.Tech = map[string]bool{techInfraID: opened}
		Apply(s, Tick{})
		if !kindUnlocked(s, BuildingCharger) {
			t.Errorf(
				"opened=%v: an existing infrastructure unlock was revoked",
				opened,
			)
		}
		if s.Tech[techInfraID] != opened {
			t.Errorf("opened=%v: save changed infrastructure to %v", opened,
				s.Tech[techInfraID])
		}
		if !opened && techPending(s) != techInfraID {
			t.Errorf("the unacknowledged infrastructure badge became %q",
				techPending(s))
		}
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
	Apply(s, AckTech{ID: techIndustryID})
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

// TestWriteTechShotState writes the states the schematics callout's
// shots start from: NIEBLA_TECH_SHOT_STATE with the infrastructure
// waiting (three blueprints) and NIEBLA_TECH_PIPES_SHOT_STATE with the
// frontier kit (the protector, the pump and the pipes). Skipped
// otherwise, the way tests write nothing. Each run prints where the
// badge stands on the screen for the current layout.
func TestWriteTechShotState(t *testing.T) {
	infra := os.Getenv("NIEBLA_TECH_SHOT_STATE")
	frontier := os.Getenv("NIEBLA_TECH_PIPES_SHOT_STATE")
	if infra == "" && frontier == "" {
		t.Skip("set NIEBLA_TECH_SHOT_STATE / NIEBLA_TECH_PIPES_SHOT_STATE" +
			" to write the schematics callout's shot states")
	}
	write := func(path string, tech map[string]bool) {
		s := newGame()
		noRivals(s)
		s.Deliveries = 1
		s.Tech = tech
		x, y := techBadgeAt(newPlayScene(s))
		t.Logf("schematics badge click: %0.0f,%0.0f", x, y)
		data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
		if err != nil {
			t.Fatalf("the state doesn't marshal: %v", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	if infra != "" {
		write(infra, map[string]bool{techInfraID: false})
	}
	if frontier != "" {
		write(frontier, map[string]bool{
			techInfraID: true, techFrontierID: false,
		})
	}
}
