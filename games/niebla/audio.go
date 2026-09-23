package main

import (
	"math"

	"golib"
)

// The region's sound. All of it is view: the play scene owns the field,
// the state never hears of it, and its randomness is golib's. The world
// speaks where it happens — a shell sounds from the gun that fired it, a
// drip from the pool it left — and the view weighs every world sound by
// the distance from its source to the camera's center on the ground.
// The three ambience loops are
// synthesized by tools/soundgen; short pings, whistles and impacts are
// made in code.

const (
	uiClickVolume  = 0.8        // a press of the interface
	shellVolume    = 1.0        // the colony's artillery speaking
	shellFarVolume = 0.9        // a rival base's gun
	gunVolume      = 0.9        // small arms, the colony's and the rivals'
	gunCooldown    = 4          // ticks between two small-arm sounds at most
	burstVolume    = 1.0        // a shell landing
	whistleVolume  = 0.18       // a shell coming down where the view stands
	whistleCount   = 8          // independent incoming shells at once
	dripVolume     = 0.6        // an oil pool's bloop
	gurgleVolume   = 0.55       // its rarer, thicker cousin
	tinkVolume     = 0.2125 / 12 // a lilac vein's crystal ping
	ringVolume     = 0.28       // the vein's continuous resonance
	tinkCrowdBoost = 0.15       // how much every extra audible vein lifts a ping
	tinkCrowdHurry = 0.5        // and how much it hurries the next one along
	windFarVolume  = 0.18       // the wind, the whole region in view
	windNearVolume = 0.04       // the wind, the ground in view
	poolBedVolume  = 1.0        // the pool's buried seethe, up close
	dripSoonest    = 120        // ticks between two drips at the least: 2 s
	dripLatest     = 420        // and at the most: 7 s
	gurgleSoonest  = 600        // ticks between gurgles: 10 s
	gurgleLatest   = 1500       // 25 s
	tinkSoonest    = 12         // ticks between two pings at the least: 0.2 s
	tinkLatest     = 75         // and at the most: 1.25 s
	nearZoomSpan   = 2.0        // zooms of glide from a whisper to a full world
	nearZoomFloor  = 0.35       // the weight the world keeps at the farthest stop
	audioReach     = 2800.0     // meters a world sound carries from the camera
	shellReach     = 3800.0     // a cannon carries farther than small arms
	audioHigh      = 2100.0     // receiver height in meters at the farthest zoom
	audioFloor     = 0.02       // quieter than this and a sound doesn't start
)

type audioField struct {
	click    *golib.Sound
	shell    *golib.Sound
	shellFar *golib.Sound
	burst    *golib.Sound
	whistles [whistleCount]*golib.Sound
	gunA     *golib.Sound
	gunB     *golib.Sound
	drip     *golib.Sound
	gurgle   *golib.Sound
	tinks    [3]*golib.Sound
	ring     *golib.Sound
	wind     *golib.Sound
	oilBed   *golib.Sound

	known        map[int64]Shot
	whistled     map[int64]bool
	whistleSlots map[int64]int
	gunWait      int
	dripIn       int
	gurgleIn     int
	tinkIn       int
}

func newAudioField() *audioField {
	a := &audioField{
		click:    golib.NewSoundFile("sounds/click.ogg"),
		shell:    golib.NewSoundFile("sounds/artillery-fire.ogg"),
		shellFar: golib.NewSoundFile("sounds/artillery-fire-distant.ogg"),
		burst: golib.NewSound(golib.SoundSpec{
			// The whump of a shell arriving: a wide, slow breath of
			// noise, deeper and longer than the framework's own burst.
			Wave: golib.WaveNoise, Frequency: 140, Slide: -100,
			Duration: 1.1, Attack: 0.002, Release: 0.95, Volume: 1,
		}),
		gunA:   golib.NewSoundFile("sounds/gun-a.ogg"),
		gunB:   golib.NewSoundFile("sounds/gun-b.ogg"),
		drip:   golib.NewSoundFile("sounds/oil-drip.ogg"),
		gurgle: golib.NewSoundFile("sounds/oil-gurgle.ogg"),
		tinks: [3]*golib.Sound{
			// A crystal's ping: a soft note with a short life of its
			// overtones, and the pitch the play varies.
			golib.NewSound(golib.SoundSpec{
				Wave: golib.WaveTriangle, Frequency: 3840,
				Duration: 0.7, Attack: 0.001, Release: 0.65, Volume: 1,
			}),
			golib.NewSound(golib.SoundSpec{
				Wave: golib.WaveTriangle, Frequency: 5120,
				Duration: 0.65, Attack: 0.001, Release: 0.6, Volume: 1,
			}),
			golib.NewSound(golib.SoundSpec{
				Wave: golib.WaveTriangle, Frequency: 6400,
				Duration: 0.6, Attack: 0.001, Release: 0.55, Volume: 1,
			}),
		},
		ring:         golib.NewSoundFile("sounds/mineral-ring.wav"),
		wind:         golib.NewSoundFile("sounds/wind-loop.ogg"),
		oilBed:       golib.NewSoundFile("sounds/oil-bed.ogg"),
		known:        map[int64]Shot{},
		whistled:     map[int64]bool{},
		whistleSlots: map[int64]int{},
		dripIn:       dripSoonest,
		gurgleIn:     gurgleSoonest,
		tinkIn:       tinkSoonest,
	}
	for i := range a.whistles {
		a.whistles[i] = golib.NewSound(golib.SoundSpec{
			Wave: golib.WaveSine, Frequency: 650, Slide: -55,
			Duration: 10, Attack: 0.05, Release: 0.4, Volume: 0.5,
		})
	}
	return a
}

// ui plays the interface's click; the pitch keeps each kind of press
// recognizable by ear alone.
func (a *audioField) ui(pitch float32) {
	a.click.PlayWith(uiClickVolume, pitch)
}

// nearness says how close the view stands to the ground: the weight
// every world sound carries. It never reaches nothing - far out, what
// stands by the view's middle still whispers under the wind - and from
// zoom 3 on it is whole.
func nearness(zoom float32) float32 {
	return nearZoomFloor +
		(1-nearZoomFloor)*golib.Clamp((zoom-1)/nearZoomSpan, 0, 1)
}

// audible weighs a world sound against the listener at the camera's
// center, including its height above the ground at the current zoom.
func (a *audioField) audible(s *playScene, x, y, base float64) float32 {
	return a.audibleFrom(s, x, y, base, audioReach)
}

func (a *audioField) audibleFrom(
	s *playScene, x, y, base, reach float64,
) float32 {
	center := s.camera.Center()
	rx, ry := unproject(center.X, center.Y)
	stop := golib.Clamp(
		float32(math.Log2(float64(s.zoom))), zoomOut, zoomIn,
	)
	height := audioHigh * (1 - float64(stop)/zoomIn)
	dx, dy := x-float64(rx), y-float64(ry)
	fall := 1 - math.Sqrt(dx*dx+dy*dy+height*height)/reach
	if fall <= 0 {
		return 0
	}
	v := base * fall
	if v < audioFloor {
		return 0
	}
	return float32(math.Min(v, 1))
}

// update keeps the two loops breathing, drops the pool's bloops in from
// time to time, and learns of fired and landed shots the way the lights
// do, by comparing the state's with the ones it saw last.
func (a *audioField) update(s *playScene, ticks int) {
	a.wind.SetVolume(windFarVolume + (windNearVolume-windFarVolume)*nearness(s.zoom))
	a.wind.Loop()

	if _, _, heard, _ := a.nearestDeposit(s, kindOil); heard > 0 {
		a.oilBed.SetVolume(float32(math.Sqrt(float64(heard))) * poolBedVolume)
		a.oilBed.Loop()
		a.dripIn -= ticks
		a.gurgleIn -= ticks
		if a.dripIn <= 0 {
			a.drip.PlayWith(heard*dripVolume, golib.RandomFloat(0.7, 1.1))
			a.dripIn = dripSoonest + golib.RandomInt(0, dripLatest-dripSoonest)
		}
		if a.gurgleIn <= 0 {
			a.gurgle.PlayWith(heard*gurgleVolume, golib.RandomFloat(0.9, 1.1))
			a.gurgleIn = gurgleSoonest + golib.RandomInt(0, gurgleLatest-gurgleSoonest)
		}
	} else {
		a.oilBed.SetVolume(0)
		a.oilBed.Stop()
	}

	if _, _, heard, crowd := a.nearestDeposit(s, kindLilac); heard > 0 {
		a.ring.SetVolume(heard * ringVolume)
		a.ring.Loop()
		a.tinkIn -= ticks
		if a.tinkIn <= 0 {
			tink := a.tinks[golib.RandomInt(0, len(a.tinks)-1)]
			loud := heard * tinkVolume *
				min(1.4, 1+tinkCrowdBoost*float32(crowd-1))
			tink.PlayWith(loud, golib.RandomFloat(0.9, 1.15))
			hurry := 1 + tinkCrowdHurry*float32(crowd-1)
			a.tinkIn = int(float32(tinkSoonest+
				golib.RandomInt(0, tinkLatest-tinkSoonest)) / hurry)
		}
	} else {
		a.ring.Stop()
		a.tinkIn = tinkSoonest
	}

	a.gunWait = max(0, a.gunWait-ticks)
	for id, shot := range s.state.Shots {
		if _, old := a.known[id]; old {
			continue
		}
		a.fired(s, shot)
	}
	for _, id := range sortedShotIDs(s.state) {
		shot := s.state.Shots[id]
		if shot.Kind != ShotShell {
			continue
		}
		remaining := math.Hypot(shot.ToX-shot.X, shot.ToY-shot.Y)
		whole := math.Hypot(shot.ToX-shot.FromX, shot.ToY-shot.FromY)
		if remaining > whole/2 {
			continue
		}
		newWhistle := false
		if !a.whistled[id] {
			a.whistled[id] = true
			if slot, ok := a.freeWhistle(); ok {
				a.whistleSlots[id] = slot
				newWhistle = true
			}
		}
		if slot, ok := a.whistleSlots[id]; ok {
			fade := math.Min(1, remaining/(shellSpeed*0.35))
			v := a.audible(s, shot.X, shot.Y, whistleVolume*fade)
			a.whistles[slot].SetVolume(v)
			if newWhistle {
				a.whistles[slot].Play()
			}
		}
	}
	for id, shot := range a.known {
		if _, flying := s.state.Shots[id]; flying {
			continue
		}
		if slot, ok := a.whistleSlots[id]; ok {
			a.whistles[slot].Stop()
			delete(a.whistleSlots, id)
		}
		if shot.Kind != ShotShell {
			continue
		}
		if v := a.audible(s, shot.ToX, shot.ToY, burstVolume); v > 0 {
			a.burst.PlayWith(v, golib.RandomFloat(0.9, 1.1))
		}
	}
	sawWhistle := a.whistled
	a.known = make(map[int64]Shot, len(s.state.Shots))
	a.whistled = make(map[int64]bool, len(s.state.Shots))
	for id, shot := range s.state.Shots {
		a.known[id] = shot
		a.whistled[id] = sawWhistle[id]
	}
}

func (a *audioField) freeWhistle() (int, bool) {
	var used [whistleCount]bool
	for _, slot := range a.whistleSlots {
		used[slot] = true
	}
	for i := range used {
		if !used[i] {
			return i, true
		}
	}
	return 0, false
}

// fired sounds a shot leaving its gun: the colony's artillery its cannon,
// a rival base's the far one, any small arm a shorter report, held to one
// every few ticks so a battle doesn't turn into a rattle.
func (a *audioField) fired(s *playScene, shot Shot) {
	if shot.Kind == ShotShell {
		sound, base := a.shell, shellVolume
		if shot.Rival {
			sound, base = a.shellFar, shellFarVolume
		}
		if v := a.audibleFrom(
			s, shot.FromX, shot.FromY, base, shellReach,
		); v > 0 {
			sound.PlayWith(v, golib.RandomFloat(0.96, 1.04))
		}
		return
	}
	if a.gunWait > 0 {
		return
	}
	v := a.audible(s, shot.FromX, shot.FromY, gunVolume)
	if v <= 0 {
		return
	}
	a.gunWait = gunCooldown
	gun := a.gunA
	if shot.ID%2 == 0 {
		gun = a.gunB
	}
	gun.PlayWith(v, golib.RandomFloat(0.92, 1.08))
}

// nearestDeposit finds the deposit of a kind closest to the view's
// middle and how loudly it sounds there, in units, so the pools seethe
// and the veins sparkle where the ore is — never where it ran out. It
// also counts the deposits of the kind the view hears at all, so the
// veins' shimmer can grow with how much mineral is in earshot.
func (a *audioField) nearestDeposit(s *playScene, kind byte) (float64, float64, float32, int) {
	var bx, by float64
	heard := float32(0)
	crowd := 0
	for i := range land.deposits {
		d := land.deposits[i]
		if d.Kind != kind {
			continue
		}
		tcol, trow := cellTile(d.HeartCol, d.HeartRow)
		if remainingAt(s.state, tcol, trow) <= 0 {
			continue
		}
		x, y := cellCenterUnits(d.HeartCol, d.HeartRow)
		if v := a.audible(s, x, y, 1); v > 0 {
			crowd++
			if v > heard {
				heard, bx, by = v, x, y
			}
		}
	}
	return bx, by, heard, crowd
}

// stopLoops silences what keeps sounding on its own, for the way out of
// the region; whatever plays once is left to finish.
func (a *audioField) stopLoops() {
	a.wind.Stop()
	a.oilBed.Stop()
	a.ring.Stop()
	for _, sound := range a.whistles {
		sound.Stop()
	}
}
