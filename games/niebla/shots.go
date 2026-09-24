package main

import (
	"math"

	"golib"
)

// Shots on the screen, and what they do to the light: bullets are short
// streaks, shells climb and fall along an arc over their shadow, both
// carry a pool of light that slides over the ground and whatever stands
// on it, guns flash, and what lands bursts into sparks, embers and smoke.
// All of it is view: the play scene owns the field, the state never
// hears of it, and its randomness is golib's. The field learns what
// happened by comparing the state's shots with the ones it saw last: a
// new one was fired, a missing one has landed.

const (
	fxGravity    = 320.0 // u/s2 on a spark
	fxMaxSparks  = 1600
	fxLightRings = 14 // ellipses a pool of light is stacked from
	fxSparkRings = 3  // and a spark's own little pool

	shellArc              = 0.3 // fraction of ground range at the arc's top
	shellTrailStepUnits   = 9.0
	shellTrailLengthUnits = 54.0
	shellSmokeStepUnits   = 30.0
	shellSmokeOpacity     = 0.14
)

var (
	sparkWhite  = golib.Color{R: 255, G: 250, B: 220, A: 255}
	sparkYellow = golib.Color{R: 255, G: 214, B: 92, A: 255}
	sparkOrange = golib.Color{R: 255, G: 146, B: 48, A: 255}
	sparkRed    = golib.Color{R: 232, G: 62, B: 40, A: 255}
	smokeColor  = golib.Color{R: 30, G: 28, B: 30, A: 255}
	shellLight  = golib.Color{R: 255, G: 170, B: 80, A: 255}
	bulletLight = golib.Color{R: 255, G: 236, B: 170, A: 255}
	rivalLight  = golib.Color{R: 255, G: 120, B: 90, A: 255}
)

// spark is one particle: a point over the ground, in units, with its
// height, flying and falling. Smoke is a spark that rises and darkens
// instead of shining.
type spark struct {
	x, y, z    float64
	vx, vy, vz float64
	age, life  float32
	size       float32 // u
	opacity    float32
	color      golib.Color
	smoke      bool
}

// flash is a pool of light that fades where it was lit, and a ring that
// spreads from it when it is a burst's.
type flash struct {
	x, y      float64
	reach     float64 // u
	age, life float32
	color     golib.Color
	ring      bool
}

type fxField struct {
	known         map[int64]Shot
	smokeDistance map[int64]float64
	sparks        []spark
	flashes       []flash
}

func newFxField() *fxField {
	return &fxField{
		known:         map[int64]Shot{},
		smokeDistance: map[int64]float64{},
	}
}

func spread(low, high float64) float64 {
	return float64(golib.RandomFloat(float32(low), float32(high)))
}

// burst throws n sparks out of a spot, up and around, in the fire's
// colors, the hottest the fastest.
func (f *fxField) burst(x, y, z float64, n int, speed, lift float64, life float32) {
	colors := []golib.Color{sparkYellow, sparkYellow, sparkOrange, sparkOrange, sparkRed, sparkRed}
	for i := 0; i < n && len(f.sparks) < fxMaxSparks; i++ {
		angle := spread(0, 2*math.Pi)
		push := spread(0.25, 1)
		f.sparks = append(f.sparks, spark{
			x: x, y: y, z: z,
			vx: math.Cos(angle) * speed * push, vy: math.Sin(angle) * speed * push,
			vz:    spread(0.3, 1) * lift,
			life:  life * float32(spread(0.5, 1)),
			size:  float32(spread(0.7, 1.6)),
			color: colors[int((1-push)/0.75*float64(len(colors)-1))],
		})
	}
}

func (f *fxField) smoke(x, y, z float64, n int, reach float64) {
	for i := 0; i < n && len(f.sparks) < fxMaxSparks; i++ {
		f.sparks = append(f.sparks, spark{
			x: x + spread(-reach, reach), y: y + spread(-reach, reach), z: z,
			vx: spread(-6, 6), vy: spread(-6, 6), vz: spread(10, 28),
			life: float32(spread(1.2, 2.6)),
			size: float32(spread(4, 9)), opacity: 0.32,
			color: smokeColor, smoke: true,
		})
	}
}

func (f *fxField) shellSmoke(x, y, z float64) {
	if len(f.sparks) >= fxMaxSparks {
		return
	}
	f.sparks = append(f.sparks, spark{
		x: x + spread(-2, 2), y: y + spread(-2, 2), z: z,
		vx: spread(-2, 2), vy: spread(-2, 2), vz: spread(8, 16),
		life: float32(spread(0.7, 1.1)),
		size: float32(spread(2.5, 4.5)), opacity: shellSmokeOpacity,
		color: smokeColor, smoke: true,
	})
}

// update compares the state's shots with the ones seen last, lights what
// was fired and bursts what landed, then moves every particle a frame.
func (f *fxField) update(s *State, dt float32) {
	for id, shot := range s.Shots {
		if _, old := f.known[id]; old {
			continue
		}
		color := bulletLight
		if shot.Rival {
			color = rivalLight
		}
		if shot.Kind == ShotShell {
			f.flashes = append(f.flashes, flash{
				x: shot.FromX, y: shot.FromY, reach: 90, life: 0.22, color: shellLight,
			})
			f.burst(shot.FromX, shot.FromY, 8, 8, 60, 90, 0.4)
			f.smoke(shot.FromX, shot.FromY, 8, 4, 6)
			continue
		}
		f.flashes = append(f.flashes, flash{
			x: shot.FromX, y: shot.FromY, reach: 22, life: 0.07, color: color,
		})
	}
	f.updateShellSmoke(s)
	for id, shot := range f.known {
		if _, flying := s.Shots[id]; flying {
			continue
		}
		if shot.Kind == ShotShell {
			f.flashes = append(f.flashes, flash{
				x: shot.ToX, y: shot.ToY, reach: 190, life: 0.45,
				color: shellLight, ring: true,
			})
			f.burst(shot.ToX, shot.ToY, 1, 90, 170, 230, 1.5)
			f.smoke(shot.ToX, shot.ToY, 2, 14, shellBlastUnits/3)
			continue
		}
		f.flashes = append(f.flashes, flash{
			x: shot.ToX, y: shot.ToY, reach: 26, life: 0.12, color: sparkYellow,
		})
		f.burst(shot.ToX, shot.ToY, 3, 5, 45, 60, 0.35)
	}
	f.known = make(map[int64]Shot, len(s.Shots))
	for id, shot := range s.Shots {
		f.known[id] = shot
	}

	d := float64(dt)
	sparks := f.sparks[:0]
	for _, p := range f.sparks {
		p.age += dt
		if p.age >= p.life {
			continue
		}
		p.x += p.vx * d
		p.y += p.vy * d
		p.z += p.vz * d
		if p.smoke {
			p.size += 5 * dt
		} else {
			p.vz -= fxGravity * d
			// An ember that comes down stays where it fell, glowing out.
			if p.z < 0 {
				p.z, p.vx, p.vy, p.vz = 0, p.vx*0.3, p.vy*0.3, -p.vz*0.25
			}
		}
		sparks = append(sparks, p)
	}
	f.sparks = sparks
	flashes := f.flashes[:0]
	for _, fl := range f.flashes {
		if fl.age += dt; fl.age < fl.life {
			flashes = append(flashes, fl)
		}
	}
	f.flashes = flashes
}

func (f *fxField) updateShellSmoke(s *State) {
	for _, id := range sortedShotIDs(s) {
		shot := s.Shots[id]
		if shot.Kind != ShotShell {
			continue
		}
		distance := shotTravelled(shot)
		previous, found := f.smokeDistance[id]
		if !found {
			f.smokeDistance[id] = distance
			continue
		}
		for next := previous + shellSmokeStepUnits;
			next <= distance;
			next += shellSmokeStepUnits {
			progress := next / distance
			x := shot.FromX + (shot.X-shot.FromX)*progress
			y := shot.FromY + (shot.Y-shot.FromY)*progress
			f.shellSmoke(x, y, shellHeightAt(shot, next))
		}
		f.smokeDistance[id] = distance
	}
	for id := range f.smokeDistance {
		if _, flying := s.Shots[id]; !flying {
			delete(f.smokeDistance, id)
		}
	}
}

// lightPool adds a soft pool of light on the ground around a spot: a
// stack of ellipses, each smaller and as bright again, so the middle
// burns and the rim fades to nothing. Call it under BlendAdd.
func lightPool(
	screen *golib.Screen,
	x, y, reach float64,
	color golib.Color,
	strength float32,
	rings int,
) {
	if strength <= 0 || reach <= 0 {
		return
	}
	cx, cy := project(float32(x), float32(y))
	for ring := 0; ring < rings; ring++ {
		part := 1 - float32(ring)/float32(rings)
		screen.DrawPolygon(groundEllipse(cx, cy, float32(reach)*part),
			golib.WithOpacity(color, strength/float32(rings)))
	}
}

// groundEllipse returns a circle of the ground, a radius in units around
// a screen point, as the projection flattens it.
func groundEllipse(cx, cy, radius float32) []golib.Vector2 {
	const points = 28
	halfW, halfH := ellipseSemiAxes(radius / unitsPerTile)
	shape := make([]golib.Vector2, points)
	for i := range shape {
		angle := float64(i) / points * 2 * math.Pi
		shape[i] = golib.Vector2{
			X: cx + halfW*float32(math.Cos(angle)),
			Y: cy + halfH*float32(math.Sin(angle)),
		}
	}
	return shape
}

// shotHeight returns how high a shot flies where it is: a bullet skims
// the ground, a shell follows its arc.
func shotHeight(shot Shot) float64 {
	if shot.Kind != ShotShell {
		return 4
	}
	return shellHeightAt(shot, shotTravelled(shot))
}

func shotTravelled(shot Shot) float64 {
	return math.Hypot(shot.X-shot.FromX, shot.Y-shot.FromY)
}

func shellTrailSteps(shot Shot) int {
	steps := int(shotTravelled(shot) / shellTrailStepUnits)
	maximum := int(shellTrailLengthUnits / shellTrailStepUnits)
	if steps > maximum {
		return maximum
	}
	return steps
}

func shellHeightAt(shot Shot, distance float64) float64 {
	whole := math.Hypot(shot.ToX-shot.FromX, shot.ToY-shot.FromY)
	if whole <= 0 {
		return 0
	}
	done := distance / whole
	return 4 * shellArc * whole * done * (1 - done)
}

// draw paints the field over the region: shadows and smoke in plain
// paint, then everything that shines, added to what is under it.
func (f *fxField) draw(s *State, screen *golib.Screen, zoom float32) {
	for _, id := range sortedShotIDs(s) {
		if shot := s.Shots[id]; shot.Kind == ShotShell {
			gx, gy := project(float32(shot.X), float32(shot.Y))
			screen.DrawPolygon(groundEllipse(gx, gy, dotRadius(5, zoom, 2)/unitW*2),
				golib.WithOpacity(golib.Black, 0.35))
		}
	}
	for _, p := range f.sparks {
		if !p.smoke {
			continue
		}
		px, py := project(float32(p.x), float32(p.y))
		fade := 1 - p.age/p.life
		screen.DrawCircle(px, py-float32(p.z)*unitH, dotRadius(p.size, zoom, 1.5),
			golib.WithOpacity(p.color, p.opacity*fade))
	}

	screen.SetBlendMode(golib.BlendAdd)
	for _, fl := range f.flashes {
		fade := 1 - fl.age/fl.life
		lightPool(screen, fl.x, fl.y, fl.reach*(0.6+0.4*float64(1-fade)), fl.color,
			0.9*fade*fade, fxLightRings)
		if fl.ring {
			cx, cy := project(float32(fl.x), float32(fl.y))
			screen.DrawPolygonOutline(
				groundEllipse(cx, cy, float32(fl.reach)*(0.2+1.1*(1-fade))),
				3/zoom, golib.WithOpacity(sparkYellow, 0.7*fade))
		}
	}
	for _, id := range sortedShotIDs(s) {
		shot := s.Shots[id]
		color := bulletLight
		if shot.Rival {
			color = rivalLight
		}
		h := float32(shotHeight(shot)) * unitH
		px, py := project(float32(shot.X), float32(shot.Y))
		dx, dy := shot.ToX-shot.X, shot.ToY-shot.Y
		if gap := math.Hypot(dx, dy); gap > 0 {
			dx, dy = dx/gap, dy/gap
		}
		if shot.Kind == ShotBullet {
			lightPool(screen, shot.X, shot.Y, 26, color, 0.35, fxLightRings/2)
			tx, ty := project(float32(shot.X-dx*14), float32(shot.Y-dy*14))
			screen.DrawLine(tx, ty-h, px, py-h, dotRadius(0.7, zoom, 0.75)*2, color)
			screen.DrawCircle(px, py-h, dotRadius(1, zoom, 1), sparkWhite)
			continue
		}
		// The higher the shell, the wider and the fainter its light below.
		lift := shotHeight(shot)
		lightPool(screen, shot.X, shot.Y, 60+lift*0.5, shellLight,
			float32(0.55/(1+lift/150)), fxLightRings)
		for k := 1; k <= shellTrailSteps(shot); k++ {
			back := shot
			back.X -= dx * float64(k) * shellTrailStepUnits
			back.Y -= dy * float64(k) * shellTrailStepUnits
			bx, by := project(float32(back.X), float32(back.Y))
			screen.DrawCircle(bx, by-float32(shotHeight(back))*unitH,
				dotRadius(2.2, zoom, 1.5)*(1-float32(k)/8),
				golib.WithOpacity(sparkOrange, 0.8-float32(k)*0.11))
		}
		screen.DrawCircle(px, py-h, dotRadius(3, zoom, 2), sparkYellow)
		screen.DrawCircle(px, py-h, dotRadius(1.6, zoom, 1), sparkWhite)
	}
	for _, p := range f.sparks {
		if p.smoke {
			continue
		}
		px, py := project(float32(p.x), float32(p.y))
		fade := 1 - p.age/p.life
		// A spark cools as it goes: whatever it was, it ends an ember's red.
		cool := p.age / p.life
		color := golib.Color{
			R: uint8(golib.Lerp(float32(p.color.R), float32(sparkRed.R), cool)),
			G: uint8(golib.Lerp(float32(p.color.G), float32(sparkRed.G), cool)),
			B: uint8(golib.Lerp(float32(p.color.B), float32(sparkRed.B), cool)),
			A: 255,
		}
		screen.DrawCircle(px, py-float32(p.z)*unitH, dotRadius(p.size, zoom, 1),
			golib.WithOpacity(color, fade))
		// A spark near the ground lights it a little.
		if p.z < 12 {
			lightPool(screen, p.x, p.y, 10, p.color, 0.12*fade, fxSparkRings)
		}
	}
	screen.SetBlendMode(golib.BlendNormal)
}
