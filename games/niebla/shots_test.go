package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestShellLaunchPointStartsAtTheMuzzle(t *testing.T) {
	fromX, fromY := 80.0, 120.0
	toX, toY := 380.0, 520.0
	muzzleX, muzzleY := shellLaunchPoint(fromX, fromY, toX, toY)
	got := math.Hypot(muzzleX-fromX, muzzleY-fromY)
	if math.Abs(got-shellMuzzleOffsetUnits) > 0.0001 {
		t.Fatalf("the shell starts %v u ahead of the gun, want %v",
			got, shellMuzzleOffsetUnits)
	}
	remaining := math.Hypot(toX-muzzleX, toY-muzzleY)
	original := math.Hypot(toX-fromX, toY-fromY)
	if remaining >= original {
		t.Fatal("the muzzle point did not move toward the target")
	}
	closeX, closeY := shellLaunchPoint(fromX, fromY, fromX+10, fromY)
	got = math.Hypot(closeX-fromX, closeY-fromY)
	if math.Abs(got-5) > 0.0001 {
		t.Fatalf("a close shot starts %v u from the gun, want 5", got)
	}
}

func TestShellTrailGrowsWithItsFlight(t *testing.T) {
	shot := Shot{Kind: ShotShell, ToX: 500}
	if got := shellTrailSteps(shot); got != 0 {
		t.Fatalf("a shell at the muzzle has %d trail marks", got)
	}
	shot.X = shellTrailStepUnits - 0.1
	if got := shellTrailSteps(shot); got != 0 {
		t.Fatalf("a shell before its first trail step has %d marks", got)
	}
	shot.X = shellTrailStepUnits * 2
	if got, want := shellTrailSteps(shot), 2; got != want {
		t.Fatalf("a shell after two steps has %d trail marks, want %d", got, want)
	}
	shot.X = shellTrailLengthUnits * 2
	if got, want := shellTrailSteps(shot), 6; got != want {
		t.Fatalf("a long-flight shell has %d trail marks, want %d", got, want)
	}
}

func TestShellLeavesSubtleSmokeAlongItsFlight(t *testing.T) {
	f := newFxField()
	shot := Shot{
		ID: 1, Kind: ShotShell,
		FromX: 10, FromY: 20, ToX: 1010, ToY: 20,
	}
	shot.X, shot.Y = shot.FromX, shot.FromY
	s := &State{Shots: map[int64]Shot{shot.ID: shot}}
	f.update(s, 1.0/60)
	shot.X = shot.FromX + shellSmokeStepUnits*2 + shellSmokeStepUnits/2
	shot.Y = shot.FromY
	s.Shots[shot.ID] = shot
	f.update(s, 1.0/60)

	trailPuffs := 0
	for _, particle := range f.sparks {
		if particle.smoke && particle.opacity == shellSmokeOpacity {
			trailPuffs++
		}
	}
	if trailPuffs != 2 {
		t.Fatalf("the shell left %d trail puffs, want 2", trailPuffs)
	}
}

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

func TestOnlyAMechanicRepairsAndSpendsOil(t *testing.T) {
	s := newGame()
	noRivals(s)
	col, row := groundNearCore()
	silo := raised(t, s, BuildingSilo, col, row)
	s.hurtBuilding(silo.ID, 120)
	runTicks(s, 60*30)
	if got := s.Buildings[silo.ID].Damage; got != 120 {
		t.Fatalf("ordinary workers repaired the silo to %v damage", got)
	}
	if got := robotCaption(s, s.Robots[1]); got == "repairing" {
		t.Errorf("an ordinary worker still says %q", got)
	}

	home := raised(t, s, BuildingWarFactory, col+2, row)
	s.Stock = Stock{Oil: 1000, Lilac: 2500}
	Apply(s, QueueMechanic{Building: home.ID})
	work := s.Buildings[home.ID]
	if work.Work != mechanicBuildTicks || work.WorkKind != RobotRepair {
		t.Fatalf("the war factory queued %+v, want a mechanic", work)
	}
	if s.Stock.Lilac != 2500-mechanicCostLilac ||
		s.Stock.Oil != 1000-mechanicCostOil {
		t.Fatalf("the mechanic cost left %+v in the stores", s.Stock)
	}
	runTicks(s, mechanicBuildTicks)
	mechanic, found := mechanicForFactory(s, home.ID)
	if !found || mechanic.Health != mechanicHealth || mechanic.Tank != robotTankLiters {
		t.Fatalf("the war factory produced mechanic %+v, found %v", mechanic, found)
	}
	if !tickUntil(s, 60*60, func() bool {
		return s.Buildings[silo.ID].Damage == 0
	}) {
		t.Fatalf("the mechanic left the silo at %v damage",
			s.Buildings[silo.ID].Damage)
	}
	mechanic = s.Robots[mechanic.ID]
	wantTank := robotTankLiters - 120*repairOilPerPoint
	if math.Abs(mechanic.Tank-wantTank) > 0.001 {
		t.Errorf("the mechanic has %v L left, want %v after repairs",
			mechanic.Tank, wantTank)
	}
}

func TestOneMechanicCannotOutrepairContinuousArtillery(t *testing.T) {
	s := newGame()
	noRivals(s)
	col, row := groundNearCore()
	silo := raised(t, s, BuildingSilo, col, row)
	cx, cy := cellCenterUnits(col, row)
	id := s.spawnRobot(RobotRepair, cx, cy)
	r := s.Robots[id]
	r.Tank = robotTankLiters
	s.Robots[id] = r
	s.hurtBuilding(silo.ID, cityArtilleryDamage)
	for i := 0; i < 5; i++ {
		r = s.Robots[id]
		angle := float64(id) * goldenAngle
		r.X = cx + math.Cos(angle)*11
		r.Y = cy + math.Sin(angle)*11
		s.Robots[id] = r
		runTicks(s, cityArtilleryReload)
		s.hurtBuilding(silo.ID, cityArtilleryDamage)
	}
	if got := s.Buildings[silo.ID].Damage; got <= cityArtilleryDamage {
		t.Errorf("the mechanic held damage to %v under continuous fire", got)
	}
}

func TestRivalShotsDamageMechanicsButNotWorkers(t *testing.T) {
	s := newGame()
	noRivals(s)
	worker := s.Robots[1]
	col, row := groundNearCore()
	home := raised(t, s, BuildingWarFactory, col, row)
	cx, cy := cellCenterUnits(col, row)
	id := s.spawnRobot(RobotRepair, cx, cy)
	mechanic := s.Robots[id]
	mechanic.Factory = home.ID
	mechanic.Tank = 80
	s.Robots[id] = mechanic

	s.fire(Shot{
		Kind: ShotBullet, FromX: cx - 1, FromY: cy,
		ToX: cx, ToY: cy, Robot: id, Damage: 15, Rival: true,
	})
	runTicks(s, 2)
	if got := s.Robots[id].Health; got != mechanicHealth-15 {
		t.Fatalf("the rival bullet left the mechanic at %v health", got)
	}
	if got := s.Robots[worker.ID].Health; got != 0 {
		t.Errorf("a worker gained health from a shot: %v", got)
	}

	s.fire(Shot{
		Kind: ShotShell, FromX: cx, FromY: cy,
		ToX: cx, ToY: cy, Damage: 30, Rival: true,
	})
	runTicks(s, 1)
	if got := s.Robots[id].Health; got != mechanicHealth-45 {
		t.Fatalf("the rival shell left the mechanic at %v health", got)
	}
	wreckCol, wreckRow := robotCell(s.Robots[id])
	s.hurtColonyUnit(id, mechanicHealth-45)
	if _, alive := s.Robots[id]; alive {
		t.Fatal("the mechanic survived lethal damage")
	}
	pile, found := pileAt(s, wreckCol, wreckRow)
	if !found {
		t.Fatal("the fallen mechanic left no wreck")
	}
	if want := mechanicCostLilac * unitWreckRefund; pile.Lilac != want {
		t.Errorf("the mechanic wreck holds %v kg, want %v", pile.Lilac, want)
	}
	if want := (mechanicCostOil + 80) * unitWreckRefund; pile.Oil != want {
		t.Errorf("the mechanic wreck holds %v L, want %v", pile.Oil, want)
	}
}

func TestOldWarFactoryWorkStillFinishesAsATrooper(t *testing.T) {
	s := newGame()
	col, row := groundNearCore()
	home := raised(t, s, BuildingWarFactory, col, row)
	home.Work = 5
	s.Buildings[home.ID] = home
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var save map[string]json.RawMessage
	if err := json.Unmarshal(data, &save); err != nil {
		t.Fatal(err)
	}
	var buildings map[string]json.RawMessage
	if err := json.Unmarshal(save["Buildings"], &buildings); err != nil {
		t.Fatal(err)
	}
	for id, encoded := range buildings {
		var building map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &building); err != nil {
			t.Fatal(err)
		}
		delete(building, "WorkKind")
		buildings[id], err = json.Marshal(building)
		if err != nil {
			t.Fatal(err)
		}
	}
	save["Buildings"], err = json.Marshal(buildings)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(save)
	if err != nil {
		t.Fatal(err)
	}
	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	runTicks(&loaded, 5)
	if got := len(squadMembers(&loaded, home.ID)); got != 1 {
		t.Fatalf("an old save's factory produced %d troopers, want one", got)
	}
	if got := mechanicCount(&loaded, home.ID); got != 0 {
		t.Errorf("an old save's factory produced %d mechanics", got)
	}
}

func mechanicForFactory(s *State, factory int64) (Robot, bool) {
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		if r.Kind == RobotRepair && r.Factory == factory {
			return r, true
		}
	}
	return Robot{}, false
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
