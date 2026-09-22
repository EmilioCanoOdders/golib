package main

// The schematics: the buildings arrive little by little, sent remotely
// to the core. Each drop of the ladder is one tooth of the difficulty
// saw (DESIGN.md, "Introduction to the game"): it comes in a valley of
// calm, just before the thing it answers makes it necessary, never in
// the middle of a peak the player is busy with. A drop arrives when its
// trigger says so - the clock, the rivals' visits, a base that dug in -
// and waits over the core as a glowing badge until the player opens it.
// The trigger is derived from the state, so an old save wakes up with
// exactly what it has earned; only "opened" is written down, in
// State.Tech, so a save with the badge unclicked keeps it.

// Tuning: when the clock drops come in, with units in the name. They
// sit in the long calm between the scout's leaving and the first raid's
// camp. The other teeth answer events: the first delivery brings the
// infrastructure in; the guard post comes when the scout's drawing has
// become inevitable - a rival drinking at the tanks, or the mark already
// on the ground - too late to stop it, in time for the next visit.
const (
	techFrontierTicks = 5*60*60 + 30*60 // mid-valley, two minutes before the raid camps
	techIndustryTicks = 7 * 60 * 60     // the valley's last tooth, before the raid camps
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
	{techInfraID, []BuildingKind{BuildingSilo, BuildingWarehouse, BuildingCharger},
		func(s *State) bool { return s.Deliveries > 0 }},
	{techGuardID, []BuildingKind{BuildingGuard},
		func(s *State) bool {
			if len(s.Marks) > 0 {
				return true // the drawing lies on the ground
			}
			for _, e := range s.Enemies {
				if e.Oil > 0 {
					return true // at the tanks, drinking: the drawing is inevitable
				}
			}
			return false
		}},
	{techFrontierID, []BuildingKind{BuildingProtector, BuildingPump},
		func(s *State) bool { return s.Ticks >= techFrontierTicks }},
	{techIndustryID, []BuildingKind{BuildingFactory},
		func(s *State) bool { return s.Ticks >= techIndustryTicks }},
	{techMobileID, []BuildingKind{BuildingWarFactory},
		func(s *State) bool { return s.Raids.Visits >= 2 }},
	{techArtilleryID, []BuildingKind{BuildingArtillery},
		func(s *State) bool { return settled(s) }},
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
				if techLadder[i].trigger(s) {
					s.Tech[techLadder[i].id] = true
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
