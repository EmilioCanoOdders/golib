package golib

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Tune limits. A tune is made in memory, at soundSampleRate, so a long one
// costs memory: two minutes take about 10 MB.
const (
	tuneMaxSeconds = 120
	tuneMaxVoices  = 8
	tuneBeatGap    = 0.04 // seconds of silence at the end of a note by default, so notes are told apart
)

// TuneSpec is the recipe for music made from notes, without a music file:
// how fast it goes, and the voices that play together.
//
//	var theme = golib.NewTune(golib.TuneSpec{
//		Tempo: 132,
//		Voices: []golib.Voice{
//			{Notes: "c4 e4 g4 e4 f4 a4 c5 a4"},                                  // the melody
//			{Wave: golib.WaveTriangle, Volume: 0.4, Notes: "c2/2 c2/2 f2/2 f2/2"}, // the bass
//		},
//	})
type TuneSpec struct {
	// Tempo is the speed, in beats per minute, from 20 to 400. A beat is
	// what a note without a length lasts, so a tune written in eighth notes
	// at 110 beats per minute has a Tempo of 220. Default: 120.
	Tempo float32

	// Voices are the lines that play together, at most 8. A tune needs one.
	Voices []Voice

	// Reverb plays the tune in a room, from 0 to 1: 0.2 a small room, 0.5 a
	// hall, 1 a cave. Its echoes soften notes made in code, which sound dry
	// on their own. The echoes at the end of the tune carry over into its
	// start, so it loops without a seam. Default: 0, no room.
	Reverb float32
}

// Voice is one line of a [TuneSpec]: a sound, and the notes it plays.
type Voice struct {
	// Wave is the voice's sound, as in [SoundSpec]. Default: WaveSquare.
	Wave Waveform

	// Volume is how loud this voice is, from 0 to 1, before the tune's own
	// volume. Default: 0.5.
	Volume float32

	// Duty shapes a square wave, from 0.05 to 0.95, as in [SoundSpec].
	// Default: 0.5.
	Duty float32

	// Vibrato wobbles the pitch by this many Hz, as in [SoundSpec], and
	// VibratoRate says how many times a second. Default: no wobble, 12 times
	// a second.
	Vibrato     float32
	VibratoRate float32

	// Gap is the silence at the end of every note, in seconds, so that two of
	// the same note in a row are heard as two. Default: 0.04. Use 0 for
	// notes that run into each other, and more for a short, clipped sound,
	// such as a drum: a note is never shorter than half its length.
	Gap float32

	// Attack is how long each note takes to swell to its full loudness, in
	// seconds, as a bowed string or a soft pad does: 0.1 to 0.5. Default:
	// 0.005, at once.
	Attack float32

	// Decay makes each note fade away as a plucked or struck string does:
	// Decay seconds after it starts it is at about a third of its loudness,
	// and it keeps fading. Short for a marimba or a pluck, 0.1 to 0.3, long
	// for a piano or a bell, 1 to 2. Default: 0, a note holds its loudness.
	Decay float32

	// Ring is how long each note goes on sounding after it ends, in seconds,
	// fading out over the notes that follow, as a piano does with its pedal
	// down: up to 10. Notes ringing at the end of the tune carry over into its
	// start, so it loops without a seam. Default: 0, a note ends with its
	// length.
	Ring float32

	// LowPass softens the voice above this many Hz, which takes the edge off
	// it: around 3000 for a warm square or saw wave, 800 for a muffled one,
	// from 20 to 20000. Default: 0, no filter.
	LowPass float32

	// Detune adds a second copy of the wave, this many cents higher, a
	// hundredth of a semitone each, which beats against the first for a
	// fuller, warmer sound, like a chorus: 5 to 15 is subtle, up to 100.
	// Not for WaveNoise. Default: 0, one wave.
	Detune float32

	// Notes are the notes to play, separated by spaces, each of them a letter
	// from a to g, an optional # or b, and the octave, such as "c4", "f#3" or
	// "eb5"; capital letters work too. Middle C is c4, and octaves go from 0
	// to 8. A dot is a silence, and a dash holds the note before it for
	// another beat. "/" and a number give a length in beats: "c4/2" lasts two
	// beats and "c4/0.5" half a beat.
	//
	//	"c4 e4 g4 c5 - - . g4/0.5 e4/0.5 c4/2"
	//
	// Every voice of a tune should add up to the same number of beats, or the
	// short ones end in silence and the loop falls out of step. Counting the
	// beats of each voice is worth a test in the game.
	Notes string
}

// NewTune returns music made from the notes in spec, for a game that has no
// music file. It plays like music read from a file, and loops:
//
//	theme.Play()  // in Update, such as when the play scene starts
//	theme.Pause() // while the game is paused
//
// The tune is made the first time it plays, so a game can create it before
// Run opens the window, as a package variable. Making it takes a moment, a
// few tens of milliseconds for a tune of half a minute, and longer with
// Voice.Ring and TuneSpec.Reverb, about a third of a second for a minute and
// a half, so start it on a title screen rather than in the middle of the
// action. A mistake in the
// notes, or a tune longer than two minutes, stops Run with a message, and
// [Music.Err] returns it, in tests too:
//
//	var themeSpec = golib.TuneSpec{...} // keep the recipe to test it
//	var theme = golib.NewTune(themeSpec)
func NewTune(spec TuneSpec) *Music {
	tune := spec
	return &Music{name: "the tune", tune: &tune, volume: 1}
}

// samples renders the tune: every voice's notes, mixed together.
func (spec TuneSpec) samples() ([]int16, error) {
	tempo := spec.Tempo
	if tempo == 0 {
		tempo = 120
	}
	if tempo < 20 || tempo > 400 {
		return nil, fmt.Errorf("golib.NewTune: the tempo is %g beats per minute: use 20 to 400", tempo)
	}
	if len(spec.Voices) == 0 {
		return nil, fmt.Errorf("golib.NewTune: the tune has no voices: give it at least one, with its notes")
	}
	if len(spec.Voices) > tuneMaxVoices {
		return nil, fmt.Errorf("golib.NewTune: the tune has %d voices: use at most %d", len(spec.Voices), tuneMaxVoices)
	}
	if spec.Reverb < 0 || spec.Reverb > 1 {
		return nil, fmt.Errorf("golib.NewTune: the reverb is %g: use 0, none, to 1, a cave", spec.Reverb)
	}
	beat := 60 / float64(tempo)

	type rendered struct {
		samples []int16 // the voice's notes, and after them what rings on past its end
		length  int     // the samples its notes last
	}
	lines := make([]rendered, len(spec.Voices))
	length := 0
	for i, voice := range spec.Voices {
		if err := voice.check(); err != nil {
			return nil, fmt.Errorf("golib.NewTune: voice %d: %w", i+1, err)
		}
		notes, err := parseNotes(voice.Notes)
		if err != nil {
			return nil, fmt.Errorf("golib.NewTune: voice %d: %w", i+1, err)
		}
		if len(notes) == 0 {
			return nil, fmt.Errorf("golib.NewTune: voice %d has no notes", i+1)
		}
		volume := voice.Volume
		if volume == 0 {
			volume = 0.5
		}
		line, notesLength, err := voice.render(notes, beat, volume)
		if err != nil {
			return nil, fmt.Errorf("golib.NewTune: voice %d: %w", i+1, err)
		}
		lines[i] = rendered{line, notesLength}
		// The longest voice sets the tune's length; shorter ones end in
		// silence.
		length = max(length, notesLength)
	}

	mixed := make([]int16, length)
	if length == 0 {
		return mixed, nil // notes too short to last a sample
	}
	for _, line := range lines {
		// What rings on past the end of the tune sounds over its start, as it
		// would when the tune loops.
		for j, sample := range line.samples {
			at := j % length
			mixed[at] = clipSample(int(mixed[at]) + int(sample))
		}
	}
	if spec.Reverb > 0 {
		reverberate(mixed, float64(spec.Reverb))
	}
	return mixed, nil
}

// check reports the first of the voice's settings out of range.
func (v Voice) check() error {
	for _, setting := range []struct {
		name       string
		value, top float32
		unit       string
	}{
		{"Attack", v.Attack, 10, "seconds"},
		{"Decay", v.Decay, 10, "seconds"},
		{"Ring", v.Ring, 10, "seconds"},
		{"LowPass", v.LowPass, 20000, "Hz"},
		{"Detune", v.Detune, 100, "cents"},
	} {
		if setting.value < 0 || setting.value > setting.top {
			return fmt.Errorf("%s is %g: use 0 to %g %s, or leave it out", setting.name, setting.value, setting.top, setting.unit)
		}
	}
	if v.LowPass > 0 && v.LowPass < 20 {
		return fmt.Errorf("LowPass is %g Hz, below what anyone hears: use 20 to 20000 Hz, or leave it out for no filter", v.LowPass)
	}
	return nil
}

// render returns the samples of one voice, each note made by the same
// synthesizer as the sound effects, and how many of them its notes last:
// with Ring, the samples go on past that, as the last notes ring on.
func (v Voice) render(notes []note, beat float64, volume float32) ([]int16, int, error) {
	var samples []int16
	at := 0 // where the next note starts
	for _, n := range notes {
		length := n.beats * beat
		if float64(at)/soundSampleRate+length > tuneMaxSeconds {
			return nil, 0, fmt.Errorf("the tune is longer than %d seconds: make it shorter, and let it loop", tuneMaxSeconds)
		}
		count := int(length * soundSampleRate)
		if n.frequency == 0 { // a silence
			at += count
			samples = grow(samples, at)
			continue
		}
		// The note stops a little before the next one starts, so that two of
		// the same note in a row are heard as two.
		gap := float64(tuneBeatGap)
		if v.Gap != 0 {
			gap = float64(max(0, v.Gap))
		}
		sound := min(length, max(length-gap, length/2))
		spec := SoundSpec{
			Wave:        v.Wave,
			Frequency:   float32(n.frequency),
			Duration:    float32(sound),
			Attack:      0.005,
			Release:     float32(min(0.05, sound/2)),
			Volume:      volume,
			Duty:        v.Duty,
			Vibrato:     v.Vibrato,
			VibratoRate: v.VibratoRate,
			decay:       v.Decay,
			detune:      v.Detune,
		}
		if v.Attack > 0 {
			spec.Attack = v.Attack
		}
		if v.Ring > 0 {
			// The note sounds on past its end, fading out as it rings.
			spec.Duration = float32(sound) + v.Ring
			spec.Release = v.Ring
		}
		played := spec.samples()
		if v.Ring == 0 && len(played) > count {
			played = played[:count]
		}
		samples = grow(samples, at+len(played))
		for i, sample := range played {
			samples[at+i] = clipSample(int(samples[at+i]) + int(sample))
		}
		at += count
		samples = grow(samples, at)
	}
	if v.LowPass > 0 {
		lowPass(samples, at, float64(v.LowPass))
	}
	return samples, at, nil
}

// grow returns samples made at least length long, with silence.
func grow(samples []int16, length int) []int16 {
	if length > len(samples) {
		samples = append(samples, make([]int16, length-len(samples))...)
	}
	return samples
}

// lowPass softens samples above cutoff Hz, in place, with a two-pole filter,
// as a voice's tone control would. loop is where the voice's notes end and it
// starts again: the filter starts as it stands there, so the loop has no
// seam.
func lowPass(samples []int16, loop int, cutoff float64) {
	cutoff = min(cutoff, soundSampleRate*0.45)
	// A Butterworth low-pass, from the Audio EQ Cookbook.
	w := 2 * math.Pi * cutoff / soundSampleRate
	alpha := math.Sin(w) / math.Sqrt2
	a0 := 1 + alpha
	b0 := (1 - math.Cos(w)) / 2 / a0
	b1 := (1 - math.Cos(w)) / a0
	b2 := b0
	a1 := -2 * math.Cos(w) / a0
	a2 := (1 - alpha) / a0

	var x1, x2, y1, y2 float64
	filter := func(x float64) float64 {
		y := b0*x + b1*x1 + b2*x2 - a1*y1 - a2*y2
		x2, x1 = x1, x
		y2, y1 = y1, y
		return y
	}
	// A tenth of a second before the loop's end sets the filter going.
	for i := max(0, loop-soundSampleRate/10); i < loop && i < len(samples); i++ {
		filter(float64(samples[i]))
	}
	for i, sample := range samples {
		samples[i] = clipSample(int(math.Round(filter(float64(sample)))))
	}
}

// reverberate adds the echoes of a room to samples, in place, from amount 0,
// none, to 1, a cave: Schroeder's reverb, four echoing delays side by side
// and two that blur them, with Freeverb's delays. The samples loop, so the
// room starts as it rings at their end.
func reverberate(samples []int16, amount float64) {
	feedback := 0.7 + 0.2*amount // how long the room rings
	const damping = 0.3          // how much faster it loses its highs
	wet := 0.08 * amount         // how loud the echoes are

	// The delays, in samples: four echoing ones side by side, then two that
	// blur their echoes, one after the other.
	combDelays := [4]int{1557, 1617, 1491, 1422}
	allPassDelays := [2]int{556, 441}
	var combs [4][]float64
	var combAt [4]int
	var combStore [4]float64
	for c, delay := range combDelays {
		combs[c] = make([]float64, delay)
	}
	var allPasses [2][]float64
	var allPassAt [2]int
	for a, delay := range allPassDelays {
		allPasses[a] = make([]float64, delay)
	}
	echo := func(x float64) float64 {
		out := 0.0
		for c := range combs {
			buffer, at := combs[c], combAt[c]
			y := buffer[at]
			combStore[c] = y*(1-damping) + combStore[c]*damping
			buffer[at] = x + combStore[c]*feedback
			if at++; at == len(buffer) {
				at = 0
			}
			combAt[c] = at
			out += y
		}
		for a := range allPasses {
			buffer, at := allPasses[a], allPassAt[a]
			delayed := buffer[at]
			buffer[at] = out + delayed*0.5
			if at++; at == len(buffer) {
				at = 0
			}
			allPassAt[a] = at
			out = delayed - out
		}
		return out
	}

	// Three seconds of the tune's end, over and over if it is shorter, set
	// the room ringing before its start.
	warm := 3 * soundSampleRate
	for i := range warm {
		echo(float64(samples[(len(samples)-warm%len(samples)+i)%len(samples)]))
	}
	for i, sample := range samples {
		samples[i] = clipSample(int(math.Round(float64(sample) + wet*echo(float64(sample)))))
	}
}

// note is one note of a voice: how many beats it lasts, and the pitch it
// sounds at, or 0 for a silence.
type note struct {
	frequency float64
	beats     float64
}

// noteSteps gives each note letter its place in an octave, in semitones from
// C.
var noteSteps = map[byte]int{'c': 0, 'd': 2, 'e': 4, 'f': 5, 'g': 7, 'a': 9, 'b': 11}

// parseNotes turns the notes of a voice into pitches and lengths.
func parseNotes(notes string) ([]note, error) {
	var parsed []note
	for _, item := range strings.Fields(notes) {
		text, beats := item, 1.0
		if name, length, found := strings.Cut(item, "/"); found {
			value, err := strconv.ParseFloat(length, 64)
			if err != nil || value <= 0 || value > 64 {
				return nil, fmt.Errorf("%q: after / write how many beats the note lasts, such as %q or %q", item, name+"/2", name+"/0.5")
			}
			text, beats = name, value
		}
		if text == "-" {
			if len(parsed) == 0 {
				return nil, fmt.Errorf("%q: a dash holds the note before it, so it can't come first", item)
			}
			parsed[len(parsed)-1].beats += beats
			continue
		}
		if text == "." {
			parsed = append(parsed, note{beats: beats})
			continue
		}
		frequency, err := noteFrequency(text)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", item, err)
		}
		parsed = append(parsed, note{frequency: frequency, beats: beats})
	}
	return parsed, nil
}

// noteFrequency returns the pitch of a note such as "c4", "f#3" or "eb5", in
// Hz, with a4 at 440 Hz.
func noteFrequency(text string) (float64, error) {
	invalid := fmt.Errorf(`write a note as a letter from a to g, an optional # or b, and its octave, such as "c4", "f#3" or "eb5", "." for a silence, or "-" to hold the note before it`)
	if len(text) < 2 {
		return 0, invalid

	}
	step, found := noteSteps[text[0]|0x20] // lowercase
	if !found {
		return 0, invalid
	}
	rest := text[1:]
	switch rest[0] {
	case '#':
		step++
		rest = rest[1:]
	case 'b', 'B':
		step--
		rest = rest[1:]
	}
	octave, err := strconv.Atoi(rest)
	if err != nil || octave < 0 || octave > 8 {
		return 0, invalid
	}
	// a4 is 440 Hz, and an octave doubles the pitch. Steps count from c0.
	semitones := step + 12*octave - (noteSteps['a'] + 12*4)
	return 440 * math.Pow(2, float64(semitones)/12), nil
}

// clipSample keeps a mixed sample within what a 16-bit sample holds.
func clipSample(value int) int16 {
	return int16(max(math.MinInt16, min(value, math.MaxInt16)))
}
