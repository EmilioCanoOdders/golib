package main

import (
	"math"
	"sort"
)

// Pumps and pipes. Oil has a place (sim_oil.go), and a pipe moves it
// from one place to another with no robot carrying it: from a pump,
// which draws a pool, or from a tank - the core's, a silo's, a charger's
// - into another tank. Pipes make a network by meeting at buildings: each
// building takes a few, in or out. A pipe is a curve through the
// player's clicks, paid in lilac by the section and laid by the robots,
// from its source out.
const (
	pumpLitersPerSecond = 2.0 // L/s a pump draws, shared by its pipes
	pipeLitersPerSecond = 4.0 // L/s one pipe carries at the most

	pipePorts     = 3 // the pipes a building takes, in and out together
	corePipePorts = 6 // and the core

	pipeSectionMeters    = 25.0 // u, one section of pipe: a cell's side
	pipeSectionLilac     = 5.0  // kg a section
	pipeSectionWorkTicks = 60   // ticks of robot work a section: 1 s

	pipeMaxBends    = 16   // the clicks a pipe takes between its ends
	pipeBendGap     = 1.0  // u; a bend closer than this to the last one is dropped
	pipeSpanSamples = 12   // straight pieces the curve is cut into, per span
	pipeSagFactor   = 0.12 // how far a pipe with no bends sags, by its length
	pipeLayStandoff = 6.0  // u from the pipe's head to where a robot lays it
)

// PipePoint is a spot on the ground, in units.
type PipePoint struct {
	X, Y float64
}

// Pipe carries oil one way, From a pump or a tank To a tank. Its ends
// are buildings, so they are IDs - the core's tank is 0 - and only the
// bends between them are its own. Left counts the robot work still owed,
// and the pipe carries oil once it reaches zero.
type Pipe struct {
	ID       int64
	From     int64       // the pump or the tank it draws
	To       int64       // the tank it fills
	Bends    []PipePoint // the clicks between its ends
	Sections int64       // its length in whole sections: what it cost
	Left     int64       // ticks of robot work left; 0 is laid
}

func sortedPipeIDs(s *State) []int64 {
	ids := make([]int64, 0, len(s.Pipes))
	for id := range s.Pipes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// pipesOf returns the pipes that start or end at a building, or at the
// core's tank, in ID order.
func pipesOf(s *State, end int64) []Pipe {
	var pipes []Pipe
	for _, id := range sortedPipeIDs(s) {
		if p := s.Pipes[id]; p.From == end || p.To == end {
			pipes = append(pipes, p)
		}
	}
	return pipes
}

// freePorts returns how many more pipes an end takes.
func freePorts(s *State, end int64) int {
	ports := pipePorts
	if end == coreTank {
		ports = corePipePorts
	}
	return ports - len(pipesOf(s, end))
}

// unlaidPipe returns the oldest pipe the robots still have to lay.
func unlaidPipe(s *State) (Pipe, bool) {
	for _, id := range sortedPipeIDs(s) {
		if p := s.Pipes[id]; p.Left > 0 {
			return p, true
		}
	}
	return Pipe{}, false
}

// isPump reports whether an end is a raised pump.
func isPump(s *State, end int64) bool {
	b, ok := s.Buildings[end]
	return ok && b.Kind == BuildingPump
}

// pipeEndSpot returns the middle of what a pipe may start or end at: a
// pump or a tank.
func pipeEndSpot(s *State, end int64) (PipePoint, bool) {
	if isPump(s, end) {
		b := s.Buildings[end]
		x, y := cellCenterUnits(b.Col, b.Row)
		return PipePoint{x, y}, true
	}
	return tankSpot(s, end)
}

// pipeEnds lists everything a pipe may start or end at: the core's tank
// first, then the pumps and the tanks in ID order.
func pipeEnds(s *State) []int64 {
	ends := []int64{coreTank}
	for _, id := range sortedBuildingIDs(s) {
		if _, ok := pipeEndSpot(s, id); ok {
			ends = append(ends, id)
		}
	}
	return ends
}

// pipeSpine returns the curve a pipe lies along, from its source to its
// tank, and whether both ends still stand.
func pipeSpine(s *State, p Pipe) ([]PipePoint, bool) {
	from, okFrom := pipeEndSpot(s, p.From)
	to, okTo := pipeEndSpot(s, p.To)
	if !okFrom || !okTo {
		return nil, false
	}
	return pipePath(from, p.Bends, to), true
}

// pipePath draws the curve through a pipe's ends and bends: a centripetal
// Catmull-Rom spline, which passes through every click and never loops
// on itself, cut into straight pieces. A pipe with no bends sags a
// little to one side instead of running like a ruler's line.
func pipePath(from PipePoint, bends []PipePoint, to PipePoint) []PipePoint {
	knots := []PipePoint{from}
	for _, bend := range bends {
		if pointGap(knots[len(knots)-1], bend) >= pipeBendGap {
			knots = append(knots, bend)
		}
	}
	if pointGap(knots[len(knots)-1], to) < pipeBendGap {
		if len(knots) == 1 {
			return []PipePoint{from, to}
		}
		knots = knots[:len(knots)-1]
	}
	knots = append(knots, to)
	if len(knots) == 2 {
		knots = []PipePoint{from, sagPoint(from, to), to}
	}
	// The spline asks for a knot before the first and one after the last:
	// the ends' neighbors, mirrored.
	last := len(knots) - 1
	before := mirrorPoint(knots[1], knots[0])
	after := mirrorPoint(knots[last-1], knots[last])
	all := append(append([]PipePoint{before}, knots...), after)
	path := []PipePoint{from}
	for i := 1; i+2 < len(all); i++ {
		for n := 1; n <= pipeSpanSamples; n++ {
			u := float64(n) / pipeSpanSamples
			path = append(path, catmullRom(all[i-1], all[i], all[i+1], all[i+2], u))
		}
	}
	return path
}

// sagPoint is the bend a pipe with none gets: off the middle of its ends,
// always to the side that hangs down on the screen, so the curve doesn't
// flip while an end moves.
func sagPoint(from, to PipePoint) PipePoint {
	dx, dy := to.X-from.X, to.Y-from.Y
	nx, ny := -dy, dx
	if nx+ny < 0 {
		nx, ny = -nx, -ny
	}
	return PipePoint{
		X: (from.X+to.X)/2 + nx*pipeSagFactor,
		Y: (from.Y+to.Y)/2 + ny*pipeSagFactor,
	}
}

func pointGap(a, b PipePoint) float64 {
	return math.Hypot(b.X-a.X, b.Y-a.Y)
}

// mirrorPoint returns p mirrored through center.
func mirrorPoint(p, center PipePoint) PipePoint {
	return PipePoint{X: 2*center.X - p.X, Y: 2*center.Y - p.Y}
}

// lerpPoint walks from a to b by t, past either end when t says so.
func lerpPoint(a, b PipePoint, t float64) PipePoint {
	return PipePoint{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
}

// catmullRom returns the point at u, 0 to 1, of the span from p1 to p2 of
// the centripetal Catmull-Rom spline through the four knots, by Barry and
// Goldman's pyramid. The knots' parameters are spaced by the square root
// of their distances, which is what keeps the curve from overshooting
// between a short span and a long one.
func catmullRom(p0, p1, p2, p3 PipePoint, u float64) PipePoint {
	t0 := 0.0
	t1 := t0 + math.Sqrt(pointGap(p0, p1))
	t2 := t1 + math.Sqrt(pointGap(p1, p2))
	t3 := t2 + math.Sqrt(pointGap(p2, p3))
	t := t1 + (t2-t1)*u
	a1 := lerpPoint(p0, p1, (t-t0)/(t1-t0))
	a2 := lerpPoint(p1, p2, (t-t1)/(t2-t1))
	a3 := lerpPoint(p2, p3, (t-t2)/(t3-t2))
	b1 := lerpPoint(a1, a2, (t-t0)/(t2-t0))
	b2 := lerpPoint(a2, a3, (t-t1)/(t3-t1))
	return lerpPoint(b1, b2, (t-t1)/(t2-t1))
}

// pathLength returns how long a path is, in units.
func pathLength(path []PipePoint) float64 {
	length := 0.0
	for i := 1; i < len(path); i++ {
		length += pointGap(path[i-1], path[i])
	}
	return length
}

// pathPointAt returns the point a distance along a path, held at its
// ends.
func pathPointAt(path []PipePoint, along float64) PipePoint {
	if len(path) == 0 {
		return PipePoint{}
	}
	for i := 1; i < len(path); i++ {
		gap := pointGap(path[i-1], path[i])
		if along <= gap && gap > 0 {
			return lerpPoint(path[i-1], path[i], math.Max(0, along)/gap)
		}
		along -= gap
	}
	return path[len(path)-1]
}

// pipeSections returns how many whole sections a length of pipe takes.
func pipeSections(length float64) int64 {
	return int64(math.Max(1, math.Ceil(length/pipeSectionMeters)))
}

// pipeCost returns what a pipe of so many sections asks for, in lilac.
func pipeCost(sections int64) float64 {
	return float64(sections) * pipeSectionLilac
}

// pipeLaidPart returns how much of a pipe the robots have laid, 0 to 1.
func pipeLaidPart(p Pipe) float64 {
	total := float64(p.Sections * pipeSectionWorkTicks)
	if total <= 0 {
		return 1
	}
	return 1 - float64(p.Left)/total
}

// inRegion reports whether a spot, in units, stands inside the region.
func inRegion(x, y float64) bool {
	return x >= 0 && y >= 0 &&
		x < regionCols*unitsPerTile && y < regionRows*unitsPerTile
}

// canJoin reports whether a pipe may run from one end to another: a pump
// or a tank to a tank, two different ends with a port free each, and no
// pipe between the two already, either way.
func canJoin(s *State, from, to int64) bool {
	_, okFrom := pipeEndSpot(s, from)
	_, okTo := tankSpot(s, to)
	if !okFrom || !okTo || from == to {
		return false
	}
	if freePorts(s, from) <= 0 || freePorts(s, to) <= 0 {
		return false
	}
	for _, p := range pipesOf(s, from) {
		if p.From == to || p.To == to {
			return false
		}
	}
	return true
}

// canLayPipe reports whether a pipe may be marked between two ends (see
// canJoin) through bends inside the region, and the sections it would
// take.
func canLayPipe(s *State, from, to int64, bends []PipePoint) (int64, bool) {
	if !canJoin(s, from, to) || len(bends) > pipeMaxBends {
		return 0, false
	}
	for _, bend := range bends {
		if !inRegion(bend.X, bend.Y) {
			return 0, false
		}
	}
	a, _ := pipeEndSpot(s, from)
	b, _ := pipeEndSpot(s, to)
	return pipeSections(pathLength(pipePath(a, bends, b))), true
}

// workPipe puts a tick of robot work into a pipe.
func (s *State) workPipe(id int64) {
	p, ok := s.Pipes[id]
	if !ok || p.Left <= 0 {
		return
	}
	p.Left--
	s.Pipes[id] = p
}

// takePipesOf removes every pipe that starts or ends at a building and
// returns the lilac they were made of.
func (s *State) takePipesOf(building int64) float64 {
	lilac := 0.0
	for _, id := range sortedPipeIDs(s) {
		p := s.Pipes[id]
		if p.From != building && p.To != building {
			continue
		}
		lilac += pipeCost(p.Sections)
		delete(s.Pipes, id)
	}
	return lilac
}

// patchPumped reports whether a pump stands, or rises, on the deposit
// patch a tile belongs to: a pool takes one pump.
func patchPumped(s *State, tcol, trow int) bool {
	want, ok := depositAt(tcol, trow)
	if !ok {
		return false
	}
	onPatch := func(col, row int) bool {
		d, ok := depositAt(cellTile(col, row))
		return ok && d == want
	}
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		if b.Kind == BuildingPump && onPatch(b.Col, b.Row) {
			return true
		}
	}
	for _, job := range s.Jobs {
		if job.Kind == BuildingPump && onPatch(job.Col, job.Row) {
			return true
		}
	}
	return false
}

// pumpCell returns the cell a pool's pump stands on: the patch's middle.
func pumpCell(d Deposit) (col, row int) {
	const cellsPerTile = unitsPerTile / buildingCell
	return d.Col*cellsPerTile + d.Cols*cellsPerTile/2,
		d.Row*cellsPerTile + d.Rows*cellsPerTile/2
}

// The words for what a pump is doing.
const (
	pumpNoPipe  = "no pipe"
	pumpLaying  = "pipe being laid"
	pumpDry     = "pool dry"
	pumpBlocked = "pipe's end full"
	pumpPumping = "pumping"
)

// pumpStatus says what a pump is doing, in the order the rules ask: a
// pipe, laid, a pool with oil left in it, and room at a pipe's end.
func pumpStatus(s *State, b Building) string {
	pipes := pipesOf(s, b.ID)
	laid, room := false, false
	for _, p := range pipes {
		if p.Left == 0 {
			laid = true
			room = room || tankRoom(s, p.To) > 0
		}
	}
	tcol, trow := pumpTile(b)
	switch {
	case len(pipes) == 0:
		return pumpNoPipe
	case !laid:
		return pumpLaying
	case remainingAt(s, tcol, trow) <= 0:
		return pumpDry
	case !room:
		return pumpBlocked
	}
	return pumpPumping
}

func pumpTile(b Building) (tcol, trow int) {
	return cellTile(b.Col, b.Row)
}

// pipeSupply returns the oil a pipe's source can still give: what is in
// a tank, or what remains in a pump's pool.
func pipeSupply(s *State, p Pipe) float64 {
	if b, ok := s.Buildings[p.From]; ok && b.Kind == BuildingPump {
		tcol, trow := pumpTile(b)
		return remainingAt(s, tcol, trow)
	}
	return tankOil(s, p.From)
}

// pipeFlowing reports whether oil runs down a pipe: it is laid, its
// source has oil and its end has room.
func pipeFlowing(s *State, p Pipe) bool {
	return p.Left == 0 && pipeSupply(s, p) > 0 && tankRoom(s, p.To) > 0
}

// stepPipes moves a tick's worth of oil down every laid pipe, in ID
// order: as much as the pipe carries, its source gives and its end has
// room for. The pipes that leave one source share what it gives this
// tick in equal parts - a tank's oil, or what a pump draws - so a silo
// that feeds two pipes feeds both.
func stepPipes(s *State) {
	var flowing []Pipe
	outlets := map[int64]float64{}
	for _, id := range sortedPipeIDs(s) {
		if p := s.Pipes[id]; pipeFlowing(s, p) {
			flowing = append(flowing, p)
			outlets[p.From]++
		}
	}
	share := map[int64]float64{}
	for from := range outlets {
		gives := pipeSupply(s, Pipe{From: from})
		if isPump(s, from) {
			gives = math.Min(gives, pumpLitersPerSecond/60)
		}
		share[from] = gives / outlets[from]
	}
	for _, p := range flowing {
		flow := math.Min(pipeLitersPerSecond/60, share[p.From])
		flow = math.Min(flow, tankRoom(s, p.To))
		s.addOil(p.To, flow)
		if !isPump(s, p.From) {
			s.addOil(p.From, -flow)
			continue
		}
		tcol, trow := pumpTile(s.Buildings[p.From])
		d, _ := depositAt(tcol, trow)
		s.Drain[depositKey(d)] -= flow
	}
}
