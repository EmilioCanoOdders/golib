package main

import (
	"fmt"
	"math"
	"sort"

	"golib"
)

// The fog's wear, made visible: mites of darkness orbit whatever stands
// in the mist - robots, building sites, piles - a few per cubic unit of
// its body. They chase their orbit with a lag, so a walker leaves them
// trailing behind, never quite caught; what stands still they close in
// on. They are view, never state: nothing in the simulation reads them,
// and their randomness is golib's, for looks only.

// Tuning: the mites, with units in the name.
const (
	mitesPerCubicUnit = 0.04 // mites per u3 of body, in full fog
	mitesMaxPerHost   = 400
	mitesBornPerTick  = 3 // how fast a swarm gathers around a newcomer

	miteLagSeconds   = 0.6  // how late a mite follows its orbit
	miteGripSeconds  = 2.0  // standing still this long, the swarm has closed in
	miteLooseSeconds = 0.35 // walking this long, it has let go
	miteFadeSeconds  = 0.5  // a mite's fade in and out

	miteWideOrbit  = 3.0 // orbit radius around a walker, in body radii
	miteTightOrbit = 1.0 // and around what stands still

	miteTurnsPerSecond = 0.25 // a mite's mean turning rate around its host
	miteSizeUnits      = 0.3  // the dark center's side; never under a pixel

	miteRings       = 6   // the heart and the rings of halo around it
	miteHaloOpacity = 0.2 // the halo's darkness right beside the heart

	robotBodyAcross = 6.0 // u, the robot's body for the mites' count
	robotBodyHeight = 5.0
)

// miteFalloff returns a mite's darkness from its center out, one entry
// per ring a heart's side wide: a spark turned inside out, black only at
// the heart, then a halo that starts at miteHaloOpacity and eases down
// to nothing at the rim.
func miteFalloff() []float32 {
	falloff := make([]float32, miteRings)
	falloff[0] = 1
	for ring := 1; ring < miteRings; ring++ {
		left := 1 - float32(ring-1)/float32(miteRings-1)
		falloff[ring] = miteHaloOpacity * left * left
	}
	return falloff
}

type mite struct {
	X, Y, Z float32 // world units; Z is height over the ground
	Angle   float32 // where on its orbit, in radians
	Turn    float32 // radians per second, signed
	Reach   float32 // its own share of the swarm's orbit radius
	Bob     float32 // the phase of its rise and fall
	Life    float32 // 0 to 1, its fade
}

// miteHost is something the mites orbit. It outlives its thing by the
// mites' fade, so a digested robot's swarm closes on the empty spot.
type miteHost struct {
	X, Y           float32 // where it stood at the last update
	Across, Height float32 // its body, in units
	Robot          bool    // drawn under the robots' icon law, not the buildings'
	Grip           float32 // 0 walking, 1 stood still long enough
	Wanted         int     // mites its body and the fog around it call for
	Mites          []mite
}

type miteField struct {
	hosts map[string]*miteHost
}

func newMiteField() *miteField {
	return &miteField{hosts: map[string]*miteHost{}}
}

func robotMiteKey(id int64) string {
	return fmt.Sprintf("robot:%d", id)
}

func siteMiteKey(col, row int) string {
	return fmt.Sprintf("site:%d,%d", col, row)
}

// update moves the mites one step. It reads the state and never changes
// it. It draws random numbers, so it belongs in Update, never in Draw.
func (f *miteField) update(s *State, dt float32) {
	for _, h := range f.hosts {
		h.Wanted = 0
	}
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		h := f.host(robotMiteKey(id), r.X, r.Y, dt)
		h.Across, h.Height, h.Robot = robotBodyAcross, robotBodyHeight, true
		h.want(s)
	}
	for _, job := range s.Jobs {
		x, y := cellCenterUnits(job.Col, job.Row)
		h := f.host(siteMiteKey(job.Col, job.Row), x, y, dt)
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
		if h.Wanted == 0 && len(h.Mites) == 0 {
			delete(f.hosts, key)
		}
	}
}

// host finds or makes the host under a key, and works out its grip from
// whether it moved since the last update.
func (f *miteField) host(key string, x, y float64, dt float32) *miteHost {
	h, known := f.hosts[key]
	if !known {
		h = &miteHost{X: float32(x), Y: float32(y)}
		f.hosts[key] = h
	}
	moved := math.Hypot(x-float64(h.X), y-float64(h.Y)) > 1e-4
	if moved {
		h.Grip -= dt / miteLooseSeconds
	} else {
		h.Grip += dt / miteGripSeconds
	}
	h.Grip = golib.Clamp(h.Grip, 0, 1)
	h.X, h.Y = float32(x), float32(y)
	return h
}

// want counts the mites a host calls for: its body's volume, thinned by
// how much fog stands on it. Under a bubble it calls for none.
func (h *miteHost) want(s *State) {
	fog := fogAt(s, float64(h.X), float64(h.Y))
	volume := float64(h.Across * h.Across * h.Height)
	h.Wanted = int(volume * mitesPerCubicUnit * fog)
	if h.Wanted > mitesMaxPerHost {
		h.Wanted = mitesMaxPerHost
	}
}

func (h *miteHost) step(dt float32) {
	radius := h.Across / 2
	for born := 0; born < mitesBornPerTick && len(h.Mites) < h.Wanted; born++ {
		h.Mites = append(h.Mites, h.newMite(radius))
	}
	grip := golib.EaseInOut(h.Grip)
	orbit := radius * golib.Lerp(miteWideOrbit, miteTightOrbit, grip)
	follow := 1 - float32(math.Exp(float64(-dt/miteLagSeconds)))
	fade := dt / miteFadeSeconds
	for i := range h.Mites {
		m := &h.Mites[i]
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
	for n := len(h.Mites); n > h.Wanted && h.Mites[n-1].Life <= 0; n-- {
		h.Mites = h.Mites[:n-1]
	}
}

// newMite makes a mite out past the widest orbit, so it swoops in.
func (h *miteHost) newMite(radius float32) mite {
	angle := golib.RandomFloat(0, 2*math.Pi)
	turn := golib.RandomFloat(0.5, 1.5) * miteTurnsPerSecond * 2 * math.Pi
	if golib.RandomFloat(0, 1) < 0.5 {
		turn = -turn
	}
	far := float64(radius * miteWideOrbit * 2)
	return mite{
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
func (h *miteHost) lift(zoom float32) float32 {
	if h.Robot {
		return dotRadius(h.Across/2, zoom, 3) / (h.Across / 2 * unitW)
	}
	return buildingIcon(h.Across, h.Height, zoom)
}

// miteLayers returns the opacity to draw each ring's disc with, from the
// center out, so that stacked from the outside in they show falloff.
func miteLayers(falloff []float32) []float32 {
	layers := make([]float32, len(falloff))
	under := float32(0)
	for i := len(falloff) - 1; i >= 0; i-- {
		layers[i] = 1 - (1-falloff[i])/(1-under)
		under = falloff[i]
	}
	return layers
}

// draw paints the mites over the fog. Black under normal blending takes
// light away from what is under it, and taking away commutes, so the
// mites need no order.
func (f *miteField) draw(screen *golib.Screen, zoom float32) {
	layers := miteLayers(miteFalloff())
	side := dotRadius(miteSizeUnits/2, zoom, 0.5) * 2
	for _, h := range f.hosts {
		k := h.lift(zoom)
		hx, hy := project(h.X, h.Y)
		// A lifted icon has room for fewer mites than the body it
		// stands for.
		shown := int(float32(len(h.Mites)) * golib.Clamp(1/k, 0.2, 1))
		for _, m := range h.Mites[:shown] {
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
