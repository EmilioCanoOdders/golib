package main

import (
	"math"
	"testing"
)

// settledCityNexus builds a city for shot tests and returns its Nexus.
func settledCityNexus(t *testing.T, s *State) Enemy {
	t.Helper()
	cityID := s.foundCity(3300, 3000, 0.4)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	s.Cities[cityID] = city
	partyID := s.NextID
	s.NextID++
	party := Party{ID: partyID, Stage: StageSettled, City: cityID}
	s.Parties[partyID] = party
	city = s.Cities[cityID]
	var core Enemy
	for _, id := range city.BuildingIDs {
		e := s.Enemies[id]
		if e.Kind != EnemyBase {
			continue
		}
		e.Party = partyID
		s.Enemies[id] = e
		core = e
	}
	if core.ID == 0 {
		t.Fatal("the city has no core")
	}
	return core
}

// cellNear returns the cell a spot of the region falls on.
func cellNear(x, y float64) (col, row int) {
	return int(x / buildingCell), int(y / buildingCell)
}

// towardCore returns the spot a way from a point toward the core.
func towardCore(x, y, way float64) (float64, float64) {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	gap := math.Hypot(cx-x, cy-y)
	return x + (cx-x)/gap*way, y + (cy-y)/gap*way
}

func TestABulletFliesBeforeItHurts(t *testing.T) {
	s := newGame()
	seedStock(s)
	col, row := groundNearCore()
	raised(t, s, BuildingGuard, col, row)
	visitNow(s)
	if !tickUntil(s, 60*600, func() bool { return len(s.Shots) > 0 }) {
		t.Fatalf("the guard post never fired")
	}
	var shot Shot
	for _, sh := range s.Shots {
		shot = sh
	}
	scout := s.Enemies[shot.Enemy]
	if shot.Kind != ShotBullet || scout.Health != enemySpecOf(EnemyScout).health {
		t.Fatalf("the shot is %+v and the scout has %v health: a bullet hurts on arrival",
			shot, scout.Health)
	}
	gap := math.Hypot(scout.X-shot.X, scout.Y-shot.Y)
	runTicks(s, int(gap/(bulletSpeed/60))+2)
	if got := s.Enemies[scout.ID].Health; got != scout.Health-guardShotDamage {
		t.Errorf("after the bullet's flight the scout has %v health, want %v",
			got, scout.Health-guardShotDamage)
	}
}

func TestSettledCityDoesNotFireByItself(t *testing.T) {
	s := newGame()
	core := settledCityNexus(t, s)
	if core.Health != enemySpecOf(EnemyBase).health {
		t.Errorf("the city Nexus has %v health", core.Health)
	}
	city := s.Cities[s.Parties[core.Party].City]
	city.Stage = len(cityBuildOrder)
	city.BuildingIDs = []int64{core.ID}
	s.Cities[city.ID] = city
	x, y := towardCore(core.X, core.Y, 800)
	col, row := cellNear(x, y)
	silo := raised(t, s, BuildingSilo, col, row)
	runTicks(s, cityBuildTicks+60*60)
	if len(s.Shots) != 0 || s.Buildings[silo.ID].Damage != 0 {
		t.Fatalf("the city fired without producing a mobile artillery unit")
	}
}

func TestRobotsMendADamagedBuilding(t *testing.T) {
	s := newGame()
	noRivals(s)
	col, row := groundNearCore()
	silo := raised(t, s, BuildingSilo, col, row)
	s.hurtBuilding(silo.ID, 120)
	if !tickUntil(s, 60*60, func() bool { return s.Buildings[silo.ID].Damage == 0 }) {
		t.Fatalf("the robots left the silo at %v damage", s.Buildings[silo.ID].Damage)
	}
	if got := robotCaption(s, s.Robots[1]); got == "repairing" {
		t.Errorf("a robot still says %q with nothing to mend", got)
	}
}

func TestArtilleryShellsWhatTheColonySeesAndCityNexusFalls(t *testing.T) {
	s := newGame()
	s.Stock = Stock{Oil: 1000, Lilac: 2500}
	delete(s.Robots, 1)
	delete(s.Robots, 2)
	core := settledCityNexus(t, s)
	city := s.Cities[s.Parties[core.Party].City]
	for _, id := range city.BuildingIDs {
		if id != core.ID {
			delete(s.Enemies, id)
		}
	}
	city.BuildingIDs = []int64{core.ID}
	s.Cities[city.ID] = city
	x, y := towardCore(core.X, core.Y, 1000)
	col, row := cellNear(x, y)
	raised(t, s, BuildingArtillery, col, row)
	runTicks(s, 60*20)
	if len(s.Shots) != 0 || s.Stock.Lilac != 2500 {
		t.Fatalf("the piece fired at a Nexus nobody sees")
	}
	// A spotter within sight of the Nexus, and the shells fly.
	sx, sy := towardCore(core.X, core.Y, sightUnits*0.8)
	s.spawnRobot(RobotCore, sx, sy)
	spotter := s.NextID - 1
	keep := func() {
		r := s.Robots[spotter]
		r.X, r.Y = sx, sy
		s.Robots[spotter] = r
	}
	fired := false
	for i := 0; i < 60*60 && !fired; i++ {
		keep()
		runTicks(s, 1)
		fired = len(s.Shots) > 0
	}
	if !fired || s.Stock.Lilac != 2500-artilleryShellLilac {
		t.Fatalf("with a spotter the piece fired %v and the lilac is %v", fired, s.Stock.Lilac)
	}
	health := s.Enemies[core.ID].Health
	flight := int(1000/(shellSpeed/60)) + 5
	for i := 0; i < flight; i++ {
		keep()
		runTicks(s, 1)
	}
	if got := s.Enemies[core.ID].Health; got > health-artilleryShellDamage {
		t.Errorf("after a shell's flight the Nexus has %v health, had %v", got, health)
	}
	for i := 0; i < 60*400; i++ {
		if _, stands := s.Enemies[core.ID]; !stands {
			break
		}
		keep()
		runTicks(s, 1)
	}
	if _, stands := s.Enemies[core.ID]; stands {
		t.Fatalf("the artillery never brought the Nexus down")
	}
	found := false
	for _, r := range s.Reports {
		found = found || r.Kind == ReportBaseDown
	}
	if !found {
		t.Errorf("nobody reported the city's fall: %+v", s.Reports)
	}
	if !tickUntil(s, 60*600, func() bool { return !settled(s) && len(s.Enemies) == 0 }) {
		t.Errorf("the garrison never left: %d vehicles", len(s.Enemies))
	}
}

func TestAShellMissesWhoMovedOn(t *testing.T) {
	s := newGame()
	noRivals(s)
	s.Enemies[900] = Enemy{ID: 900, Kind: EnemyRaider, X: 3000, Y: 3000, Health: 100}
	s.fire(Shot{Kind: ShotShell, FromX: 2500, FromY: 2500, ToX: 3000, ToY: 3000, Damage: 60})
	e := s.Enemies[900]
	e.X += shellBlastUnits * 2
	s.Enemies[900] = e
	s.Parties[901] = Party{ID: 901, Stage: StageCamp, Wait: 1 << 40}
	e.Party = 901
	s.Enemies[900] = e
	tickUntil(s, 60*60, func() bool { return len(s.Shots) == 0 })
	if got := s.Enemies[900].Health; got != 100 {
		t.Errorf("a shell hurt a vehicle two blasts away: %v health", got)
	}
}

func TestARaiderShootsAGuardPostAndThePostCanFall(t *testing.T) {
	s := newGame()
	seedStock(s)
	noRivals(s)
	col, row := groundNearCore()
	post := raised(t, s, BuildingGuard, col, row)
	// A raider parked inside its own reach, inside the bubble, with
	// nobody of the colony to shoot at but the post.
	px, py := cellCenterUnits(col, row)
	s.Enemies[900] = Enemy{
		ID: 900, Kind: EnemyRaider, X: px + 80, Y: py, Health: 100,
	}
	var shot Shot
	if !tickUntil(s, 60*60, func() bool {
		for _, sh := range s.Shots {
			if sh.Building == post.ID {
				shot = sh
				return true
			}
		}
		return false
	}) {
		t.Fatalf("the raider never shot the post")
	}
	if shot.Kind != ShotBullet || !shot.Rival || shot.Robot != 0 {
		t.Fatalf("the shot at the post is %+v", shot)
	}
	// Leave the post two hits short: the next two bring it down, and a
	// building's wreck is a pile of half its cost.
	b := s.Buildings[post.ID]
	b.Damage = buildingHealth(BuildingGuard) -
		2*enemySpecOf(EnemyRaider).damage
	s.Buildings[post.ID] = b
	if !tickUntil(s, 60*60, func() bool {
		_, stands := s.Buildings[post.ID]
		return !stands
	}) {
		t.Fatalf("the post never fell under the raider's fire")
	}
	if lastReport(s).Kind != ReportRazed {
		t.Errorf("the report is %q, want the post razed", lastReport(s).Kind)
	}
	if p, littered := pileAt(s, col, row); !littered ||
		p.Lilac != guardCostLilac*wreckRefund {
		t.Errorf("the wreck left %+v, want %v kg", p, guardCostLilac*wreckRefund)
	}
}
