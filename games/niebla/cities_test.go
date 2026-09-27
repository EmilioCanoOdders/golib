package main

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

func finishedCityForTest(s *State) int64 {
	cityID := s.foundCity(3500, 3200, 0.4)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	s.Cities[cityID] = city
	return cityID
}

func assertCityBuildsPylonBeforeNexus(
	t *testing.T,
	s *State,
	city *City,
) {
	t.Helper()
	if city.Stage != 0 || cityHasRepulsor(s, *city) ||
		cityHasBuilding(s, *city, EnemyBase) {
		t.Fatalf("city did not start with the pylon: %+v", *city)
	}
	for range cityBuildTicks - 1 {
		stepCity(s, city)
	}
	if city.Stage != 0 || city.Work != 1 ||
		cityHasRepulsor(s, *city) || cityHasBuilding(s, *city, EnemyBase) {
		t.Fatalf("the pylon started before its full build time: %+v", *city)
	}
	stepCity(s, city)
	if city.Stage != 1 || city.Work != cityBuildTicks ||
		!cityHasRepulsor(s, *city) || cityHasBuilding(s, *city, EnemyBase) {
		t.Fatalf("the Nexus appeared before the pylon was built: %+v", *city)
	}
	for range cityBuildTicks - 1 {
		stepCity(s, city)
	}
	if city.Stage != 1 || city.Work != 1 ||
		!cityHasRepulsor(s, *city) || cityHasBuilding(s, *city, EnemyBase) {
		t.Fatalf("the Nexus started before its full build time: %+v", *city)
	}
	stepCity(s, city)
	if city.Stage != 2 || !cityHasRepulsor(s, *city) ||
		!cityHasBuilding(s, *city, EnemyBase) {
		t.Fatalf("the Nexus did not follow the completed pylon: %+v", *city)
	}
}

func TestIntroVisitsShareTheScoutsBearing(t *testing.T) {
	s := newGame()
	visitNow(s)
	angle := s.Raids.FirstBearing
	if !s.Raids.BearingKnown {
		t.Fatal("the scout's entry bearing was not saved")
	}
	s.Parties = map[int64]Party{}
	s.Enemies = map[int64]Enemy{}
	s.Raids.Visits = 1
	visitNow(s)
	party := s.Parties[sortedPartyIDs(s)[0]]
	cx, cy := tileCenterUnits(coreCol, coreRow)
	most := float64(regionCols*unitsPerTile) - 1
	wantX := clamp64(cx+math.Cos(angle)*entryRadiusTiles*unitsPerTile, 1, most)
	wantY := clamp64(cy+math.Sin(angle)*entryRadiusTiles*unitsPerTile, 1, most)
	if math.Abs(party.EntryX-wantX) > 0.001 ||
		math.Abs(party.EntryY-wantY) > 0.001 {
		t.Errorf("the intro battalion entered at %.1f, %.1f, want scout bearing %.1f, %.1f",
			party.EntryX, party.EntryY, wantX, wantY)
	}
}

func TestAResidentCrawlerArrivesBeforeTheCityBuilds(t *testing.T) {
	s := newGame()
	s.Raids.Visits = 2
	s.Raids.NextAt = s.Ticks + 1
	runTicks(s, 1)
	partyID := sortedPartyIDs(s)[0]
	party := s.Parties[partyID]
	if !party.CityArrives || party.Stage != StageApproach || len(s.Cities) != 0 {
		t.Fatalf("the city arrived as %+v with %d cities", party, len(s.Cities))
	}
	if !tickUntil(s, 60*300, func() bool { return len(s.Cities) == 1 }) {
		t.Fatal("the crawler never established its city")
	}
	cityID := sortedCityIDs(s)[0]
	city := s.Cities[cityID]
	if city.AnnounceUntil != s.Ticks+cityAnnouncementTicks {
		t.Fatalf("city announcement ends at %d, want %d ticks ahead",
			city.AnnounceUntil, cityAnnouncementTicks)
	}
	cx, cy := tileCenterUnits(coreCol, coreRow)
	distance := math.Hypot(city.X-cx, city.Y-cy)
	if _, partyStillMoving := s.Parties[partyID]; partyStillMoving ||
		city.Stage != 0 || len(city.BuildingIDs) != 1 ||
		city.NexusID == 0 ||
		s.Enemies[city.NexusID].ID != 0 ||
		s.Enemies[city.BuildingIDs[0]].Kind != EnemyCityCrawler ||
		enemySpecOf(EnemyCityCrawler).bubble != 0 ||
		cityHasBuilding(s, city, EnemyBase) ||
		distance <= artilleryRangeUnits {
		t.Fatalf("crawler settled as city %+v at %.0f m", city, distance)
	}
	Apply(s, Tick{})
	if len(s.Cities) != 1 {
		t.Fatal("the city rig left after settling")
	}
	city = s.Cities[cityID]
	s.finishCityBuilding(&city)
	if city.Stage != 1 || !cityHasRepulsor(s, city) ||
		cityHasBuilding(s, city, EnemyBase) || s.Enemies[city.NexusID].ID != 0 {
		t.Fatalf("the pylon was not the first build: %+v", city)
	}
	s.finishCityBuilding(&city)
	if city.Stage != 2 || !cityHasBuilding(s, city, EnemyBase) ||
		s.Enemies[city.NexusID].Kind != EnemyBase ||
		enemySpecOf(EnemyBase).bubble != 0 {
		t.Fatalf("the nexus was not built after the pylon: %+v", city)
	}
}

func TestRivalCityBuildsPylonBeforeNexusInSeparateSteps(t *testing.T) {
	s := newGame()
	cityID := s.foundCity(3500, 3200, 0.4)
	city := s.Cities[cityID]
	if cityBuildTicks != 45*60 || city.Work != 45*60 {
		t.Fatalf("city building work is %d ticks, want 45 seconds",
			city.Work)
	}
	assertCityBuildsPylonBeforeNexus(t, s, &city)
	s.Cities[cityID] = city
}

func TestCityRebuildsMissingBuildingsBeforeResumingProduction(t *testing.T) {
	s := newGame()
	noRivals(s)
	cityID := finishedCityForTest(s)
	city := s.Cities[cityID]
	city.OilDeposit = cityOilReserve
	city.LilacDeposit = cityLilacReserve
	city.NextSortie = s.Ticks
	s.Cities[cityID] = city

	nexusID, factoryID := int64(0), int64(0)
	for _, id := range city.BuildingIDs {
		switch s.Enemies[id].Kind {
		case EnemyBase:
			nexusID = id
		case EnemyCityFactory:
			factoryID = id
		}
	}
	if nexusID == 0 || factoryID == 0 {
		t.Fatal("the completed city is missing its Nexus or factory")
	}

	s.killEnemy(nexusID)
	s.killEnemy(factoryID)
	city = s.Cities[cityID]
	stage, building := cityNextBuildingStage(s, city)
	if city.Ruined || !building || stage != 1 ||
		city.Work != cityBuildTicks || city.Stage != len(cityBuildOrder) {
		t.Fatalf("city did not prioritize its missing Nexus: %+v", city)
	}

	city.Work = 1
	s.Cities[cityID] = city
	stepCity(s, &city)
	s.Cities[cityID] = city
	stage, building = cityNextBuildingStage(s, city)
	if !building || stage != 4 || city.Work != cityBuildTicks ||
		city.Stage != len(cityBuildOrder) {
		t.Fatalf("city did not rebuild the Nexus before its factory: %+v", city)
	}
	if city.NexusID == nexusID ||
		!cityHasBuilding(s, city, EnemyBase) ||
		cityHasBuilding(s, city, EnemyCityFactory) {
		t.Fatal("the Nexus was not replaced on its original build-order step")
	}

	stepCity(s, &city)
	if city.Oil != 0 || city.Lilac != 0 || len(s.Parties) != 0 {
		t.Fatalf("the city produced while rebuilding: %+v", city)
	}
}

func TestCompletedCityShowsItsRebuildInTheHud(t *testing.T) {
	s := newGame()
	noRivals(s)
	cityID := finishedCityForTest(s)
	city := s.Cities[cityID]
	city.AnnounceUntil = s.Ticks
	var factoryID int64
	for _, id := range city.BuildingIDs {
		if s.Enemies[id].Kind == EnemyCityFactory {
			factoryID = id
		}
	}
	s.killEnemy(factoryID)
	city = s.Cities[cityID]
	want := "rival city " + compassWord(city.X, city.Y) +
		", rebuilding " + cityBuildingName(CityFactory)
	if got := threatWords(s); got != want {
		t.Fatalf("rebuilding city HUD says %q, want %q", got, want)
	}
}

func TestRazedCityRefoundsWithACrawlerAtANewSite(t *testing.T) {
	s := newGame()
	noRivals(s)
	cityID := finishedCityForTest(s)
	origin := s.Cities[cityID]
	structureIDs := make([]int64, 0, len(cityBuildOrder))
	for _, id := range origin.BuildingIDs {
		if cityStageForEnemy(s.Enemies[id].Kind) >= 0 {
			structureIDs = append(structureIDs, id)
		}
	}
	if len(structureIDs) != len(cityBuildOrder) {
		t.Fatalf("the city has %d structures, want %d",
			len(structureIDs), len(cityBuildOrder))
	}
	partyID := s.NextID
	s.NextID++
	s.Parties[partyID] = Party{
		ID: partyID, City: cityID, Stage: StageRaid,
		EntryX: origin.X, EntryY: origin.Y,
		CampX: origin.X, CampY: origin.Y,
	}
	raiderID := s.NextID
	s.NextID++
	s.Enemies[raiderID] = Enemy{
		ID: raiderID, Kind: EnemyRaider, Party: partyID, City: cityID,
		X: origin.X, Y: origin.Y,
		Health: enemySpecOf(EnemyRaider).health,
	}

	for _, id := range structureIDs {
		s.killEnemy(id)
	}
	if got, want := len(s.BuildingDeaths), len(structureIDs)+1;
		got != want {
		t.Fatalf("the razed city made %d collapse events, want %d",
			got, want)
	}
	city := s.Cities[cityID]
	if !city.Ruined || city.RefoundAt != s.Ticks+cityRefoundDelayTicks ||
		len(city.BuildingIDs) != 0 {
		t.Fatalf("the city was not fully razed: %+v", city)
	}
	if report := lastReport(s); report.Kind != ReportBaseDown {
		t.Fatalf("the razed city reported %q, want base down", report.Kind)
	}
	if s.Parties[partyID].Stage != StageLeave {
		t.Fatal("the razed city's battalion did not withdraw")
	}

	s.Deaths = nil
	s.BuildingDeaths = nil
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, &restored) {
		t.Fatal("the pending refounding changed across a JSON round trip")
	}

	robots := len(s.Robots)
	runTicks(s, int(cityRefoundDelayTicks)-1)
	runTicks(&restored, int(cityRefoundDelayTicks)-1)
	if !reflect.DeepEqual(s, &restored) {
		t.Fatal("the refounding changed after loading its saved state")
	}
	if len(s.Parties) != 0 {
		t.Fatal("a replacement crawler arrived before the one-minute delay")
	}
	Apply(s, Tick{})
	Apply(&restored, Tick{})
	if !reflect.DeepEqual(s, &restored) {
		t.Fatal("the replacement crawler did not replay deterministically")
	}
	incomingPartyID := int64(0)
	for _, id := range sortedPartyIDs(s) {
		party := s.Parties[id]
		if party.CityArrives && party.City == cityID {
			incomingPartyID = id
			break
		}
	}
	if incomingPartyID == 0 {
		t.Fatal("the city did not send a replacement crawler")
	}
	party := s.Parties[incomingPartyID]
	if math.Hypot(party.CampX-origin.X, party.CampY-origin.Y) <
		cityMinSeparation {
		t.Fatalf("the new site is too close to the razed city: %+v", party)
	}
	if len(s.Robots) != robots {
		t.Fatal("refounding spawned a colony-style builder robot")
	}
	if movingParty(s) && len(s.Parties) != 1 {
		t.Fatalf("refounding created %d parties", len(s.Parties))
	}

	if !tickUntil(s, 60*300, func() bool {
		return !s.Cities[cityID].Ruined
	}) {
		t.Fatal("the replacement crawler never founded its city")
	}
	city = s.Cities[cityID]
	if city.X == origin.X && city.Y == origin.Y {
		t.Fatal("the refounded city reused its razed location")
	}
	if city.Stage != 0 || city.Work != cityBuildTicks ||
		len(city.BuildingIDs) != 1 ||
		s.Enemies[city.BuildingIDs[0]].Kind != EnemyCityCrawler {
		t.Fatalf("the refounded city did not start its normal build: %+v", city)
	}
	assertCityBuildsPylonBeforeNexus(t, s, &city)
	s.Cities[cityID] = city
}

func TestVersionNineCityWithMissingBuildingGetsFreshBuildWork(t *testing.T) {
	s := newGame()
	cityID := finishedCityForTest(s)
	city := s.Cities[cityID]
	var factoryID int64
	for _, id := range city.BuildingIDs {
		if s.Enemies[id].Kind == EnemyCityFactory {
			factoryID = id
		}
	}
	s.killEnemy(factoryID)
	city = s.Cities[cityID]
	city.Work = 0
	s.Cities[cityID] = city
	s.Version = 9

	s.migrateState()
	city = s.Cities[cityID]
	if s.Version != stateVersion || city.Ruined ||
		city.Work != cityBuildTicks {
		t.Fatalf("version 9 city migrated to %+v at version %d",
			city, s.Version)
	}
}

func TestCityAnnouncementLastsOneMinute(t *testing.T) {
	s := newGame()
	cityID := s.foundCity(3500, 3200, 0.4)
	city := s.Cities[cityID]
	if got := threatWords(s); got == "" {
		t.Fatal("a newly established city was not announced")
	}
	if report, ok := currentReport(s); !ok || report.Kind != ReportSettled {
		t.Fatalf("city founding report is %+v, visible %t", report, ok)
	}

	s.Ticks = city.AnnounceUntil - 1
	if got := threatWords(s); got == "" {
		t.Fatal("the city announcement ended before one minute")
	}
	if _, ok := currentReport(s); !ok {
		t.Fatal("the city founding report ended before one minute")
	}

	s.Ticks = city.AnnounceUntil
	if got := threatWords(s); got != "" {
		t.Errorf("city announcement remained after one minute: %q", got)
	}
	if report, ok := currentReport(s); ok {
		t.Errorf("city founding report remained after one minute: %+v", report)
	}
}

func TestLoadedSettledCityHasNoFreshAnnouncement(t *testing.T) {
	s := newGame()
	s.Cities[1] = City{ID: 1, X: 3500, Y: 3200, Stage: len(cityBuildOrder)}
	if got := threatWords(s); got != "" {
		t.Fatalf("a saved city without an active deadline was announced: %q", got)
	}
}

func TestOldSettledBaseMigratesToCityState(t *testing.T) {
	s := newGame()
	s.NextID = 10
	s.Parties[8] = Party{ID: 8, Stage: StageSettled}
	s.Enemies[9] = Enemy{
		ID: 9, Kind: EnemyBase, Party: 8,
		X: 2400, Y: 2500, Health: 900,
	}
	s.Cities = nil
	s.enterRegion()
	party := s.Parties[8]
	city, ok := s.Cities[party.City]
	if !ok || party.City == 0 || city.Stage != 2 || city.AnnounceUntil != 0 ||
		s.Raids.PressureCity != city.ID ||
		!cityHasBuilding(s, city, EnemyBase) ||
		!cityHasRepulsor(s, city) ||
		s.Enemies[9].Health != enemySpecOf(EnemyBase).health ||
		s.Enemies[9].Reload != 0 {
		t.Fatalf("old settlement migrated to party %+v and city %+v", party, city)
	}
}

func TestVersionThreeCitySaveMigratesItsPressureCityAndPartySize(t *testing.T) {
	s := newGame()
	cityID := s.foundCity(3500, 3200, 0.4)
	s.Version = 3
	partyID := s.NextID
	s.NextID++
	s.Parties[partyID] = Party{
		ID: partyID, City: cityID, Stage: StageUnload,
	}
	for _, kind := range []EnemyKind{EnemyRaider, EnemyArtillery} {
		id := s.NextID
		s.NextID++
		s.Enemies[id] = Enemy{
			ID: id, Kind: kind, Party: partyID, City: cityID,
			Health: enemySpecOf(kind).health,
		}
	}
	s.migrateState()
	party := s.Parties[partyID]
	if s.Version != stateVersion || s.Raids.PressureCity != cityID ||
		party.Size != 2 || !party.Artillery {
		t.Fatalf("version 3 city state migrated to %+v with raids %+v",
			party, s.Raids)
	}
}

func TestCityBuildsEconomyAndLaunchesArtilleryOnlyOnItsSecondSortie(t *testing.T) {
	s := newGame()
	cityID := s.foundCity(3500, 3200, 0.4)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	s.Cities[cityID] = city
	if city.Stage != len(cityBuildOrder) ||
		len(city.BuildingIDs) != len(cityBuildOrder)+1 {
		t.Fatalf("city has stage %d and %d buildings", city.Stage, len(city.BuildingIDs))
	}
	if !cityHasRepulsor(s, city) ||
		!cityHasBuilding(s, city, EnemyCityOilworks) ||
		!cityHasBuilding(s, city, EnemyCityMine) ||
		!cityHasBuilding(s, city, EnemyCityFactory) {
		t.Fatal("the completed city is missing one of its essential buildings")
	}
	var pylon Enemy
	for _, id := range city.BuildingIDs {
		if e := s.Enemies[id]; e.Kind == EnemyCityRepulsor {
			pylon = e
		}
	}
	for _, id := range city.BuildingIDs {
		e := s.Enemies[id]
		if math.Hypot(e.X-pylon.X, e.Y-pylon.Y)+15 >
			enemySpecOf(EnemyCityRepulsor).bubble {
			t.Fatalf("city building %s lies outside its pylon", e.Kind)
		}
	}
	city.Oil, city.Lilac, city.NextSortie = citySortieOil*2,
		citySortieLilac*2, s.Ticks
	s.Cities[cityID] = city
	stepCity(s, &city)
	s.Cities[cityID] = city
	first := s.Parties[sortedPartyIDs(s)[0]]
	if first.Stage != StageRaid || first.Artillery {
		t.Fatalf("the first city force is %+v, want an immediate force without artillery", first)
	}
	delete(s.Parties, first.ID)
	for _, id := range sortedEnemyIDs(s) {
		if s.Enemies[id].Party == first.ID {
			delete(s.Enemies, id)
		}
	}
	city = s.Cities[cityID]
	city.NextSortie = s.Ticks
	s.Cities[cityID] = city
	stepCity(s, &city)
	second := s.Parties[sortedPartyIDs(s)[0]]
	if !second.Artillery {
		t.Fatal("the second city force did not bring mobile artillery")
	}
	for _, id := range sortedEnemyIDs(s) {
		if e := s.Enemies[id]; e.Party == second.ID && e.Kind == EnemyArtillery {
			return
		}
	}
	t.Fatal("the second force has no mobile artillery unit")
}

func TestCitiesDoNotSendTwoPartiesAtOnce(t *testing.T) {
	s := newGame()
	cities := []City{}
	for _, point := range [][3]float64{
		{3300, 2950, 0.4}, {1700, 2950, 2.4},
	} {
		id := s.foundCity(point[0], point[1], point[2])
		city := s.Cities[id]
		for range cityBuildOrder {
			s.finishCityBuilding(&city)
		}
		city.Oil, city.Lilac, city.NextSortie =
			citySortieOil*2, citySortieLilac*2, s.Ticks
		s.Cities[id] = city
		cities = append(cities, city)
	}
	first := s.Cities[cities[0].ID]
	stepCity(s, &first)
	s.Cities[first.ID] = first
	second := s.Cities[cities[1].ID]
	stepCity(s, &second)
	s.Cities[second.ID] = second
	if len(s.Parties) != 1 || s.Parties[sortedPartyIDs(s)[0]].City != first.ID {
		t.Fatalf("cities launched %d parties while the first force was moving",
			len(s.Parties))
	}
}

func TestReturnedCityForceStartsTheSortieCooldownAtHome(t *testing.T) {
	s := newGame()
	cityID := s.foundCity(4500, 2500, 0)
	party := Party{
		ID: 100, City: cityID, Stage: StageLeave,
	}
	s.Parties[party.ID] = party
	s.Ticks = 600
	s.endParty(party, ReportLeft, 60, 3300, 2950)
	city := s.Cities[cityID]
	if city.NextSortie != s.Ticks+cityRebuildTicks {
		t.Fatalf("next sortie is due at %d, want %d",
			city.NextSortie, s.Ticks+cityRebuildTicks)
	}
	if s.Raids.Visits != 0 {
		t.Fatalf("a city sortie counted as %d intro visits", s.Raids.Visits)
	}
}

func TestReturnedFullCityForceUnloadsThenAttacksAgain(t *testing.T) {
	s := newGame()
	noRivals(s)
	cityID := s.foundCity(4500, 2500, 0)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	city.Oil, city.Lilac = 500, 1000
	s.Cities[cityID] = city
	s.spawnCitySortie(city, false)
	partyID := sortedPartyIDs(s)[0]
	party := s.Parties[partyID]
	if party.Size != 2 {
		t.Fatalf("force has size %d, want 2", party.Size)
	}
	for _, id := range sortedEnemyIDs(s) {
		e := s.Enemies[id]
		if e.Party != partyID {
			continue
		}
		e.Oil = 12
		s.Enemies[id] = e
	}
	party.Stage = StageLeave
	s.Parties[partyID] = party
	stepParty(s, party)
	if s.Parties[partyID].Stage != StageUnload {
		t.Fatalf("returned force is at stage %q, want unload",
			s.Parties[partyID].Stage)
	}
	oil := s.Cities[cityID].Oil
	runTicks(s, 59)
	want := 59 * (2*cityUnloadPerSecond/60 + cityOilExtractPerSecond/60)
	if got := s.Cities[cityID].Oil - oil; math.Abs(got-want) > 0.001 {
		t.Fatalf("the city received %.3f L in 59 ticks, want %.3f L",
			got, want)
	}
	if !tickUntil(s, 10*60, func() bool {
		return s.Parties[partyID].Stage == StageRegroup
	}) {
		t.Fatal("the full force did not rest after unloading")
	}
	if got := s.Cities[cityID].NextSortie; got != s.Ticks+citySortieCooldownTicks {
		t.Fatalf("the next sortie is due at %d, want %d ticks later",
			got, s.Ticks+citySortieCooldownTicks)
	}
	runTicks(s, int(citySortieCooldownTicks)-1)
	if s.Parties[partyID].Stage != StageRegroup {
		t.Fatal("the full force attacked before completing its rest")
	}
	runTicks(s, 1)
	if s.Parties[partyID].Stage != StageRaid {
		t.Fatal("the full force did not attack when its rest ended")
	}
	if got := s.Cities[cityID].Oil; got < 524 {
		t.Fatalf("the city unloaded to %.3f L, want at least 524 L", got)
	}
	if got := len(partyMembers(s, partyID)); got != party.Size {
		t.Fatalf("the force has %d members after unloading, want %d",
			got, party.Size)
	}
}

func TestDamagedCityForceUnloadsThenCompletesItsSquad(t *testing.T) {
	s := newGame()
	noRivals(s)
	cityID := s.foundCity(4500, 2500, 0)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	city.Oil, city.Lilac = 500, 1000
	city.Sorties = 1
	s.Cities[cityID] = city
	s.spawnCitySortie(city, true)
	partyID := sortedPartyIDs(s)[0]
	party := s.Parties[partyID]
	if party.Size != 3 {
		t.Fatalf("force has size %d, want 3", party.Size)
	}
	loaded := false
	for _, id := range sortedEnemyIDs(s) {
		e := s.Enemies[id]
		if e.Party != partyID {
			continue
		}
		if e.Kind == EnemyArtillery {
			delete(s.Enemies, id)
			continue
		}
		if e.Kind == EnemyRaider && e.Oil == 0 && !loaded {
			e.Oil = 6
			s.Enemies[id] = e
			loaded = true
		}
	}
	if got := len(partyMembers(s, partyID)); got != 2 {
		t.Fatalf("force has %d members after losing artillery, want 2", got)
	}
	party.Stage = StageLeave
	s.Parties[partyID] = party
	stepParty(s, party)
	if s.Parties[partyID].Stage != StageUnload {
		t.Fatalf("damaged force is at stage %q, want unload",
			s.Parties[partyID].Stage)
	}
	if !tickUntil(s, 5*60, func() bool {
		return s.Parties[partyID].Stage == StageRebuild
	}) {
		t.Fatalf("the damaged force did not enter squad completion: %+v, %d members",
			s.Parties[partyID], len(partyMembers(s, partyID)))
	}
	runTicks(s, int(cityRebuildTicks)-1)
	if s.Parties[partyID].Stage != StageRebuild ||
		len(partyMembers(s, partyID)) != 2 {
		t.Fatal("the missing vehicle was restored before the one-minute wait")
	}
	runTicks(s, 1)
	if s.Parties[partyID].Stage != StageRaid ||
		len(partyMembers(s, partyID)) != 3 {
		t.Fatal("the force did not complete its ranks and attack again")
	}
	for _, e := range partyMembers(s, partyID) {
		if e.Kind == EnemyArtillery {
			return
		}
	}
	t.Fatal("the damaged force did not replace its artillery")
}

func TestDestroyedCityForceWaitsOneMinuteBeforeRebuilding(t *testing.T) {
	s := newGame()
	noRivals(s)
	cityID := s.foundCity(4500, 2500, 0)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	city.Oil, city.Lilac = 500, 1000
	s.Cities[cityID] = city
	stepCity(s, &city)
	s.Cities[cityID] = city
	partyID := sortedPartyIDs(s)[0]
	for _, id := range sortedEnemyIDs(s) {
		if s.Enemies[id].Party == partyID {
			s.killEnemy(id)
		}
	}
	runTicks(s, 1)
	if len(s.Parties) != 0 {
		t.Fatal("the destroyed force remained in the region")
	}
	if got := s.Cities[cityID].NextSortie - s.Ticks; got != cityRebuildTicks {
		t.Fatalf("the rebuild wait is %d ticks, want %d", got,
			cityRebuildTicks)
	}
	runTicks(s, int(cityRebuildTicks)-1)
	if len(s.Parties) != 0 {
		t.Fatal("the city rebuilt before waiting one minute")
	}
	runTicks(s, 1)
	if len(s.Parties) != 1 {
		t.Fatal("the city did not rebuild a force after one minute")
	}
}

func TestCityDevelopmentAndSortiesSurviveJSONDeterministically(t *testing.T) {
	play := func() *State {
		s := newGame()
		cityID := s.foundCity(3300, 3000, 0.8)
		city := s.Cities[cityID]
		for range cityBuildOrder {
			s.finishCityBuilding(&city)
		}
		city.Oil, city.Lilac, city.NextSortie = 500, 1000, 0
		s.Cities[cityID] = city
		runTicks(s, 8)
		return s
	}
	s := play()
	if !reflect.DeepEqual(s, play()) {
		t.Fatal("identical city states produced different futures")
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, &restored) {
		t.Fatal("city state changed across JSON round trip")
	}
}

func TestDevelopmentActionsFinishOneCityStepAndReleaseItsForce(t *testing.T) {
	s := newGame()
	Apply(s, DevNewCity{})
	if len(s.Cities) != 1 {
		t.Fatalf("new city action created %d cities", len(s.Cities))
	}
	Apply(s, DevFinishCityBuilding{})
	city := s.Cities[sortedCityIDs(s)[0]]
	if city.Stage != 1 || len(city.BuildingIDs) != 2 ||
		!cityHasRepulsor(s, city) || cityHasBuilding(s, city, EnemyBase) {
		t.Fatalf("finish building advanced to stage %d with %d buildings",
			city.Stage, len(city.BuildingIDs))
	}
	for range cityBuildOrder[1:] {
		Apply(s, DevFinishCityBuilding{})
	}
	Apply(s, DevFinishCityBattalion{})
	if len(s.Parties) != 1 {
		t.Fatalf("finish battalion created %d waiting forces", len(s.Parties))
	}
	party := s.Parties[sortedPartyIDs(s)[0]]
	if party.Stage != StageRaid || party.Wait != 0 {
		t.Fatalf("finished force did not attack immediately: %+v", party)
	}
	Apply(s, DevSendCityBattalion{})
	if s.Parties[party.ID].Wait != 0 {
		t.Fatal("send battalion did not release its wait")
	}
	if s.Parties[party.ID].Stage != StageRaid {
		t.Fatal("the city force stopped attacking")
	}
}

func TestDevelopmentActionCanEndAFullCityForceRegroup(t *testing.T) {
	s := newGame()
	cityID := s.foundCity(3500, 3200, 0.4)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	s.Cities[cityID] = city
	s.spawnCitySortie(city, false)
	partyID := sortedPartyIDs(s)[0]
	party := s.Parties[partyID]
	party.Stage = StageRegroup
	party.Wait = citySortieCooldownTicks
	s.Parties[partyID] = party

	Apply(s, DevSendCityBattalion{})
	if s.Parties[partyID].Wait != 0 {
		t.Fatal("send battalion did not end the full force's regroup wait")
	}
	Apply(s, Tick{})
	if s.Parties[partyID].Stage != StageRaid {
		t.Fatal("the full force did not attack after its regroup was ended")
	}
}

func TestWriteCityShotState(t *testing.T) {
	path := os.Getenv("NIEBLA_CITY_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_CITY_SHOT_STATE to write a city shot state")
	}
	s := newGame()
	cityID := s.foundCity(4500, 2500, 0)
	city := s.Cities[cityID]
	for range cityBuildOrder {
		s.finishCityBuilding(&city)
	}
	city.Oil, city.Lilac = 700, 1400
	city.Sorties = 1
	city.NextSortie = s.Ticks
	s.Cities[cityID] = city
	s.finishCitySortie(&city)
	s.Cities[cityID] = city
	Apply(s, DevSendCityBattalion{})
	data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
	if err != nil {
		t.Fatalf("the city state doesn't marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestWriteCityConstructionShotState(t *testing.T) {
	path := os.Getenv("NIEBLA_CITY_CONSTRUCTION_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_CITY_CONSTRUCTION_SHOT_STATE to write " +
			"a city construction state")
	}
	s := newGame()
	cityID := s.foundCity(4500, 2500, 0)
	city := s.Cities[cityID]
	for range 3 {
		s.finishCityBuilding(&city)
	}
	city.Work = cityBuildTicks / 2
	s.Cities[cityID] = city
	data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
	if err != nil {
		t.Fatalf("the city state doesn't marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestWriteCityRebuildShotState(t *testing.T) {
	path := os.Getenv("NIEBLA_CITY_REBUILD_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_CITY_REBUILD_SHOT_STATE to write a rebuild state")
	}
	s := newGame()
	cityID := finishedCityForTest(s)
	city := s.Cities[cityID]
	var nexusID, factoryID int64
	for _, id := range city.BuildingIDs {
		switch s.Enemies[id].Kind {
		case EnemyBase:
			nexusID = id
		case EnemyCityFactory:
			factoryID = id
		}
	}
	s.killEnemy(nexusID)
	s.killEnemy(factoryID)
	city = s.Cities[cityID]
	city.Work = cityBuildTicks / 2
	s.Cities[cityID] = city
	s.Reports = nil
	writeCityShotState(t, path, s)
}

func TestWriteCityRefoundingShotState(t *testing.T) {
	path := os.Getenv("NIEBLA_CITY_REFOUNDING_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_CITY_REFOUNDING_SHOT_STATE to write " +
			"a refounding state")
	}
	s := newGame()
	cityID := finishedCityForTest(s)
	city := s.Cities[cityID]
	for _, id := range append([]int64(nil), city.BuildingIDs...) {
		if cityStageForEnemy(s.Enemies[id].Kind) >= 0 {
			s.killEnemy(id)
		}
	}
	city = s.Cities[cityID]
	city.RefoundAt = s.Ticks
	s.Cities[cityID] = city
	s.Reports = nil
	stepCity(s, &city)
	s.Cities[cityID] = city
	runTicks(s, 12*60)
	writeCityShotState(t, path, s)
}

func writeCityShotState(t *testing.T, path string, s *State) {
	t.Helper()
	data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
	if err != nil {
		t.Fatalf("the city state doesn't marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
