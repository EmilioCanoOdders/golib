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

	// Idle robots rest by the core, lined up in ranks before its broad
	// face. Past parkSlots of them the ranks are full: the rest stand
	// inside the first ones, and the view writes how many there are.
	parkSlots    = 10   // robots the ranks show
	parkRankSize = 5    // robots to a rank
	parkSpacing  = 7.0  // u between two parked robots: one, and a bit
	parkFromCore = 10.0 // u from the core's middle to the first rank

	tankFullSlack = 0.001 // L short of the brim that still count as a full tank

	goldenAngle = 2.399963229728653 // spreads robots around what they work at
)

// stepSim moves the world one tick forward: the weather, the
// factories, the pipes, the rivals and the guard posts, then the robots
// in ID order, so the outcome never depends on map iteration. A built robot
// empty of oil outside every bubble is digested by the fog and leaves a
// quarter of each resource in a wreck: its cost and onboard resources.
func stepSim(s *State) {
	stepTech(s)
	stepFog(s)
	stepFactories(s)
	stepPipes(s)
	stepSquads(s)
	stepEnemies(s)
	stepGuards(s)
	stepArtillery(s)
	stepShots(s)
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		// A trooper the rivals shot this tick is gone already.
		if _, alive := s.Robots[id]; !alive {
			continue
		}
		stepRobot(s, &r)
		if r.tanked() && r.Tank <= 0 && !inSafeZone(s, r.X, r.Y) {
			delete(s.Robots, id)
			s.dropRobotWreck(r)
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
	taskBuild   = "build"   // raise the oldest build job, then lay its section of pipe
	taskCollect = "collect" // pick up loose items
	taskPost    = "post"    // work its own post
	taskSquad   = "squad"   // troopers: follow the squad's order
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
var robotDay []robotTask

// Idling asks what the other robots are at, which reads the day back:
// filling it here keeps Go from seeing an initialization cycle.
func init() {
	robotDay = []robotTask{
		{taskHaul, (*Robot).hauling, (*Robot).stepHaul},
		{taskRefuel, (*Robot).refueling, (*Robot).stepRefuel},
		{taskLoad, (*Robot).loading, (*Robot).stepLoad},
		{taskBuild, (*Robot).building, (*Robot).stepBuild},
		{taskCollect, (*Robot).collecting, (*Robot).stepCollect},
		{taskPost, (*Robot).posted, (*Robot).stepPost},
		{taskSquad, (*Robot).squadded, (*Robot).stepSquad},
		{taskIdle, (*Robot).idling, (*Robot).stepIdle},
	}
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
// The one thing it remembers is the section of pipe it claimed, so the
// others take another, and only while it builds.
func stepRobot(s *State, r *Robot) {
	// A post with nothing left to give holds nobody: it ran dry under
	// another robot's hands, or an old save names ground the region has
	// since lost.
	if r.hasPost() && remainingAt(s, r.PostCol, r.PostRow) <= 0 {
		r.clearPost()
	}
	if r.Kind == RobotCombat {
		r.shoot(s)
	}
	// The tank pays for carrying: an empty-handed robot burns nothing.
	if r.tanked() && r.Tank > 0 && r.Carry > 0 {
		burn := robotBurnPerSecond / 60
		if !inSafeZone(s, r.X, r.Y) {
			// The swell's mist is meaner in the open, by as much as it presses.
			burn *= 1 + (fogSwellBurn-1)*s.Fog.Pressure
		}
		r.Tank = math.Max(0, r.Tank-burn)
	}
	task := r.taskNow(s)
	if task.name != taskBuild {
		r.Pipe, r.Section = 0, 0
	}
	task.step(r, s)
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
	if r.Kind == RobotCombat {
		return false
	}
	if _, hasJob := oldestJob(s); hasJob {
		return true
	}
	if _, damaged := damagedBuilding(s); damaged {
		return true
	}
	if claimStands(s, *r) {
		return true
	}
	_, _, free := freeSection(s, *r)
	return free
}

// Builders stand on their cell's edge, spread by ID, so the rising body
// doesn't swallow them. With no site left to raise, they lay pipe.
func (r *Robot) stepBuild(s *State) {
	job, hasJob := oldestJob(s)
	if b, damaged := damagedBuilding(s); !hasJob && damaged {
		r.Pipe, r.Section = 0, 0
		cx, cy := cellCenterUnits(b.Col, b.Row)
		angle := float64(r.ID) * goldenAngle
		if r.walkTowards(s, cx+math.Cos(angle)*11, cy+math.Sin(angle)*11) {
			s.mend(b.ID)
		}
		return
	}
	if !hasJob {
		r.stepLayPipe(s)
		return
	}
	r.Pipe, r.Section = 0, 0
	cx, cy := cellCenterUnits(job.Col, job.Row)
	angle := float64(r.ID) * goldenAngle
	if r.walkTowards(s, cx+math.Cos(angle)*11, cy+math.Sin(angle)*11) {
		s.workJob()
	}
}

// stepLayPipe lays one section of pipe: the robot claims the next one
// nobody has, walks to it and stands by it until it is laid, then claims
// another. The claim is in the state, so every robot takes its own
// section and a pipe is laid by as many hands as there are.
func (r *Robot) stepLayPipe(s *State) {
	if !claimStands(s, *r) {
		pipe, section, free := freeSection(s, *r)
		if !free {
			r.Pipe, r.Section = 0, 0
			return
		}
		r.Pipe, r.Section = pipe, section
	}
	path, ok := pipeSpine(s, s.Pipes[r.Pipe])
	if !ok {
		r.Pipe, r.Section = 0, 0
		return
	}
	spot := sectionSpot(path, r.Section)
	angle := float64(r.ID) * goldenAngle
	x := spot.X + math.Cos(angle)*pipeLayStandoff
	y := spot.Y + math.Sin(angle)*pipeLayStandoff
	if r.walkTowards(s, x, y) {
		s.workSection(r.Pipe, r.Section)
		if !claimStands(s, *r) {
			r.Pipe, r.Section = 0, 0
		}
	}
}

func (r *Robot) collecting(s *State) bool {
	if r.Kind == RobotCombat {
		return false
	}
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
	cx, cy := postSpot(r.PostCol, r.PostRow)
	if r.walkTowards(s, cx, cy) {
		r.WorkTicks = robotLoadTicks
		r.Pile = 0
	}
}

// postSpot returns where a robot loads at its post: by the heart of the
// tile's deposit, where the ore is richest, a cell to the side the pump
// leaves free. Whatever tile of the deposit the robot was sent to, the
// deposit is one thing and is worked at one place.
func postSpot(col, row int) (x, y float64) {
	d, ok := depositAt(col, row)
	if !ok {
		return tileCenterUnits(col, row)
	}
	return cellCenterUnits(d.HeartCol+1, d.HeartRow)
}

func (r *Robot) idling(s *State) bool {
	return true
}

func (r *Robot) stepIdle(s *State) {
	px, py := parkSlot(idleRank(s, *r))
	r.walkTowards(s, px, py)
}

// idleRank returns how many idle robots come before this one, by ID:
// its place in the ranks. When one leaves, the ranks close up.
func idleRank(s *State, r Robot) int {
	rank := 0
	for _, id := range sortedRobotIDs(s) {
		if id >= r.ID {
			break
		}
		other := s.Robots[id]
		if other.taskNow(s).name == taskIdle {
			rank++
		}
	}
	return rank
}

// idleRobots returns how many robots have nothing to do.
func idleRobots(s *State) int {
	idle := 0
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		if r.taskNow(s).name == taskIdle {
			idle++
		}
	}
	return idle
}

// refueling reports whether the tank owns the robot's day: low, it
// claims it and walks to the nearest refill post; there it stands until
// the tank is full, even past the low line, so it doesn't dance between
// post and work. A dry post holds nobody with oil left in its tank: the
// way to fill the posts again is to go and fetch oil.
func (r *Robot) refueling(s *State) bool {
	if !r.tanked() || r.Tank >= robotTankLiters-tankFullSlack {
		return false
	}
	if r.Tank > 0 && tankOil(s, refuelTank(s, *r)) <= 0 {
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
	if !r.tanked() {
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
	if put > 0 {
		s.Deliveries++
	}
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

// parkSlot returns where the idle robot of a rank rests: in ranks of
// parkRankSize before the core's broad face, which looks down the
// world's x axis, centered on it. Past parkSlots the ranks start over.
func parkSlot(rank int) (x, y float64) {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	slot := rank % parkSlots
	line, place := slot/parkRankSize, slot%parkRankSize
	return cx + parkFromCore + float64(line)*parkSpacing,
		cy + (float64(place)-float64(parkRankSize-1)/2)*parkSpacing
}

// parkCenter returns the middle of the ranks.
func parkCenter() (x, y float64) {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	lines := float64(parkSlots/parkRankSize - 1)
	return cx + parkFromCore + lines*parkSpacing/2, cy
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
		if r.Kind == RobotCombat {
			continue
		}
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
