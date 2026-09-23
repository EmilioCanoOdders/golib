package main

import (
	"fmt"
	"math"
)

const (
	cityFirstDelayCycles = 30
	cityIntervalCycles   = 30
	cityLimit            = 3
	cityRadiusTiles      = 10.0
	cityMinSeparation    = 900.0
	cityBuildTicks       = 90 * 60
	cityOilReserve       = 900.0
	cityLilacReserve     = 1800.0
	citySortieTicks      = 5 * 60 * 60
)

type City struct {
	ID           int64
	NexusID      int64 // Reserved at founding; built after the pylon
	X, Y         float64
	Angle        float64
	Stage        int
	Work         int64
	Oil          float64
	Lilac        float64
	OilDeposit   float64
	LilacDeposit float64
	NextSortie   int64
	Sorties      int64
	BuildingIDs  []int64
}

const (
	CityCore     BuildingKind = "citycore"
	CityRepulsor BuildingKind = "cityrepulsor"
	CityOilworks BuildingKind = "cityoilworks"
	CityMine     BuildingKind = "citymine"
	CityFactory  BuildingKind = "cityfactory"
)

var cityBuildOrder = []BuildingKind{
	CityRepulsor, CityCore, CityOilworks, CityMine, CityFactory,
}

func sortedCityIDs(s *State) []int64 {
	ids := make([]int64, 0, len(s.Cities))
	for id := range s.Cities {
		ids = append(ids, id)
	}
	sortIDs(ids)
	return ids
}

func (s *State) migrateSettledCities() {
	if len(s.Cities) != 0 {
		return
	}
	for _, partyID := range sortedPartyIDs(s) {
		if s.Parties[partyID].Stage != StageSettled {
			continue
		}
		for _, enemyID := range sortedEnemyIDs(s) {
			e := s.Enemies[enemyID]
			if e.Party != partyID || e.Kind != EnemyBase {
				continue
			}
			cityID := s.NextID
			s.NextID++
			repulsorID := s.NextID
			s.NextID++
			nexusID := enemyID
			e.Health = clamp64(e.Health/900.0, 0, 1) * enemySpecOf(EnemyBase).health
			e.Reload, e.Aim = 0, 0
			s.Cities = map[int64]City{
				cityID: {
					ID: cityID, X: e.X, Y: e.Y, Stage: 2,
					NexusID: nexusID,
					Work:    cityBuildTicks, OilDeposit: cityOilReserve,
					LilacDeposit: cityLilacReserve,
					BuildingIDs:  []int64{enemyID, repulsorID},
				},
			}
			e.City = cityID
			s.Enemies[enemyID] = e
			s.Enemies[repulsorID] = Enemy{
				ID: repulsorID, Kind: EnemyCityRepulsor,
				X: e.X, Y: e.Y, Health: enemySpecOf(EnemyCityRepulsor).health,
				City: cityID,
			}
			p := s.Parties[partyID]
			p.City = cityID
			s.Parties[partyID] = p
			return
		}
	}
}

func (s *State) foundCity(x, y, angle float64) int64 {
	if s.Cities == nil {
		s.Cities = map[int64]City{}
	}
	if s.Enemies == nil {
		s.Enemies = map[int64]Enemy{}
	}
	id := s.NextID
	s.NextID++
	rigID := s.NextID
	s.NextID++
	nexusID := s.NextID
	s.NextID++
	city := City{
		ID: id, NexusID: nexusID, X: x, Y: y, Angle: angle,
		Work: cityBuildTicks, OilDeposit: cityOilReserve,
		LilacDeposit: cityLilacReserve,
		BuildingIDs:  []int64{rigID},
	}
	rigX, rigY := cityCrawlerPosition(city)
	s.Enemies[rigID] = Enemy{
		ID: rigID, Kind: EnemyCityCrawler, X: rigX, Y: rigY,
		Health: enemySpecOf(EnemyCityCrawler).health, City: id,
	}
	s.Cities[id] = city
	s.report(ReportSettled, 0, x, y)
	return id
}

func (s *State) nextCitySpot() (float64, float64, float64, bool) {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	mostX := float64(regionCols*unitsPerTile) - 1
	mostY := float64(regionRows*unitsPerTile) - 1
	for range 32 {
		angle := s.roll() * 2 * math.Pi
		x := clamp64(cx+math.Cos(angle)*cityRadiusTiles*unitsPerTile, 1, mostX)
		y := clamp64(cy+math.Sin(angle)*cityRadiusTiles*unitsPerTile, 1, mostY)
		clear := true
		for _, id := range sortedCityIDs(s) {
			city := s.Cities[id]
			if math.Hypot(x-city.X, y-city.Y) < cityMinSeparation {
				clear = false
				break
			}
		}
		for _, partyID := range sortedPartyIDs(s) {
			party := s.Parties[partyID]
			if !party.CityArrives || party.Stage != StageApproach {
				continue
			}
			if math.Hypot(x-party.CampX, y-party.CampY) < cityMinSeparation {
				clear = false
				break
			}
		}
		if clear {
			return x, y, angle, true
		}
	}
	return 0, 0, 0, false
}

func (s *State) foundCityFromCrawler(crawler Enemy, angle float64) int64 {
	if s.Cities == nil {
		s.Cities = map[int64]City{}
	}
	id := s.NextID
	s.NextID++
	nexusID := s.NextID
	s.NextID++
	s.Cities[id] = City{
		ID: id, NexusID: nexusID,
		X: crawler.X, Y: crawler.Y, Angle: angle,
		Work:       cityBuildTicks,
		OilDeposit: cityOilReserve, LilacDeposit: cityLilacReserve,
		BuildingIDs: []int64{crawler.ID},
	}
	return id
}

func (s *State) spawnCityVisit() bool {
	if s.Enemies == nil {
		s.Enemies = map[int64]Enemy{}
	}
	if s.Parties == nil {
		s.Parties = map[int64]Party{}
	}
	campX, campY, angle, ok := s.nextCitySpot()
	if !ok {
		return false
	}
	coreX, coreY := tileCenterUnits(coreCol, coreRow)
	point := func(radius float64) (float64, float64) {
		return clamp64(coreX+math.Cos(angle)*radius*unitsPerTile, 1,
				float64(regionCols*unitsPerTile)-1),
			clamp64(coreY+math.Sin(angle)*radius*unitsPerTile, 1,
				float64(regionRows*unitsPerTile)-1)
	}
	ex, ey := point(entryRadiusTiles)
	partyID := s.NextID
	s.NextID++
	party := Party{
		ID: partyID, Stage: StageApproach,
		EntryX: campX, EntryY: campY, CampX: campX, CampY: campY,
		CityArrives: true,
	}
	s.Parties[partyID] = party
	id := s.NextID
	s.NextID++
	s.Enemies[id] = Enemy{
		ID: id, Kind: EnemyCrawler, Party: partyID,
		X: ex, Y: ey, Health: enemySpecOf(EnemyCrawler).health,
	}
	s.report(ReportCityIncoming, 0, ex, ey)
	return true
}

func stepCities(s *State) {
	for _, id := range sortedCityIDs(s) {
		city := s.Cities[id]
		stepCity(s, &city)
		s.Cities[id] = city
	}
}

func stepCity(s *State, city *City) {
	if city.Stage < len(cityBuildOrder) {
		city.Work--
		if city.Work <= 0 {
			s.finishCityBuilding(city)
		}
		return
	}
	if cityHasBuilding(s, *city, EnemyCityOilworks) && city.OilDeposit > 0 {
		amount := math.Min(cityOilExtractPerSecond/60, city.OilDeposit)
		city.OilDeposit -= amount
		city.Oil += amount
	}
	if cityHasBuilding(s, *city, EnemyCityMine) && city.LilacDeposit > 0 {
		amount := math.Min(cityLilacExtractPerSecond/60, city.LilacDeposit)
		city.LilacDeposit -= amount
		city.Lilac += amount
	}
	if movingParty(s) {
		return
	}
	if !cityHasBuilding(s, *city, EnemyCityFactory) || city.NextSortie > s.Ticks ||
		city.Oil < citySortieOil || city.Lilac < citySortieLilac {
		return
	}
	city.Oil -= citySortieOil
	city.Lilac -= citySortieLilac
	city.Sorties++
	city.NextSortie = s.Ticks + citySortieTicks
	s.spawnCitySortie(*city, city.Sorties >= 2)
}

const (
	cityOilExtractPerSecond   = 2.0
	cityLilacExtractPerSecond = 4.0
	citySortieOil             = 120.0
	citySortieLilac           = 240.0
)

type cityBuildingSpecValue struct {
	kind   EnemyKind
	health float64
}

func cityBuildingSpec(kind BuildingKind) cityBuildingSpecValue {
	switch kind {
	case CityCore:
		return cityBuildingSpecValue{EnemyBase, 1200}
	case CityRepulsor:
		return cityBuildingSpecValue{EnemyCityRepulsor, 300}
	case CityOilworks:
		return cityBuildingSpecValue{EnemyCityOilworks, 200}
	case CityMine:
		return cityBuildingSpecValue{EnemyCityMine, 200}
	default:
		return cityBuildingSpecValue{EnemyCityFactory, 250}
	}
}

func (s *State) spawnCitySortie(city City, artillery bool) {
	partyID := s.NextID
	s.NextID++
	count := raidFirstRaiders
	if city.Sorties > 2 {
		count = raidersOf(city.Sorties - 1)
	}
	p := Party{
		ID: partyID, Stage: StageCamp, City: city.ID,
		Wait: campPrepareTicks, Siphon: raidSiphonTicks, Artillery: artillery,
		CampX: city.X, CampY: city.Y,
		EntryX: city.X, EntryY: city.Y,
	}
	if s.Parties == nil {
		s.Parties = map[int64]Party{}
	}
	s.Parties[partyID] = p
	for i := 0; i < count; i++ {
		dx, dy := formationOffset(i)
		id := s.NextID
		s.NextID++
		s.Enemies[id] = Enemy{
			ID: id, Kind: EnemyRaider, Party: partyID,
			City: city.ID,
			X:    city.X + dx, Y: city.Y + dy,
			Health: enemySpecOf(EnemyRaider).health,
		}
	}
	if artillery {
		id := s.NextID
		s.NextID++
		s.Enemies[id] = Enemy{
			ID: id, Kind: EnemyArtillery, Party: partyID,
			City: city.ID,
			X:    city.X, Y: city.Y,
			Health: enemySpecOf(EnemyArtillery).health,
		}
	}
	s.report(ReportSortie, float64(count), city.X, city.Y)
}

func cityHasBuilding(s *State, city City, kind EnemyKind) bool {
	for _, id := range city.BuildingIDs {
		if e, ok := s.Enemies[id]; ok && e.Kind == kind {
			return true
		}
	}
	return false
}

func (s *State) finishCityBuilding(city *City) {
	if city.Stage >= len(cityBuildOrder) {
		return
	}
	stage := city.Stage
	kind := cityBuildOrder[stage]
	eid := int64(0)
	if kind == CityCore && city.NexusID != 0 {
		eid = city.NexusID
	} else {
		eid = s.NextID
		s.NextID++
		if kind == CityCore {
			city.NexusID = eid
		}
	}
	spec := cityBuildingSpec(kind)
	x, y := cityBuildingPosition(*city, city.Stage)
	s.Enemies[eid] = Enemy{
		ID: eid, Kind: spec.kind, X: x, Y: y,
		Health: spec.health, City: city.ID,
	}
	city.BuildingIDs = append(city.BuildingIDs, eid)
	city.Stage++
	city.Work = cityBuildTicks
	s.report(ReportCityBuilding, 0, city.X, city.Y)
	s.Reports[len(s.Reports)-1].Stage = int64(stage)
	if city.Stage == len(cityBuildOrder) {
		city.NextSortie = s.Ticks + citySortieTicks
	}
}

func (s *State) finishCitySortie(city *City) {
	if movingParty(s) {
		return
	}
	if !cityHasBuilding(s, *city, EnemyCityFactory) {
		return
	}
	city.Sorties++
	s.spawnCitySortie(*city, city.Sorties >= 2)
	city.NextSortie = s.Ticks + citySortieTicks
}

func movingParty(s *State) bool {
	for _, id := range sortedPartyIDs(s) {
		if s.Parties[id].Stage != StageSettled {
			return true
		}
	}
	return false
}

func cityHasRepulsor(s *State, city City) bool {
	for _, id := range city.BuildingIDs {
		if e, ok := s.Enemies[id]; ok && e.Kind == EnemyCityRepulsor {
			return true
		}
	}
	return false
}

func cityBuildingPosition(city City, stage int) (float64, float64) {
	spots := [][2]float64{{0, 0}, {0, -80}, {-75, 0}, {75, 0}, {0, 80}}
	offset := spots[stage]
	return cityOffset(city, offset[0], offset[1])
}

func cityCrawlerPosition(city City) (float64, float64) {
	return cityOffset(city, 0, 150)
}

func cityOffset(city City, dx, dy float64) (float64, float64) {
	cos, sin := math.Cos(city.Angle), math.Sin(city.Angle)
	return city.X + dx*cos - dy*sin,
		city.Y + dx*sin + dy*cos
}

func cityBuildingName(kind BuildingKind) string {
	switch kind {
	case CityRepulsor:
		return "antimist post"
	case CityCore:
		return "Nexus"
	case CityOilworks:
		return "oil extractor"
	case CityMine:
		return "lilac mine"
	case CityFactory:
		return "military factory"
	}
	return string(kind)
}

func cityBuildingCaption(s *State, e Enemy) string {
	switch e.Kind {
	case EnemyCityCrawler:
		return "city construction rig"
	case EnemyBase:
		return "Nexus"
	case EnemyCityRepulsor:
		return "repelling the fog"
	case EnemyCityOilworks:
		return "extracting oil"
	case EnemyCityMine:
		return "mining lilac"
	case EnemyCityFactory:
		return "building the next force"
	}
	if city, ok := s.Cities[e.City]; ok && city.Stage < len(cityBuildOrder) {
		return fmt.Sprintf("city construction: %s",
			cityBuildingName(cityBuildOrder[city.Stage]))
	}
	return "city structure"
}
