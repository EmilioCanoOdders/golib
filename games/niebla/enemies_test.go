package main

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"golib"
)

func TestWorldSpritesHaveEveryFacing(t *testing.T) {
	for name, model := range map[string]worldSprite{
		"core":      coreWorkerModel,
		"carrier":   carrierModel,
		"trooper":   trooperModel,
		"mechanic":  mechanicModel,
		"scout":     rivalScoutModel,
		"crawler":   rivalCrawlerModel,
		"raider":    rivalRaiderModel,
		"artillery": rivalArtilleryModel,
	} {
		if got := model.sprite.Frames(); got != 8 {
			t.Errorf("%s sprite has %d facings, want 8", name, got)
		}
	}
}

func TestWorldSpritesMoveInScreenPixelsAtCloseZoom(t *testing.T) {
	camera := golib.NewCamera(screenWidth, screenHeight)
	camera.Zoom = 32
	camera.Target = golib.Vector2{X: 100, Y: 200}
	camera.Snap()
	before := modelScreenPosition(camera, golib.Vector2{X: 100.2, Y: 200})
	after := modelScreenPosition(camera, golib.Vector2{X: 100.4, Y: 200})
	if moved := after.X - before.X; math.Abs(float64(moved-6.4)) > 0.01 {
		t.Errorf("sprite moved %.2f screen pixels, want 6.4", moved)
	}
}

// noRivals keeps the rivals away for good, for the tests that are about
// something else.
func noRivals(s *State) {
	s.Raids.NextAt = math.MaxInt64
}

// visitNow brings the next visit in on the next tick.
func visitNow(s *State) {
	Apply(s, DevNextVisit{})
	runTicks(s, 1)
}

func lastReport(s *State) Report {
	if len(s.Reports) == 0 {
		return Report{}
	}
	return s.Reports[len(s.Reports)-1]
}

func TestTheFirstScoutSiphonsLeavesItsMarkAndGoes(t *testing.T) {
	s := newGame()
	runTicks(s, raidFirstScoutTicks-1)
	if len(s.Enemies) != 0 {
		t.Fatalf("%d vehicles before the scout's tick", len(s.Enemies))
	}
	runTicks(s, 1)
	if len(s.Enemies) != 1 || len(s.Parties) != 1 {
		t.Fatalf("the first visit is %d vehicles in %d parties, want a lone scout",
			len(s.Enemies), len(s.Parties))
	}
	oil := oilTotal(s)
	if !tickUntil(s, 60*600, func() bool { return len(s.Marks) == 1 }) {
		t.Fatalf("the scout never left its mark")
	}
	cap := enemySpecOf(EnemyScout).oilCap
	if got := oil - oilTotal(s); math.Abs(got-cap) > 0.001 {
		t.Errorf("the scout took %v L, want its tank's %v", got, cap)
	}
	if r := lastReport(s); r.Kind != ReportScout || math.Abs(r.Oil-cap) > 0.001 {
		t.Errorf("the last report is %+v, want the scout's with its %v L", r, cap)
	}
	if !tickUntil(s, 60*600, func() bool { return len(s.Parties) == 0 }) {
		t.Fatalf("the scout never left")
	}
	if len(s.Enemies) != 0 || s.Raids.Visits != 1 {
		t.Errorf("%d vehicles left and %d visits counted, want 0 and 1",
			len(s.Enemies), s.Raids.Visits)
	}
	if got, want := s.Raids.NextAt-s.Ticks, calmTicks(1); got != want {
		t.Errorf("the next visit comes in %d ticks, want %d", got, want)
	}
}

func TestARaidCampsGetsReadyStealsAndLeaves(t *testing.T) {
	s := newGame()
	seedStock(s)
	s.Raids.Visits = 1
	visitNow(s)
	if got, want := len(s.Enemies), 1+raidFirstRaiders; got != want {
		t.Fatalf("the first raid brings %d vehicles, want %d", got, want)
	}
	var party Party
	for _, p := range s.Parties {
		party = p
	}
	if party.Stage != StageApproach {
		t.Fatalf("the raid comes in at stage %q, want approach", party.Stage)
	}
	staged := func(stage PartyStage) func() bool {
		return func() bool { return s.Parties[party.ID].Stage == stage }
	}
	if !tickUntil(s, 60*300, staged(StageCamp)) {
		t.Fatalf("the raid never camped")
	}
	if lastReport(s).Kind != ReportCamp {
		t.Errorf("camping reported %q", lastReport(s).Kind)
	}
	lead := partyMembers(s, party.ID)[0]
	cx, cy := tileCenterUnits(coreCol, coreRow)
	gap := math.Hypot(lead.X-cx, lead.Y-cy) / unitsPerTile
	if math.Abs(gap-campRadiusTiles) > 0.01 {
		t.Errorf("the camp is %v tiles out, want %v", gap, campRadiusTiles)
	}
	oil := oilTotal(s)
	runTicks(s, int(prepareTicks(1))-1)
	if !staged(StageCamp)() || oilTotal(s) != oil {
		t.Errorf("the raid moved before its time")
	}
	runTicks(s, 1)
	if !staged(StageRaid)() || lastReport(s).Kind != ReportRaid {
		t.Errorf("after getting ready the stage is %q and the report %q",
			s.Parties[party.ID].Stage, lastReport(s).Kind)
	}
	if !tickUntil(s, 60*600, func() bool { return len(s.Parties) == 0 }) {
		t.Fatalf("the raid never left")
	}
	want := raidFirstRaiders * enemySpecOf(EnemyRaider).oilCap
	if got := oil - oilTotal(s); math.Abs(got-want) > 0.001 {
		t.Errorf("the raid took %v L, want every raider full: %v", got, want)
	}
	if r := lastReport(s); r.Kind != ReportLeft || math.Abs(r.Oil-want) > 0.001 {
		t.Errorf("the last report is %+v, want them gone with %v L", r, want)
	}
	if raidersOf(2) != raidFirstRaiders+1 || prepareTicks(2) >= prepareTicks(1) {
		t.Errorf("the second raid is no bigger or no quicker than the first")
	}
}

func TestAGuardPostShootsForOilAndTheWrecksDropWhatTheyStole(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundNearCore()
	raised(t, s, BuildingGuard, col, row)
	visitNow(s)
	oil := oilTotal(s)
	if !tickUntil(s, 60*600, func() bool { return len(s.Parties) == 0 }) {
		t.Fatalf("the scout's visit never ended")
	}
	if lastReport(s).Kind != ReportDestroyed || len(s.Marks) != 0 {
		t.Errorf("the last report is %q with %d marks, want the scout shot down first",
			lastReport(s).Kind, len(s.Marks))
	}
	spec := enemySpecOf(EnemyScout)
	shots := math.Ceil(spec.health / guardShotDamage)
	if got, want := oil-oilTotal(s), shots*guardShotOil; math.Abs(got-want) > 0.001 {
		t.Errorf("the colony burned %v L, want %v shots' worth: %v", got, shots, want)
	}
	if len(s.Piles) != 1 {
		t.Fatalf("%d piles after the wreck, want 1", len(s.Piles))
	}
	for _, p := range s.Piles {
		if math.Abs(p.Oil-spec.lootOil) > 0.001 || math.Abs(p.Lilac-spec.lootLilac) > 0.001 {
			t.Errorf("the wreck dropped %v L and %v kg, want %v and %v",
				p.Oil, p.Lilac, spec.lootOil, spec.lootLilac)
		}
	}
	// A dry colony doesn't shoot.
	s.Stock.Oil = 0
	s.Raids.Visits = 0
	visitNow(s)
	runTicks(s, 60*60)
	for _, e := range s.Enemies {
		if e.Health != spec.health {
			t.Errorf("a guard with no oil hurt the scout: %v health", e.Health)
		}
	}
}

func TestRaidersWithoutTheirCrawlerAreDigested(t *testing.T) {
	s := newGame()
	s.Raids.Visits = 1
	visitNow(s)
	var crawler int64
	for _, e := range s.Enemies {
		if e.Kind == EnemyCrawler {
			crawler = e.ID
		}
	}
	runTicks(s, 60*30) // far enough in that running back out takes longer than the fog
	for _, e := range s.Enemies {
		if e.Fogged != 0 {
			t.Fatalf("a %s under its crawler's bubble wears the fog", e.Kind)
		}
	}
	s.killEnemy(crawler)
	runTicks(s, enemyFogTicks-1)
	if len(s.Enemies) != raidFirstRaiders {
		t.Fatalf("%d raiders stand before the fog's time, want %d",
			len(s.Enemies), raidFirstRaiders)
	}
	runTicks(s, 2)
	if len(s.Enemies) != 0 || len(s.Parties) != 0 {
		t.Errorf("%d vehicles and %d parties after the fog's time, want none",
			len(s.Enemies), len(s.Parties))
	}
	if lastReport(s).Kind != ReportDestroyed {
		t.Errorf("the last report is %q, want destroyed", lastReport(s).Kind)
	}
	if len(s.Piles) == 0 {
		t.Errorf("the wrecks left nothing on the ground")
	}
}

func TestRivalsReplayAndSurviveASave(t *testing.T) {
	play := func() *State {
		s := newGame()
		s.Raids.Visits = 2
		visitNow(s)
		runTicks(s, 60*200)
		return s
	}
	s := play()
	if !reflect.DeepEqual(s, play()) {
		t.Errorf("two runs of the same raid ended in different states")
	}
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
	// A save from before the rivals has no tables and no clock for them.
	old := newGame()
	old.Enemies, old.Parties, old.Marks, old.Raids = nil, nil, nil, Raids{}
	runTicks(old, raidFirstScoutTicks+60*600)
	if old.Raids.Visits != 1 || len(old.Marks) != 1 {
		t.Errorf("an old save saw %d visits and %d marks, want the scout's",
			old.Raids.Visits, len(old.Marks))
	}
}
