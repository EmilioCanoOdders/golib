package main

import "math"

// Squads: the colony's arm. A war factory builds troopers - robots on a
// combat chassis, which do no work - and the troopers it builds are its
// squad: one unit, ordered as one. A squad guards a spot or attacks a
// rival party, a vehicle of it first; nobody places a trooper by hand.
// A trooper shoots whatever rival comes in its reach, whatever it is
// doing, and pays each shot out of its own tank. The rivals shoot back,
// at troopers and at nothing else.

// Tuning: the squads' numbers, with units in the name.
const (
	warFactoryCostLilac = 250.0 // kg
	warFactoryCostOil   = 50.0  // L

	trooperCostLilac   = 80.0 // kg
	trooperCostOil     = 60.0 // L
	trooperBuildTicks  = 900  // ticks to build one: 15 s
	trooperWreckLilac  = 40.0 // kg a fallen trooper leaves on the ground
	squadSize          = 6    // troopers to a war factory
	trooperHealth      = 80.0
	trooperRangeUnits  = 120.0 // u
	trooperReloadTicks = 30    // ticks between two shots
	trooperShotDamage  = 6.0
	trooperShotOil     = 0.2 // L a shot burns, out of the trooper's own tank

	trooperHealPerSecond = 2.0 // health a trooper mends resting at its spot, under a bubble
	squadStandoff        = 0.8 // the share of its reach an attacker closes to
)

// SquadOrder is what a squad was told to do.
type SquadOrder string

const (
	OrderGuard  SquadOrder = "guard"  // stand at a spot
	OrderAttack SquadOrder = "attack" // go after a rival party
)

// Squad is a war factory's troopers as one unit: its order, and what the
// order is about. It lives under its war factory's ID. A squad nobody
// has ordered yet has no entry, and guards its war factory.
type Squad struct {
	Home  int64 // the war factory's entity ID
	Order SquadOrder
	X, Y  float64 // guard: the spot, in units
	Party int64   // attack: the rival party
	Focus int64   // attack: the vehicle to shoot first
}

// squadOf returns a war factory's squad: the one in the state, or the
// one that guards its own door.
func squadOf(s *State, home int64) Squad {
	if sq, ok := s.Squads[home]; ok {
		return sq
	}
	sq := Squad{Home: home, Order: OrderGuard}
	sq.X, sq.Y = parkCenter()
	if b, ok := s.Buildings[home]; ok {
		sq.X, sq.Y = cellCenterUnits(b.Col, b.Row)
	}
	return sq
}

// squadMembers lists a squad's troopers, in ID order.
func squadMembers(s *State, home int64) []Robot {
	var members []Robot
	for _, id := range sortedRobotIDs(s) {
		if r := s.Robots[id]; r.Kind == RobotCombat && r.Squad == home {
			members = append(members, r)
		}
	}
	return members
}

// squadRoom reports whether a war factory may build one more trooper.
func squadRoom(s *State, b Building) bool {
	building := 0
	if b.Work > 0 {
		building = 1
	}
	return len(squadMembers(s, b.ID))+building < squadSize
}

// stepSquads keeps every order about something: an attack on a party
// that is gone turns back into guarding the war factory's door, and a
// focus that fell passes to the party's leader.
func stepSquads(s *State) {
	for _, id := range sortedSquadIDs(s) {
		sq := s.Squads[id]
		if _, stands := s.Buildings[id]; !stands {
			delete(s.Squads, id)
			continue
		}
		if sq.Order != OrderAttack {
			continue
		}
		members := partyMembers(s, sq.Party)
		if len(members) == 0 {
			delete(s.Squads, id)
			continue
		}
		if _, alive := s.Enemies[sq.Focus]; !alive {
			sq.Focus = members[0].ID
			s.Squads[id] = sq
		}
	}
}

func sortedSquadIDs(s *State) []int64 {
	ids := make([]int64, 0, len(s.Squads))
	for id := range s.Squads {
		ids = append(ids, id)
	}
	sortIDs(ids)
	return ids
}

// squadded claims a trooper's day: all of it that its tank leaves.
func (r *Robot) squadded(s *State) bool {
	return r.Kind == RobotCombat
}

// stepSquad is a trooper's tick under its squad's order: to its place
// around the guarded spot, where it mends under a bubble, or after the
// vehicle its squad is to shoot first, as close as squadStandoff of its
// reach.
func (r *Robot) stepSquad(s *State) {
	sq := squadOf(s, r.Squad)
	place := 0
	for _, other := range squadMembers(s, r.Squad) {
		if other.ID < r.ID {
			place++
		}
	}
	dx, dy := formationOffset(place + 1)
	if target, ok := s.Enemies[sq.Focus]; ok && sq.Order == OrderAttack {
		if math.Hypot(target.X-r.X, target.Y-r.Y) > trooperRangeUnits*squadStandoff {
			r.walkTowards(s, target.X+dx, target.Y+dy)
		}
		return
	}
	if r.walkTowards(s, sq.X+dx, sq.Y+dy) && inSafeZone(s, r.X, r.Y) {
		r.Health = math.Min(trooperHealth, r.Health+trooperHealPerSecond/60)
	}
}

// shoot reloads a trooper's gun and fires it when it is ready, has oil
// for the shot and somebody in reach: its squad's focus before anybody
// else.
func (r *Robot) shoot(s *State) {
	if r.Reload > 0 {
		r.Reload--
		return
	}
	target, found := s.Enemies[squadOf(s, r.Squad).Focus]
	if !found || math.Hypot(target.X-r.X, target.Y-r.Y) > trooperRangeUnits {
		target, found = nearestEnemy(s, r.X, r.Y, trooperRangeUnits)
	}
	if !found || r.Tank < trooperShotOil {
		r.Aim = 0
		return
	}
	r.Tank -= trooperShotOil
	r.Reload, r.Aim = trooperReloadTicks, target.ID
	target.Health -= trooperShotDamage
	s.Enemies[target.ID] = target
	if target.Health <= 0 {
		s.killEnemy(target.ID)
	}
}

// trooperFiring reports whether a trooper's last shot still shows, and
// at whom.
func trooperFiring(s *State, r Robot) (Enemy, bool) {
	if r.Aim == 0 || r.Reload <= trooperReloadTicks-guardFlashTicks {
		return Enemy{}, false
	}
	e, ok := s.Enemies[r.Aim]
	return e, ok
}

// stepEnemyGuns is the rivals shooting back: a vehicle with a gun fires
// at the nearest trooper in its reach, and at nothing else. A trooper
// that falls leaves a wreck's worth of lilac and what its tank held.
func stepEnemyGuns(s *State) {
	for _, id := range sortedEnemyIDs(s) {
		e := s.Enemies[id]
		spec := enemySpecOf(e.Kind)
		if spec.damage <= 0 {
			continue
		}
		if e.Reload > 0 {
			e.Reload--
			s.Enemies[id] = e
			continue
		}
		target, found := nearestTrooper(s, e.X, e.Y, spec.gunRange)
		if !found {
			e.Aim = 0
			s.Enemies[id] = e
			continue
		}
		e.Reload, e.Aim = spec.reload, target.ID
		s.Enemies[id] = e
		target.Health -= spec.damage
		s.Robots[target.ID] = target
		if target.Health <= 0 {
			delete(s.Robots, target.ID)
			col, row := robotCell(target)
			s.dropPile(col, row, target.Tank, trooperWreckLilac)
		}
	}
}

// nearestTrooper returns the trooper closest to a spot, within a reach;
// IDs break ties.
func nearestTrooper(s *State, x, y, reach float64) (Robot, bool) {
	var best Robot
	found, bestGap := false, reach
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		if r.Kind != RobotCombat {
			continue
		}
		if gap := math.Hypot(r.X-x, r.Y-y); gap <= bestGap && (!found || gap < bestGap) {
			best, found, bestGap = r, true, gap
		}
	}
	return best, found
}

// enemyFiring reports whether a vehicle's last shot still shows, and at
// whom.
func enemyFiring(s *State, e Enemy) (Robot, bool) {
	if e.Aim == 0 || e.Reload <= enemySpecOf(e.Kind).reload-guardFlashTicks {
		return Robot{}, false
	}
	r, ok := s.Robots[e.Aim]
	return r, ok
}
