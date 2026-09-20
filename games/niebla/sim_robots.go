package main

import (
	"math"
	"sort"
)

// Tuning: the robots' numbers, with units in the name. The starting
// robots work for nothing; charge and upkeep arrive with built robots.
const (
	startingRobots = 2 // robots the core starts with

	robotSpeed      = 3.0  // units (m) per second
	robotLoadTicks  = 150  // ticks of loading at a deposit: 2.5 s
	robotCarryOil   = 90.0 // liters per trip
	robotCarryLilac = 60.0 // kilograms per trip

	robotParkRadius = 3.5  // units, the idle ring around the core
	goldenAngle     = 2.399963229728653 // spreads the parking spots
)

// stepSim moves the world one tick forward: the robots, in ID order, so
// the outcome never depends on map iteration.
func stepSim(s *State) {
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		stepRobot(s, &r)
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
// first it brings home what it carries, then it finishes loading at its
// post, then it raises the oldest build job, then it works its own
// post, and with nothing of all that it idles by the core. A robot
// arriving home therefore builds first and returns to its own task
// after, exactly as the design asks.
func stepRobot(s *State, r *Robot) {
	job, hasJob := oldestJob(s)
	switch {
	case r.Carry > 0:
		if r.walkTowards(parkSpot(r.ID)) {
			s.deposit(r)
		}
	case r.WorkTicks > 0:
		r.WorkTicks--
		if r.WorkTicks == 0 {
			s.takeLoad(r)
		}
	case hasJob:
		cx, cy := tileCenterUnits(job.Col, job.Row)
		if r.walkTowards(cx, cy) {
			s.workJob()
		}
	case r.hasPost() && remainingAt(s, r.PostCol, r.PostRow) > 0:
		cx, cy := tileCenterUnits(r.PostCol, r.PostRow)
		if r.walkTowards(cx, cy) {
			r.WorkTicks = robotLoadTicks
		}
	default:
		r.walkTowards(parkSpot(r.ID))
	}
}

// walkTowards moves the robot one tick's step towards a point and
// reports whether it arrived.
func (r *Robot) walkTowards(x, y float64) bool {
	step := robotSpeed / 60
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

// takeLoad fills the robot's arms from its post and sends it home with
// them. A post that ran dry releases the robot.
func (s *State) takeLoad(r *Robot) {
	remaining := remainingAt(s, r.PostCol, r.PostRow)
	capacity := robotCarryLilac
	if tileAt(r.PostCol, r.PostRow) == kindOil {
		capacity = robotCarryOil
	}
	take := math.Min(capacity, remaining)
	if take <= 0 {
		r.clearPost()
		return
	}
	key := drainKey(r.PostCol, r.PostRow)
	s.Drain[key] = remaining - take
	r.Carry = take
	if tileAt(r.PostCol, r.PostRow) == kindOil {
		r.Cargo = TypeOil
	} else {
		r.Cargo = TypeLilac
	}
	if s.Drain[key] <= 0 {
		r.clearPost()
	}
}

// deposit empties the robot's arms into the core's stores.
func (s *State) deposit(r *Robot) {
	switch r.Cargo {
	case TypeOil:
		s.Stock.Oil += r.Carry
	case TypeLilac:
		s.Stock.Lilac += r.Carry
	}
	r.Carry = 0
	r.Cargo = ""
}

// workJob puts a tick of work into the oldest job; done, it leaves the
// list, and the robots go back to their own tasks. A later slice raises
// the building from the finished job.
func (s *State) workJob() {
	if len(s.Jobs) == 0 {
		return
	}
	s.Jobs[0].Left--
	if s.Jobs[0].Left <= 0 {
		s.Jobs = s.Jobs[1:]
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

// remainingAt returns what is left in a deposit tile; a tile that holds
// no deposit has nothing.
func remainingAt(s *State, col, row int) float64 {
	return s.Drain[drainKey(col, row)]
}

// postOwner returns the robot whose post is this tile.
func postOwner(s *State, col, row int) (Robot, bool) {
	for _, id := range sortedRobotIDs(s) {
		if r := s.Robots[id]; r.atPost(col, row) {
			return r, true
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
