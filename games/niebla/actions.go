package main

// Actions in, state out: nothing but Apply's reducers touch the state.
// An action is a serializable struct, and Apply is total — any action on
// any state gives a well-defined result — and deterministic, so a seed
// plus a log of actions replays a game, which is also the shape of the
// later multiplayer server. Apply mutates the state it is given: the
// state has one owner (the scene, the tests, a later server), and
// copying it per action would buy nothing.

// Action is one order for the simulation.
type Action interface {
	apply(s *State)
}

// Apply applies an action to the state, in place.
func Apply(s *State, a Action) {
	a.apply(s)
}

// Tick is the action the game loop sends 60 times per second of game
// time: one step of the simulation.
type Tick struct{}

func (Tick) apply(s *State) {
	s.Ticks++
	stepSim(s)
}

// SendRobot sends a robot to work a deposit tile: the tile becomes its
// post. The nearest free robot takes it; when every robot already has a
// post, the nearest one is retasked. It does nothing for a tile that
// holds no deposit, a dry one, or one that has its robot already.
type SendRobot struct {
	Col int
	Row int
}

func (a SendRobot) apply(s *State) {
	kind := tileAt(a.Col, a.Row)
	if kind != kindOil && kind != kindLilac {
		return
	}
	if remainingAt(s, a.Col, a.Row) <= 0 {
		return
	}
	if _, owned := postOwner(s, a.Col, a.Row); owned {
		return
	}
	id := pickRobot(s, a.Col, a.Row)
	if id < 0 {
		return
	}
	r := s.Robots[id]
	r.PostCol, r.PostRow = a.Col, a.Row
	s.Robots[id] = r
}

// RecallRobot takes a tile's post away from its robot, which finishes
// any carry it holds and then idles by the core.
type RecallRobot struct {
	Col int
	Row int
}

func (a RecallRobot) apply(s *State) {
	if r, ok := postOwner(s, a.Col, a.Row); ok {
		r.clearPost()
		s.Robots[r.ID] = r
	}
}

// MarkBuilding marks a cell for a building: the blueprint's cost is paid
// from the stores at once, and the job joins the queue for the robots to
// raise. It does nothing on a cell that can't take the kind (see
// canPlace) or when the stores can't pay.
type MarkBuilding struct {
	Kind    BuildingKind
	Col, Row int // the cell, the tile grid's last subdivision
}

func (a MarkBuilding) apply(s *State) {
	if !canPlace(s, a.Kind, a.Col, a.Row) {
		return
	}
	lilac, oil := buildingCost(a.Kind)
	if s.Stock.Lilac < lilac || s.Stock.Oil < oil {
		return
	}
	s.Stock.Lilac -= lilac
	s.Stock.Oil -= oil
	s.Jobs = append(s.Jobs, Job{
		Kind: a.Kind, Col: a.Col, Row: a.Row, Left: buildingWorkTicks,
	})
}

// QueueRobot puts a factory to work on one more robot, paying lilac and
// oil for it at once. A factory already building, or stores that can't
// pay, leave it as it is.
type QueueRobot struct {
	Building int64 // the factory's entity ID
}

func (a QueueRobot) apply(s *State) {
	b, ok := s.Buildings[a.Building]
	if !ok || b.Kind != BuildingFactory || b.Work > 0 {
		return
	}
	if s.Stock.Lilac < robotCostLilac || s.Stock.Oil < robotCostOil {
		return
	}
	s.Stock.Lilac -= robotCostLilac
	s.Stock.Oil -= robotCostOil
	b.Work = factoryRobotTicks
	s.Buildings[b.ID] = b
}

// Demolish takes a building down at once: its tasks die with it, and its
// cost, the cost of a robot its factory was building and what the stores
// lose the roof for fall on its cell as one pile. It does nothing for a
// building that isn't there, or for a protector that alone shelters
// another building (see canDemolish).
type Demolish struct {
	Building int64 // the building's entity ID
}

func (a Demolish) apply(s *State) {
	b, ok := s.Buildings[a.Building]
	if !ok || !canDemolish(s, b) {
		return
	}
	delete(s.Buildings, b.ID)
	lilac, oil := buildingCost(b.Kind)
	lilac *= demolishRefund
	oil *= demolishRefund
	if b.Kind == BuildingFactory && b.Work > 0 {
		lilac += robotCostLilac
		oil += robotCostOil
	}
	s.dropPile(b.Col, b.Row, oil, lilac)
	s.spillOverflow(b.Col, b.Row)
}

// CancelJob takes a site out of the queue: the work put into it is gone
// and its cost falls on its cell as a pile. It does nothing on a cell
// with no site.
type CancelJob struct {
	Col, Row int // the site's cell
}

func (a CancelJob) apply(s *State) {
	for i, job := range s.Jobs {
		if job.Col != a.Col || job.Row != a.Row {
			continue
		}
		s.Jobs = append(s.Jobs[:i:i], s.Jobs[i+1:]...)
		lilac, oil := buildingCost(job.Kind)
		s.dropPile(job.Col, job.Row, oil*demolishRefund, lilac*demolishRefund)
		return
	}
}

// The developer's actions, sent by the dev tools (dev.go) and by nobody
// else. They are actions like any other, so a log with them in it still
// replays.

// DevHoldSwell raises a swell and holds it up, or lets go of the held one
// and ends it. A held swell is no swell of the fog's own: it doesn't
// count, so the next ones come no sooner, longer or deeper for it.
type DevHoldSwell struct {
	On bool
}

func (a DevHoldSwell) apply(s *State) {
	s.Fog.Held = a.On
	if !a.On {
		s.Fog.SwellLeft = 0
		return
	}
	if s.Fog.SwellLeft <= 0 {
		s.Fog.SwellLeft = swellTicks(s)
	}
}

// DevSpawnRobot puts a built robot, its tank full, on a spot of the
// region, free of charge. It does nothing for a spot outside the region.
type DevSpawnRobot struct {
	X, Y float64 // units
}

func (a DevSpawnRobot) apply(s *State) {
	if a.X < 0 || a.Y < 0 ||
		a.X >= regionCols*unitsPerTile || a.Y >= regionRows*unitsPerTile {
		return
	}
	s.spawnRobot(RobotBuilt, a.X, a.Y)
}
