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

// stepSim moves the world one tick forward: the weather, the
// factories, the pipes, then the robots in ID order, so the outcome never depends
// on map iteration. A built robot empty of oil outside every bubble is
// digested by the fog and leaves the colony.
func stepSim(s *State) {
	stepFog(s)
	stepFactories(s)
	stepPipes(s)
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

// robotTask is one line of a robot's day: its name, whether it claims
// the robot this tick, and the tick of it.
type robotTask struct {
	name   string
	claims func(r *Robot, s *State) bool
	step   func(r *Robot, s *State)
}

// The lines of a robot's day, by name.
const (
	taskHaul    = "haul"    // bring home what it carries
	taskRefuel  = "refuel"  // mind its tank
	taskLoad    = "load"    // finish loading, at its post or at a pile
	taskBuild   = "build"   // raise the oldest build job, then lay pipe
	taskCollect = "collect" // pick up loose items
	taskPost    = "post"    // work its own post
	taskIdle    = "idle"    // stand by the core
)

// robotDay is the robot's day, in priority order: the first line that
// claims the robot owns its tick. It brings home what it carries, minds
// its tank — a built robot low on oil walks to the nearest charger or
// the core —, finishes loading, raises the oldest build job, picks up
// what lies on the ground before digging more, works its own post, and
// with nothing of all that idles by the core. A robot arriving home
// therefore builds first and returns to its own task after, exactly as
// the design asks.
var robotDay = []robotTask{
	{taskHaul, (*Robot).hauling, (*Robot).stepHaul},
	{taskRefuel, (*Robot).refueling, (*Robot).stepRefuel},
	{taskLoad, (*Robot).loading, (*Robot).stepLoad},
	{taskBuild, (*Robot).building, (*Robot).stepBuild},
	{taskCollect, (*Robot).collecting, (*Robot).stepCollect},
	{taskPost, (*Robot).posted, (*Robot).stepPost},
	{taskIdle, (*Robot).idling, (*Robot).stepIdle},
}

// taskNow returns the line of the day that owns the robot this tick.
func (r *Robot) taskNow(s *State) robotTask {
	for _, task := range robotDay {
		if task.claims(r, s) {
			return task
		}
	}
	return robotDay[len(robotDay)-1]
}

// stepRobot burns the tank and gives the tick to the robot's task. The
// robot carries no plan: the task is derived from the state every tick.
func stepRobot(s *State, r *Robot) {
	if r.Kind == RobotBuilt && r.Tank > 0 {
		burn := robotBurnPerSecond / 60
		if !inSafeZone(s, r.X, r.Y) {
			// The swell's mist is meaner in the open, by as much as it presses.
			burn *= 1 + (fogSwellBurn-1)*s.Fog.Pressure
		}
		r.Tank = math.Max(0, r.Tank-burn)
	}
	r.taskNow(s).step(r, s)
}

func (r *Robot) hauling(s *State) bool {
	return r.Carry > 0
}

func (r *Robot) stepHaul(s *State) {
	x, y := storeSpot(s, *r)
	if r.walkTowards(s, x, y) {
		s.deposit(r)
	}
}

func (r *Robot) stepRefuel(s *State) {
	x, y := refuelSpot(s, *r)
	if r.walkTowards(s, x, y) {
		s.refill(r)
	}
}

func (r *Robot) loading(s *State) bool {
	return r.WorkTicks > 0
}

func (r *Robot) stepLoad(s *State) {
	r.WorkTicks--
	if r.WorkTicks > 0 {
		return
	}
	if r.Pile != 0 {
		s.takeFromPile(r)
		return
	}
	s.takeLoad(r)
}

func (r *Robot) building(s *State) bool {
	if _, hasJob := oldestJob(s); hasJob {
		return true
	}
	_, laying := unlaidPipe(s)
	return laying
}

// Builders stand on their cell's edge, spread by ID, so the rising body
// doesn't swallow them. With no site left to raise, they lay pipe.
func (r *Robot) stepBuild(s *State) {
	job, hasJob := oldestJob(s)
	if !hasJob {
		r.stepLayPipe(s)
		return
	}
	cx, cy := cellCenterUnits(job.Col, job.Row)
	angle := float64(r.ID) * goldenAngle
	if r.walkTowards(s, cx+math.Cos(angle)*11, cy+math.Sin(angle)*11) {
		s.workJob()
	}
}

// stepLayPipe works on the oldest unlaid pipe, at its head: the pipe
// grows from its pump out, and whoever lays it walks along with it.
func (r *Robot) stepLayPipe(s *State) {
	p, _ := unlaidPipe(s)
	path, ok := pipeSpine(s, p)
	if !ok {
		return
	}
	head := pathPointAt(path, pathLength(path)*pipeLaidPart(p))
	angle := float64(r.ID) * goldenAngle
	x := head.X + math.Cos(angle)*pipeLayStandoff
	y := head.Y + math.Sin(angle)*pipeLayStandoff
	if r.walkTowards(s, x, y) {
		s.workPipe(p.ID)
	}
}

func (r *Robot) collecting(s *State) bool {
	_, found := nearestPile(s, *r)
	return found
}

func (r *Robot) stepCollect(s *State) {
	p, _ := nearestPile(s, *r)
	x, y := cellCenterUnits(p.Col, p.Row)
	if r.walkTowards(s, x, y) {
		r.WorkTicks = robotLoadTicks
		r.Pile = p.ID
	}
}

func (r *Robot) posted(s *State) bool {
	return r.hasPost() && remainingAt(s, r.PostCol, r.PostRow) > 0
}

func (r *Robot) stepPost(s *State) {
	cx, cy := tileCenterUnits(r.PostCol, r.PostRow)
	if r.walkTowards(s, cx, cy) {
		r.WorkTicks = robotLoadTicks
		r.Pile = 0
	}
}

func (r *Robot) idling(s *State) bool {
	return true
}

func (r *Robot) stepIdle(s *State) {
	px, py := parkSpot(r.ID)
	r.walkTowards(s, px, py)
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
		if tankOil(s, refuelTank(s, r)) <= 0 {
			return "out of oil"
		}
		return "refueling"
	case !at && r.Tank < robotTankLiters*robotLowTankAt:
		return "low oil"
	}
	return ""
}

// refill pours oil from the post's tank into the robot's, at the
// charger's pace; a dry post leaves the robot standing there.
func (s *State) refill(r *Robot) {
	post := refuelTank(s, *r)
	fill := math.Min(chargerRefillPerSec/60, robotTankLiters-r.Tank)
	fill = math.Min(fill, tankOil(s, post))
	s.addOil(post, -fill)
	r.Tank += fill
}

// walkTowards moves the robot one tick's step towards a point and
// reports whether it arrived. The fog drags at every robot, core or
// built, slowing it the deeper the tile sits in the mist; inside a
// bubble the step is whole, and a swell's pushed band is meaner
// (sim_fog.go).
func (r *Robot) walkTowards(s *State, x, y float64) bool {
	step := robotSpeed / 60 * fogDrag(s, r.X, r.Y)
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
// its arms, and the robot stands at the store trying again every tick.
func (s *State) deposit(r *Robot) {
	var room float64
	switch r.Cargo {
	case TypeOil:
		room = tankRoom(s, haulTank(s, *r))
	case TypeLilac:
		room = lilacCap(s) - s.Stock.Lilac
	default:
		r.Carry, r.Cargo = 0, ""
		return
	}
	put := math.Min(r.Carry, math.Max(0, room))
	switch r.Cargo {
	case TypeOil:
		s.addOil(haulTank(s, *r), put)
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
