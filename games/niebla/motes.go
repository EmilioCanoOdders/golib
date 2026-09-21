package main

import (
	"fmt"
	"math"
	"sort"

	"golib"
)

// The fog's wear, made visible: motes of darkness orbit whatever stands
// in the mist - robots, building sites, piles - a few per cubic unit of
// its body. They chase their orbit with a lag, so a walker leaves them
// trailing behind, never quite caught; what stands still they close in
// on. They are view, never state: nothing in the simulation reads them,
// and their randomness is golib's, for looks only.

// Tuning: the motes, with units in the name.
const (
	motesPerCubicUnit = 0.04 // motes per u3 of body, in full fog
	motesMaxPerHost   = 400
	motesBornPerTick  = 3 // how fast a swarm gathers around a newcomer

	moteLagSeconds   = 0.6  // how late a mote follows its orbit
	moteGripSeconds  = 2.0  // standing still this long, the swarm has closed in
	moteLooseSeconds = 0.35 // walking this long, it has let go
	moteFadeSeconds  = 0.5  // a mote's fade in and out

	moteWideOrbit  = 3.0 // orbit radius around a walker, in body radii
	moteTightOrbit = 1.0 // and around what stands still

	moteTurnsPerSecond = 0.25 // a mote's mean turning rate around its host
	moteSizeUnits      = 0.3  // the dark center's side; never under a pixel

	moteRings       = 6   // the heart and the rings of halo around it
	moteHaloOpacity = 0.2 // the halo's darkness right beside the heart

	robotBodyAcross = 6.0 // u, the robot's body for the motes' count
	robotBodyHeight = 5.0
)

// moteFalloff returns a mote's darkness from its center out, one entry
// per ring a heart's side wide: a spark turned inside out, black only at
// the heart, then a halo that starts at moteHaloOpacity and eases down
// to nothing at the rim.
func moteFalloff() []float32 {
	falloff := make([]float32, moteRings)
	falloff[0] = 1
	for ring := 1; ring < moteRings; ring++ {
		left := 1 - float32(ring-1)/float32(moteRings-1)
		falloff[ring] = moteHaloOpacity * left * left
	}
	return falloff
}

type mote struct {
	X, Y, Z float32 // world units; Z is height over the ground
	Angle   float32 // where on its orbit, in radians
	Turn    float32 // radians per second, signed
	Reach   float32 // its own share of the swarm's orbit radius
	Bob     float32 // the phase of its rise and fall
	Life    float32 // 0 to 1, its fade
}

// moteHost is something the motes orbit. It outlives its thing by the
// motes' fade, so a digested robot's swarm closes on the empty spot.
type moteHost struct {
	X, Y           float32 // where it stood at the last update
	Across, Height float32 // its body, in units
	Robot          bool    // drawn under the robots' icon law, not the buildings'
	Grip           float32 // 0 walking, 1 stood still long enough
	Wanted         int     // motes its body and the fog around it call for
	Motes          []mote
}

type moteField struct {
	hosts map[string]*moteHost
}

func newMoteField() *moteField {
	return &moteField{hosts: map[string]*moteHost{}}
}

func robotMoteKey(id int64) string {
	return fmt.Sprintf("robot:%d", id)
}

func siteMoteKey(col, row int) string {
	return fmt.Sprintf("site:%d,%d", col, row)
}

// update moves the motes one step. It reads the state and never changes
// it. It draws random numbers, so it belongs in Update, never in Draw.
func (f *moteField) update(s *State, dt float32) {
	for _, h := range f.hosts {
		h.Wanted = 0
	}
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		h := f.host(robotMoteKey(id), r.X, r.Y, dt)
		h.Across, h.Height, h.Robot = robotBodyAcross, robotBodyHeight, true
		h.want(s)
	}
	for _, job := range s.Jobs {
		x, y := cellCenterUnits(job.Col, job.Row)
		h := f.host(siteMoteKey(job.Col, job.Row), x, y, dt)
		h.Across, h.Height = buildingSize(job.Kind)
		h.want(s)
	}
	for _, id := range sortedPileIDs(s) {
		p := s.Piles[id]
		x, y := cellCenterUnits(p.Col, p.Row)
		h := f.host(fmt.Sprintf("pile:%d", id), x, y, dt)
		h.Across, h.Height = 14, 6
		h.want(s)
	}
	keys := make([]string, 0, len(f.hosts))
	for key := range f.hosts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		h := f.hosts[key]
		h.step(dt)
		if h.Wanted == 0 && len(h.Motes) == 0 {
			delete(f.hosts, key)
		}
	}
}

// host finds or makes the host under a key, and works out its grip from
// whether it moved since the last update.
func (f *moteField) host(key string, x, y float64, dt float32) *moteHost {
	h, known := f.hosts[key]
	if !known {
		h = &moteHost{X: float32(x), Y: float32(y)}
		f.hosts[key] = h
	}
	moved := math.Hypot(x-float64(h.X), y-float64(h.Y)) > 1e-4
	if moved {
		h.Grip -= dt / moteLooseSeconds
	} else {
		h.Grip += dt / moteGripSeconds
	}
	h.Grip = golib.Clamp(h.Grip, 0, 1)
	h.X, h.Y = float32(x), float32(y)
	return h
}

// want counts the motes a host calls for: its body's volume, thinned by
// how much fog stands on it. Under a bubble it calls for none.
func (h *moteHost) want(s *State) {
	fog := fogAt(s, float64(h.X), float64(h.Y))
	volume := float64(h.Across * h.Across * h.Height)
	h.Wanted = int(volume * motesPerCubicUnit * fog)
	if h.Wanted > motesMaxPerHost {
		h.Wanted = motesMaxPerHost
	}
}

func (h *moteHost) step(dt float32) {
	radius := h.Across / 2
	for born := 0; born < motesBornPerTick && len(h.Motes) < h.Wanted; born++ {
		h.Motes = append(h.Motes, h.newMote(radius))
	}
	grip := golib.EaseInOut(h.Grip)
	orbit := radius * golib.Lerp(moteWideOrbit, moteTightOrbit, grip)
	follow := 1 - float32(math.Exp(float64(-dt/moteLagSeconds)))
	fade := dt / moteFadeSeconds
	for i := range h.Motes {
		m := &h.Motes[i]
		if i < h.Wanted {
			m.Life += fade
		} else {
			m.Life -= fade
		}
		m.Life = golib.Clamp(m.Life, 0, 1)
		m.Angle += m.Turn * (1 + grip) * dt
		m.Bob += m.Turn * 0.7 * dt
		reach := float64(orbit * m.Reach)
		tx := h.X + float32(math.Cos(float64(m.Angle))*reach)
		ty := h.Y + float32(math.Sin(float64(m.Angle))*reach)
		tz := h.Height * (0.5 + 0.5*float32(math.Sin(float64(m.Bob))))
		m.X += (tx - m.X) * follow
		m.Y += (ty - m.Y) * follow
		m.Z += (tz - m.Z) * follow
	}
	for n := len(h.Motes); n > h.Wanted && h.Motes[n-1].Life <= 0; n-- {
		h.Motes = h.Motes[:n-1]
	}
}

// newMote makes a mote out past the widest orbit, so it swoops in.
func (h *moteHost) newMote(radius float32) mote {
	angle := golib.RandomFloat(0, 2*math.Pi)
	turn := golib.RandomFloat(0.5, 1.5) * moteTurnsPerSecond * 2 * math.Pi
	if golib.RandomFloat(0, 1) < 0.5 {
		turn = -turn
	}
	far := float64(radius * moteWideOrbit * 2)
	return mote{
		X:     h.X + float32(math.Cos(float64(angle))*far),
		Y:     h.Y + float32(math.Sin(float64(angle))*far),
		Z:     golib.RandomFloat(0, h.Height),
		Angle: angle,
		Turn:  turn,
		Reach: golib.RandomFloat(0.6, 1.4),
		Bob:   golib.RandomFloat(0, 2*math.Pi),
	}
}

// lift returns the factor the host's drawing grows by while the view is
// far out, the icon law of its kind, so the swarm rings the icon and not
// the speck the body really is.
func (h *moteHost) lift(zoom float32) float32 {
	if h.Robot {
		return dotRadius(h.Across/2, zoom, 3) / (h.Across / 2 * unitW)
	}
	return buildingIcon(h.Across, h.Height, zoom)
}

// moteLayers returns the opacity to draw each ring's disc with, from the
// center out, so that stacked from the outside in they show falloff.
func moteLayers(falloff []float32) []float32 {
	layers := make([]float32, len(falloff))
	under := float32(0)
	for i := len(falloff) - 1; i >= 0; i-- {
		layers[i] = 1 - (1-falloff[i])/(1-under)
		under = falloff[i]
	}
	return layers
}

// draw paints the motes over the fog. Black under normal blending takes
// light away from what is under it, and taking away commutes, so the
// motes need no order.
func (f *moteField) draw(screen *golib.Screen, zoom float32) {
	layers := moteLayers(moteFalloff())
	side := dotRadius(moteSizeUnits/2, zoom, 0.5) * 2
	for _, h := range f.hosts {
		k := h.lift(zoom)
		hx, hy := project(h.X, h.Y)
		// A lifted icon has room for fewer motes than the body it
		// stands for.
		shown := int(float32(len(h.Motes)) * golib.Clamp(1/k, 0.2, 1))
		for _, m := range h.Motes[:shown] {
			mx, my := project(m.X, m.Y)
			x := hx + (mx-hx)*k
			y := hy + (my-hy)*k - m.Z*unitH*k
			for ring := len(layers) - 1; ring > 0; ring-- {
				screen.DrawCircle(x, y, side*(float32(ring)+0.5),
					golib.WithOpacity(golib.Black, layers[ring]*m.Life))
			}
			// The heart stays a square: a disc half a pixel wide can fall
			// between pixel centers and draw nothing.
			screen.DrawRectangle(
				golib.Rectangle{X: x - side/2, Y: y - side/2, Width: side, Height: side},
				golib.WithOpacity(golib.Black, layers[0]*m.Life),
			)
		}
	}
}
