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
