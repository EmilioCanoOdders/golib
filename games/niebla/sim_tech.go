package main

// The schematics: the buildings arrive little by little, sent remotely
// to the core. Each drop of the ladder is one tooth of the difficulty
// saw (DESIGN.md, "Introduction to the game"): it comes in a valley of
// calm, just before the thing it answers makes it necessary, never in
// the middle of a peak the player is busy with. A drop arrives when its
// trigger says so - the clock, the rivals' visits, a city factory -
// and waits over the core as a glowing badge until the player opens it.
// The trigger is derived from the state, so an old save wakes up with
// exactly what it has earned; only "opened" is written down, in
// State.Tech, so a save with the badge unclicked keeps it.

// Tuning: the factory blueprint is the opening gift. The first delivery
// brings logistics. The frontier kit's clock drop sits before the first
// raid; the guard answers the scout's theft, and the war factory arrives
// when the first city is founded.
const (
	techFrontierTicks       = 5*60*60 + 30*60 // mid-valley, before the first raid
	legacyTechIndustryTicks = 7 * 60 * 60     // old saves' factory unlock
	techRepairFallbackTicks = 12 * 60 * 60    // no first city force by minute 12
)

// The drops' names. Treat them as identifiers, not prose: the view keys
// its words and marks by them, and AckTech takes one.
const (
	techInfraID     = "infra"
	techGuardID     = "guard"
	techFrontierID  = "frontier"
	techIndustryID  = "industry"
	techMobileID    = "mobile"
	techArtilleryID = "artillery"
	techRepairID    = "repair"
)

// techDrop is one rung of the ladder: the schematics it brings, and
// what has to be true for them to arrive.
type techDrop struct {
	id      string
	kinds   []BuildingKind
	trigger func(s *State) bool
}

// techLadder is the order the drops arrive in, which is also the order
// the badges queue in when more than one waits. The pump is not in the
// build menu, so it rides with the frontier kit; so do the pipes, whose
// LayPipe asks for the frontier drop by name.
var techLadder = []techDrop{
	{techIndustryID, []BuildingKind{BuildingFactory},
		func(s *State) bool { return true }},
	{techInfraID, []BuildingKind{BuildingSilo, BuildingWarehouse, BuildingCharger},
		func(s *State) bool { return s.Deliveries > 0 }},
	{techGuardID, []BuildingKind{BuildingGuard},
		func(s *State) bool { return s.Raids.ScoutClearedCore }},
	{techFrontierID, []BuildingKind{BuildingProtector, BuildingPump},
		func(s *State) bool { return s.Ticks >= techFrontierTicks }},
	{techMobileID, []BuildingKind{BuildingWarFactory},
		func(s *State) bool {
			return s.Raids.PressureCity != 0 || s.Raids.Visits >= 2
		}},
	{techArtilleryID, []BuildingKind{BuildingArtillery},
		func(s *State) bool {
			for _, cityID := range sortedCityIDs(s) {
				if cityHasBuilding(s, s.Cities[cityID], EnemyCityFactory) {
					return true
				}
			}
			return false
		}},
	{techRepairID, nil, repairProtocolTrigger},
}

func repairProtocolTrigger(s *State) bool {
	if s.Raids.LegacyRepairUnlocked || !s.Raids.RivalBuildingHit ||
		s.Raids.Visits < 2 ||
		!rivalForcesInLull(s) {
		return false
	}
	if s.Raids.PressureSortieStarted {
		return s.Raids.PressureSortieResolved
	}
	if s.Ticks < techRepairFallbackTicks ||
		pressureCitySortieReady(s) {
		return false
	}
	return true
}

func repairProtocolUnlocked(s *State) bool {
	return s.Raids.LegacyRepairUnlocked || dropArrived(s, techRepairID)
}

func rivalForcesInLull(s *State) bool {
	for _, id := range sortedPartyIDs(s) {
		switch s.Parties[id].Stage {
		case StageApproach, StageCamp, StageRaid, StageLeave:
			return false
		}
	}
	return true
}

func pressureCitySortieReady(s *State) bool {
	city, exists := s.Cities[s.Raids.PressureCity]
	if !exists || s.Raids.PressureSortieStarted || movingParty(s) {
		return false
	}
	if city.Stage == len(cityBuildOrder)-1 && city.Work <= 1 {
		return true
	}
	if city.Stage < len(cityBuildOrder) ||
		!cityHasBuilding(s, city, EnemyCityFactory) ||
		city.NextSortie > s.Ticks {
		return false
	}

	oil := city.Oil
	lilac := city.Lilac
	if cityHasBuilding(s, city, EnemyCityOilworks) && city.OilDeposit > 0 {
		oil += min(cityOilExtractPerSecond/60, city.OilDeposit)
	}
	if cityHasBuilding(s, city, EnemyCityMine) && city.LilacDeposit > 0 {
		lilac += min(cityLilacExtractPerSecond/60, city.LilacDeposit)
	}
	return oil >= citySortieOil && lilac >= citySortieLilac
}

func hasBuiltWorker(s *State) bool {
	for _, id := range sortedRobotIDs(s) {
		if s.Robots[id].Kind == RobotWorker {
			return true
		}
	}
	return false
}

// legacyTechArrived preserves the old ladder for saves without its ledger.
func legacyTechArrived(s *State, id string) bool {
	switch id {
	case techInfraID:
		return s.Deliveries > 0
	case techIndustryID:
		return s.Ticks >= legacyTechIndustryTicks
	case techRepairID:
		return false
	case techGuardID:
		return legacyGuardTechArrived(s)
	}
	for i := range techLadder {
		if techLadder[i].id == id {
			return techLadder[i].trigger(s)
		}
	}
	return false
}

func legacyGuardTechArrived(s *State) bool {
	if len(s.Marks) > 0 {
		return true
	}
	for _, e := range s.Enemies {
		if e.Oil > 0 {
			return true
		}
	}
	return false
}

// stepTech writes down what has arrived, and is the first thing the
// simulation does each tick, so no rule ever sees a stale ladder.
func stepTech(s *State) {
	if s.Tech == nil {
		s.Tech = map[string]bool{}
		if s.Ticks > 1 {
			// A save from before the schematics: every drop it has
			// already earned stands opened, so nobody clicks through a
			// pile of badges after the update.
			for i := range techLadder {
				id := techLadder[i].id
				if legacyTechArrived(s, id) {
					s.Tech[id] = true
				}
			}
		}
	}
	for i := range techLadder {
		d := &techLadder[i]
		if _, ok := s.Tech[d.id]; !ok && d.trigger(s) {
			s.Tech[d.id] = false
		}
	}
}

// dropArrived reports whether a drop has come in: it is in the state's
// ledger, or its trigger is true right now. The trigger check keeps a
// fresh state honest before its first Tick.
func dropArrived(s *State, id string) bool {
	for i := range techLadder {
		d := &techLadder[i]
		if d.id != id {
			continue
		}
		if _, ok := s.Tech[id]; ok {
			return true
		}
		return d.trigger(s)
	}
	return false
}

// kindUnlocked reports whether a blueprint's schematics have arrived.
func kindUnlocked(s *State, kind BuildingKind) bool {
	for i := range techLadder {
		d := &techLadder[i]
		for _, k := range d.kinds {
			if k != kind {
				continue
			}
			return dropArrived(s, d.id)
		}
	}
	return false
}

// techAnyArrived reports whether anything at all has arrived: the build
// menu only opens once something can be raised.
func techAnyArrived(s *State) bool {
	return len(s.Tech) > 0
}

// techPending returns the oldest arrived drop the player hasn't opened,
// or "" when none waits: the badge over the core, one at a time.
func techPending(s *State) string {
	for i := range techLadder {
		d := &techLadder[i]
		if !dropArrived(s, d.id) {
			continue
		}
		if opened, ok := s.Tech[d.id]; !ok || !opened {
			return d.id
		}
	}
	return ""
}
