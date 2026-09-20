package main

import (
	"math"
	"sort"
)

// Tuning: the robots' numbers, with units in the name. The core's own
// robots work for nothing and never run dry; the factory's (see
// sim_buildings.go for their cost) burn oil from a tank and answer to
// the fog. A robot is a small, fast rover, 6 u across: it shuttles like
// an ant, carrying a little and coming back for more, so a worked
// deposit shows a constant coming and going.
const (
	startingRobots = 2 // robots the core starts with

	robotSpeed      = 30.0 // units (m) per second
	robotLoadTicks  = 150  // ticks of loading at a deposit: 2.5 s
	robotCarryOil   = 30.0 // liters per trip
	robotCarryLilac = 20.0 // kilograms per trip

	robotParkRadius = 150.0             // units, the idle ring around the core: past the core tile, so the far view shows the dots spread around it
	goldenAngle     = 2.399963229728653 // spreads the parking spots
)

// stepSim moves the world one tick forward: the factories, then the
// robots in ID order, so the outcome never depends on map iteration.
// A built robot empty of oil outside every bubble is digested by the fog
// and leaves the colony.
func stepSim(s *State) {
	stepFactories(s)
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		stepRobot(s, &r)
		if r.Kind == RobotBuilt && r.Tank <= 0 && !inSafeZone(s, r.X, r.Y) {
			delete(s.Robots, id)
			continue
		}
		s.Robots[id] = r
	}
}

func sortedRobotIDs(s *State) []int64 {
	ids := make([]int64, 0, len(s.Robots))
	for id := range s.Robots {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// stepRobot derives what the robot does this tick, in priority order:
// first it brings home what it carries, then it minds its tank — a built
// robot low on oil walks to the nearest charger or the core —, then it
// finishes loading at its post, then it raises the oldest build job, then
// it works its own post, and with nothing of all that it idles by the
// core. A robot arriving home therefore builds first and returns to its
// own task after, exactly as the design asks.
func stepRobot(s *State, r *Robot) {
	job, hasJob := oldestJob(s)
	if r.Kind == RobotBuilt && r.Tank > 0 {
		r.Tank = math.Max(0, r.Tank-robotBurnPerSecond/60)
	}
	switch {
	case r.Carry > 0:
		hx, hy := parkSpot(r.ID)
		if r.walkTowards(s, hx, hy) {
			s.deposit(r)
		}
	case r.refueling(s):
		x, y := refuelSpot(s, *r)
		if r.walkTowards(s, x, y) {
			s.refill(r)
		}
	case r.WorkTicks > 0:
		r.WorkTicks--
		if r.WorkTicks == 0 {
			s.takeLoad(r)
		}
	case hasJob:
		// Builders stand on their cell's edge, spread by ID, so the
		// rising body doesn't swallow them.
		cx, cy := cellCenterUnits(job.Col, job.Row)
		angle := float64(r.ID) * goldenAngle
		if r.walkTowards(s, cx+math.Cos(angle)*11, cy+math.Sin(angle)*11) {
			s.workJob()
		}
	case r.hasPost() && remainingAt(s, r.PostCol, r.PostRow) > 0:
		cx, cy := tileCenterUnits(r.PostCol, r.PostRow)
		if r.walkTowards(s, cx, cy) {
			r.WorkTicks = robotLoadTicks
		}
	default:
		px, py := parkSpot(r.ID)
		r.walkTowards(s, px, py)
	}
}

// refueling reports whether the tank owns the robot's day: low, it
// claims it and walks to the nearest refill post; there it stands until
// the tank is full, even past the low line, so it doesn't dance between
// post and work.
func (r *Robot) refueling(s *State) bool {
	if r.Kind != RobotBuilt || r.Tank >= robotTankLiters {
		return false
	}
	if r.Tank < robotTankLiters*robotLowTankAt {
		return true
	}
	x, y := refuelSpot(s, *r)
	return math.Hypot(r.X-x, r.Y-y) < 0.5
}

// chargeStatus names what a built robot's tank has it doing, when it has
// any claim: "" when the tank is fine, else low oil on the walk over,
// refueling while it fills, or out of oil waiting for the stores.
func chargeStatus(s *State, r Robot) string {
	if r.Kind != RobotBuilt {
		return ""
	}
	x, y := refuelSpot(s, r)
	at := math.Hypot(r.X-x, r.Y-y) < 0.5
	switch {
	case at && r.Tank < robotTankLiters:
		if s.Stock.Oil <= 0 {
			return "out of oil"
		}
		return "refueling"
	case !at && r.Tank < robotTankLiters*robotLowTankAt:
		return "low oil"
	}
	return ""
}

// refill pours oil from the colony's stores into the tank, at the
// charger's pace; dry stores leave the robot standing there.
func (s *State) refill(r *Robot) {
	fill := math.Min(chargerRefillPerSec/60, robotTankLiters-r.Tank)
	fill = math.Min(fill, s.Stock.Oil)
	s.Stock.Oil -= fill
	r.Tank += fill
}

// walkTowards moves the robot one tick's step towards a point and
// reports whether it arrived. The fog drags at every robot, core or
// built, slowing it the deeper the tile sits in the mist; inside a
// bubble the step is whole.
func (r *Robot) walkTowards(s *State, x, y float64) bool {
	step := robotSpeed / 60
	step *= 1 - (1-fogSpeedFactor)*fogAt(s, r.X, r.Y)
	dx, dy := x-r.X, y-r.Y
	d := math.Hypot(dx, dy)
	if d <= step {
		r.X, r.Y = x, y
		return true
	}
	r.X += dx / d * step
	r.Y += dy / d * step
	return false
}

// takeLoad fills the robot's arms from its post's patch and sends it
// home with them. A patch that ran dry releases the robot.
func (s *State) takeLoad(r *Robot) {
	d, ok := depositAt(r.PostCol, r.PostRow)
	if !ok {
		r.clearPost()
		return
	}
	key := depositKey(d)
	remaining := s.Drain[key]
	capacity := robotCarryLilac
	if d.Kind == kindOil {
		capacity = robotCarryOil
	}
	take := math.Min(capacity, remaining)
	if take <= 0 {
		r.clearPost()
		return
	}
	s.Drain[key] = remaining - take
	r.Carry = take
	if d.Kind == kindOil {
		r.Cargo = TypeOil
	} else {
		r.Cargo = TypeLilac
	}
	if s.Drain[key] <= 0 {
		r.clearPost()
	}
}

// deposit empties the robot's arms into the colony's stores, as much of
// the load as their room takes; a store already full leaves the rest in
// its arms, and the robot stands at the core trying again every tick.
func (s *State) deposit(r *Robot) {
	var room float64
	switch r.Cargo {
	case TypeOil:
		room = oilCap(s) - s.Stock.Oil
	case TypeLilac:
		room = lilacCap(s) - s.Stock.Lilac
	default:
		r.Carry, r.Cargo = 0, ""
		return
	}
	put := math.Min(r.Carry, math.Max(0, room))
	switch r.Cargo {
	case TypeOil:
		s.Stock.Oil += put
	case TypeLilac:
		s.Stock.Lilac += put
	}
	r.Carry -= put
	if r.Carry < 0.0001 {
		r.Carry, r.Cargo = 0, ""
	}
}

// workJob puts a tick of work into the oldest job; done, the building
// the job asked for rises from the ground.
func (s *State) workJob() {
	if len(s.Jobs) == 0 {
		return
	}
	s.Jobs[0].Left--
	if s.Jobs[0].Left <= 0 {
		job := s.Jobs[0]
		s.Jobs = s.Jobs[1:]
		if job.Kind != "" {
			s.raise(job.Kind, job.Col, job.Row)
		}
	}
}

// parkSpot returns where a robot idles: a spot on a small ring around
// the core, spread by the golden angle, so idle robots never stack.
func parkSpot(id int64) (x, y float64) {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	angle := float64(id) * goldenAngle
	return cx + math.Cos(angle)*robotParkRadius,
		cy + math.Sin(angle)*robotParkRadius
}

// oldestJob returns the build job that has waited longest.
func oldestJob(s *State) (Job, bool) {
	if len(s.Jobs) == 0 {
		return Job{}, false
	}
	return s.Jobs[0], true
}

// remainingAt returns what is left in the deposit patch a tile belongs
// to; a tile that holds no deposit has nothing.
func remainingAt(s *State, col, row int) float64 {
	d, ok := depositAt(col, row)
	if !ok {
		return 0
	}
	return s.Drain[depositKey(d)]
}

// postOwner returns the robot whose post is a tile of the deposit patch
// the given tile belongs to: a patch has one robot.
func postOwner(s *State, col, row int) (Robot, bool) {
	want, ok := depositAt(col, row)
	if !ok {
		return Robot{}, false
	}
	for _, id := range sortedRobotIDs(s) {
		if r := s.Robots[id]; r.hasPost() {
			if d, ok := depositAt(r.PostCol, r.PostRow); ok && d == want {
				return r, true
			}
		}
	}
	return Robot{}, false
}

// pickRobot chooses who takes a post: a free robot if there is one,
// else the one whose walk is shortest; IDs break ties.
func pickRobot(s *State, col, row int) int64 {
	cx, cy := tileCenterUnits(col, row)
	best := int64(-1)
	bestFree, bestDist := false, 0.0
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		free := !r.hasPost()
		d := math.Hypot(r.X-cx, r.Y-cy)
		closer := best < 0 ||
			(free && !bestFree) ||
			(free == bestFree && d < bestDist)
		if closer {
			best, bestFree, bestDist = id, free, d
		}
	}
	return best
}
