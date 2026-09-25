package main

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

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
		!cityHasBuilding(s, city, EnemyBase) ||
		!cityHasRepulsor(s, city) ||
		s.Enemies[9].Health != enemySpecOf(EnemyBase).health ||
		s.Enemies[9].Reload != 0 {
		t.Fatalf("old settlement migrated to party %+v and city %+v", party, city)
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
	if first.Stage != StageCamp || first.Artillery {
		t.Fatalf("the first city force is %+v, want a waiting force without artillery", first)
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
	if city.NextSortie != s.Ticks+citySortieTicks {
		t.Fatalf("next sortie is due at %d, want %d",
			city.NextSortie, s.Ticks+citySortieTicks)
	}
	if s.Raids.Visits != 0 {
		t.Fatalf("a city sortie counted as %d intro visits", s.Raids.Visits)
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
	if party.Stage != StageCamp || party.Wait <= 0 {
		t.Fatalf("finished force did not wait: %+v", party)
	}
	Apply(s, DevSendCityBattalion{})
	if s.Parties[party.ID].Wait != 0 {
		t.Fatal("send battalion did not release its wait")
	}
	Apply(s, Tick{})
	if s.Parties[party.ID].Stage != StageRaid {
		t.Fatal("released city force did not move on the next tick")
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
