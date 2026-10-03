package golib

import (
	"math"
	"strings"
	"testing"
)

func TestNoteFrequency(t *testing.T) {
	tests := map[string]float64{
		"a4":  440,
		"A4":  440,
		"c4":  261.6256, // middle C
		"c5":  523.2511,
		"a3":  220,
		"f#3": 184.9972,
		"gb3": 184.9972, // the same note, written as a flat
		"eb5": 622.2540,
		"b0":  30.8677,
	}
	for text, want := range tests {
		got, err := noteFrequency(text)
		if err != nil || math.Abs(got-want) > 0.001 {
			t.Errorf("noteFrequency(%q) = %v, %v; want %v", text, got, err, want)
		}
	}
	for _, text := range []string{"", "h4", "c", "c9", "c-1", "4c", "c4x", "#4"} {
		if got, err := noteFrequency(text); err == nil {
			t.Errorf("noteFrequency(%q) = %v, want an error", text, got)
		}
	}
}

func TestParseNotes(t *testing.T) {
	notes, err := parseNotes("  c4 . g4/2 c5/0.5\n a4 ")
	if err != nil {
		t.Fatal(err)
	}
	want := []note{
		{frequency: 261.6256, beats: 1},
		{beats: 1}, // the silence
		{frequency: 391.9954, beats: 2},
		{frequency: 523.2511, beats: 0.5},
		{frequency: 440, beats: 1},
	}
	if len(notes) != len(want) {
		t.Fatalf("parsed %d notes, want %d", len(notes), len(want))
	}
	for i, n := range notes {
		if math.Abs(n.frequency-want[i].frequency) > 0.001 || n.beats != want[i].beats {
			t.Errorf("note %d = %+v, want %+v", i, n, want[i])
		}
	}
	for _, notes := range []string{"c4/", "c4/0", "c4/x", "c4/-1", "c4/100", "q4"} {
		if _, err := parseNotes(notes); err == nil {
			t.Errorf("parseNotes(%q): no error", notes)
		}
	}
}

func TestTuneSamples(t *testing.T) {
	// Two beats at 120 beats per minute last a second.
	spec := TuneSpec{Voices: []Voice{{Notes: "c4 e4"}}}
	samples, err := spec.samples()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(samples); got != soundSampleRate {
		t.Errorf("%d samples, want %d: two beats at the default tempo", got, soundSampleRate)
	}
	// The second note starts where the first ends.
	if samples[0] == 0 && samples[100] == 0 {
		t.Error("the tune starts silent")
	}
	if samples[soundSampleRate/2+100] == 0 {
		t.Error("the second note is silent")
	}
	// A note stops a little before the next one starts.
	if samples[soundSampleRate/2-10] != 0 {
		t.Error("the first note plays up to the second")
	}

	// A faster tempo makes a shorter tune, and the longest voice sets the
	// length.
	spec = TuneSpec{Tempo: 240, Voices: []Voice{
		{Notes: "c4 e4"},
		{Wave: WaveTriangle, Volume: 0.3, Notes: "c3/4"},
	}}
	samples, err = spec.samples()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(samples), soundSampleRate; got != want {
		t.Errorf("%d samples, want %d: four beats at 240 beats per minute", got, want)
	}
	// Both voices sound together at the start, and only the bass at the end.
	if samples[100] == 0 || samples[len(samples)-2000] == 0 {
		t.Error("a voice is missing")
	}

	tests := map[string]TuneSpec{
		"no voices":                {},
		"no notes":                 {Voices: []Voice{{Notes: "  "}}},
		"a slow tempo":             {Tempo: 5, Voices: []Voice{{Notes: "c4"}}},
		"a fast tempo":             {Tempo: 4000, Voices: []Voice{{Notes: "c4"}}},
		"too long":                 {Tempo: 20, Voices: []Voice{{Notes: strings.Repeat("c4/64 ", 20)}}},
		"a wrong note":             {Voices: []Voice{{Notes: "c4 h4"}}},
		"a negative decay":         {Voices: []Voice{{Decay: -1, Notes: "c4"}}},
		"a long ring":              {Voices: []Voice{{Ring: 30, Notes: "c4"}}},
		"a low-pass too high":      {Voices: []Voice{{LowPass: 30000, Notes: "c4"}}},
		"a low-pass below hearing": {Voices: []Voice{{LowPass: 5, Notes: "c4"}}},
		"a detune too far":         {Voices: []Voice{{Detune: 200, Notes: "c4"}}},
		"too much reverb":          {Reverb: 2, Voices: []Voice{{Notes: "c4"}}},
		"too many lines": {Voices: []Voice{
			{Notes: "c4"}, {Notes: "c4"}, {Notes: "c4"}, {Notes: "c4"},
			{Notes: "c4"}, {Notes: "c4"}, {Notes: "c4"}, {Notes: "c4"}, {Notes: "c4"},
		}},
	}
	for name, spec := range tests {
		if _, err := spec.samples(); err == nil {
			t.Errorf("%s: no error", name)
		} else if !strings.HasPrefix(err.Error(), "golib.NewTune: ") {
			t.Errorf("%s: %v; want a message that starts with golib.NewTune", name, err)
		}
	}
}

func TestTuneWithoutASoundDevice(t *testing.T) {
	if audio.isReady() {
		t.Skip("a sound device is open")
	}
	takeError()
	good := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 e4 g4"}}})
	good.Play()
	if err := takeError(); err != nil {
		t.Errorf("a tune that is fine reported %v", err)
	}
	if good.Playing() {
		t.Error("the tune plays without a sound device")
	}

	broken := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 x9"}}})
	broken.Play()
	wantError(t, `golib.NewTune: voice 1: "x9"`)
	// Every play reports it again, and the tune is only made once.
	broken.Play()
	wantError(t, "golib.NewTune: voice 1")
}

func TestTuneWithADevice(t *testing.T) {
	if audio.isReady() {
		t.Skip("a sound device is open")
	}
	SetVolume(0) // silent
	audio.open()
	t.Cleanup(func() {
		audio.close()
		SetVolume(1)
	})
	if !audio.isReady() {
		t.Skip("this machine has no sound device")
	}
	tune := NewTune(TuneSpec{Tempo: 200, Voices: []Voice{
		{Notes: "c4 e4 g4 c5"},
		{Wave: WaveTriangle, Volume: 0.3, Notes: "c3/2 g2/2"},
	}})
	tune.SetVolume(0.5)
	tune.Play()
	if err := takeError(); err != nil {
		t.Fatal(err)
	}
	if !tune.Playing() || !tune.loaded {
		t.Fatalf("after Play: playing %v, loaded %v", tune.Playing(), tune.loaded)
	}
	if !tune.stream.Looping {
		t.Error("the tune doesn't loop")
	}
	// Run feeds it every frame.
	if err := audio.updateMusic(); err != nil {
		t.Fatal(err)
	}
	tune.Pause()
	if tune.Playing() {
		t.Error("the tune plays while paused")
	}
	tune.Play()
	tune.Stop()
	if tune.Playing() {
		t.Error("the tune plays after Stop")
	}

	broken := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 q1"}}})
	broken.Play()
	if err := audio.updateMusic(); err == nil || !strings.Contains(err.Error(), "golib.NewTune") {
		t.Errorf("a broken tune with a sound device: %v", err)
	}
}

func TestTuneHoldsAndGaps(t *testing.T) {
	// A dash holds the note before it for another beat.
	notes, err := parseNotes("c4 - -/2 . -")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 || notes[0].beats != 4 || notes[1].beats != 2 {
		t.Errorf("parsed %+v, want a 4-beat note and a 2-beat silence", notes)
	}
	if _, err := parseNotes("- c4"); err == nil {
		t.Error("a dash before any note: no error")
	}

	// Gap says how much silence ends each note, so a drum can be short.
	full := Voice{Gap: 0, Notes: "c4"}
	clipped := Voice{Gap: 0.4, Notes: "c4"}
	loud := func(v Voice) int {
		samples, _, err := v.render(mustParse(t, v.Notes), 1, 0.5)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, sample := range samples {
			if sample != 0 {
				count++
			}
		}
		return count
	}
	if a, b := loud(full), loud(clipped); a <= b {
		t.Errorf("a note sounds for %d samples with no gap and %d with a 0.4 second one; want less with the gap", a, b)
	}
}

// mustParse parses notes for a test.
func mustParse(t *testing.T, notes string) []note {
	t.Helper()
	parsed, err := parseNotes(notes)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestMusicErr(t *testing.T) {
	if err := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 e4"}}}).Err(); err != nil {
		t.Errorf("a tune that is fine: Err() = %v", err)
	}
	err := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 zz"}}}).Err()
	if err == nil || !strings.Contains(err.Error(), "golib.NewTune: voice 1") {
		t.Errorf("a broken tune: Err() = %v", err)
	}
	if pending := takeError(); pending != nil {
		t.Errorf("Err() reported %v to Run", pending)
	}
	if err := NewMusic("music/theme.ogg").Err(); err != nil {
		t.Errorf("music whose file hasn't been read yet: Err() = %v", err)
	}
}

// TestGuideTune checks the tune in the Music section of README.md: its voices
// are the same length, so the loop stays in step.
func TestGuideTune(t *testing.T) {
	spec := TuneSpec{
		Tempo:  264,
		Reverb: 0.3,
		Voices: []Voice{
			{Wave: WaveTriangle, Decay: 0.6, Ring: 0.3, Notes: "c5 - g4 . a4 g4 e4 c4 d4/2 g4/2 c5/4"},
			{Wave: WaveTriangle, Volume: 0.35, LowPass: 600, Notes: "c3/2 g3/2 c3/2 g3/2 f3/2 c4/2 g3/2 g3/2"},
			{Wave: WaveNoise, Volume: 0.05, Gap: 0.1, Decay: 0.03, Notes: ". a7 . a7 . a7 . a7 . a7 . a7 . a7 . a7"},
		},
	}
	for i, voice := range spec.Voices {
		beats := 0.0
		for _, n := range mustParse(t, voice.Notes) {
			beats += n.beats
		}
		if beats != 16 {
			t.Errorf("voice %d lasts %v beats, want 16", i+1, beats)
		}
	}
	samples, err := spec.samples()
	if err != nil {
		t.Fatal(err)
	}
	want := int(math.Round(16 * 60.0 / 264.0 * float64(soundSampleRate)))
	if samples == nil || len(samples) < want-100 || len(samples) > want+100 {
		t.Errorf("the tune is %d samples, want about %d", len(samples), want)
	}
}

// TestTuneInstruments checks the settings that make a voice sound like an
// instrument rather than a bare wave: each does what its comment says.
func TestTuneInstruments(t *testing.T) {
	render := func(spec TuneSpec) []int16 {
		t.Helper()
		samples, err := spec.samples()
		if err != nil {
			t.Fatal(err)
		}
		return samples
	}
	at := func(seconds float64) int { return int(seconds * soundSampleRate) }

	// A note of two seconds: held, it is as loud near its end as at its
	// start; with Decay it has faded away.
	held := render(TuneSpec{Voices: []Voice{{Notes: "a4/4"}}})
	plucked := render(TuneSpec{Voices: []Voice{{Decay: 0.3, Notes: "a4/4"}}})
	if start, end := loudness(held[:at(0.2)]), loudness(held[at(1.6):at(1.8)]); end < start*0.9 {
		t.Errorf("a held note fades from %.0f to %.0f", start, end)
	}
	if start, end := loudness(plucked[:at(0.2)]), loudness(plucked[at(1.6):at(1.8)]); end > start/20 {
		t.Errorf("a note with Decay 0.3 is still at %.0f of %.0f after 1.6 seconds", end, start)
	}

	// With Attack, the note starts softly.
	swelled := render(TuneSpec{Voices: []Voice{{Attack: 0.5, Notes: "a4/4"}}})
	if soft, full := loudness(swelled[:at(0.05)]), loudness(held[:at(0.05)]); soft > full/3 {
		t.Errorf("with Attack 0.5, the first 50 ms are at %.0f, against %.0f without", soft, full)
	}

	// With Ring, a note sounds on into the silence after it, and the last
	// note rings on over the start of the tune.
	dry := render(TuneSpec{Voices: []Voice{{Notes: "a4 . . ."}}})
	rung := render(TuneSpec{Voices: []Voice{{Ring: 0.5, Notes: "a4 . . ."}}})
	if loudness(dry[at(0.6):at(0.8)]) != 0 {
		t.Error("without Ring, the silence after a note isn't silent")
	}
	if loudness(rung[at(0.6):at(0.8)]) == 0 {
		t.Error("with Ring 0.5, the note doesn't ring on after it ends")
	}
	if len(rung) != len(dry) {
		t.Errorf("Ring changed the tune's length from %d samples to %d", len(dry), len(rung))
	}
	last := render(TuneSpec{Voices: []Voice{{Ring: 1, Notes: ". . . a4"}}})
	if loudness(last[:at(0.2)]) == 0 {
		t.Error("with Ring 1, the last note doesn't ring on over the start of the loop")
	}

	// LowPass takes the edge off a square wave: much less of it changes
	// from one sample to the next.
	square := render(TuneSpec{Voices: []Voice{{Notes: "a4/4"}}})
	soft := render(TuneSpec{Voices: []Voice{{LowPass: 800, Notes: "a4/4"}}})
	if a, b := harshness(square), harshness(soft); b > a/4 {
		t.Errorf("the square wave's harshness is %.4f, and %.4f with LowPass 800; want a quarter or less", a, b)
	}

	// Detune adds a second wave that beats against the first: a sine wave's
	// loudness rises and falls, where on its own it stays the same.
	if steady, beating := beats(render(TuneSpec{Voices: []Voice{{Wave: WaveSine, Notes: "a4/4"}}})),
		beats(render(TuneSpec{Voices: []Voice{{Wave: WaveSine, Detune: 50, Notes: "a4/4"}}})); steady > 1.2 || beating < 3 {
		t.Errorf("loudness varies %.2f times alone and %.2f times with Detune 50; want about 1, and 3 or more", steady, beating)
	}

	// Reverb leaves echoes after a note, and the echoes at the end of the
	// tune carry over into its start.
	room := render(TuneSpec{Reverb: 0.6, Voices: []Voice{{Notes: "a4/0.5 . . ."}}})
	if loudness(room[at(0.4):at(0.6)]) == 0 {
		t.Error("with Reverb 0.6, the silence after a note has no echo")
	}
	if len(room) != len(render(TuneSpec{Voices: []Voice{{Notes: "a4/0.5 . . ."}}})) {
		t.Error("Reverb changed the tune's length")
	}
	ending := render(TuneSpec{Reverb: 0.6, Voices: []Voice{{Notes: ". . . a4"}}})
	if loudness(ending[:at(0.1)]) == 0 {
		t.Error("with Reverb 0.6, the echoes of the last note don't carry over into the start of the loop")
	}
}

// loudness returns the root mean square of samples.
func loudness(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	sum := 0.0
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// harshness returns how much of the samples' energy is in fast changes from
// one sample to the next: high in a bare square wave, low in a soft sound.
func harshness(samples []int16) float64 {
	change, energy := 0.0, 0.0
	for i := 1; i < len(samples); i++ {
		d := float64(samples[i]) - float64(samples[i-1])
		change += d * d
		energy += float64(samples[i]) * float64(samples[i])
	}
	return change / energy
}

// beats returns how many times louder the loudest hundredth of a second of
// samples is than the quietest, ignoring the first and last tenths.
func beats(samples []int16) float64 {
	window := soundSampleRate / 100
	loudest, quietest := 0.0, math.Inf(1)
	for at := len(samples) / 10; at+window < len(samples)*9/10; at += window {
		l := loudness(samples[at : at+window])
		loudest, quietest = max(loudest, l), min(quietest, l)
	}
	return loudest / max(quietest, 1)
}

// TestMusicPreload checks that Preload makes a tune ahead of its first Play,
// once, and reports a mistake as playing would.
func TestMusicPreload(t *testing.T) {
	takeError() // nothing left over from other tests
	tune := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 e4 g4 c5"}}})
	tune.Preload()
	if tune.ready == nil || tune.format != ".wav" {
		t.Fatalf("after Preload: %d bytes ready, format %q; want the tune as a WAV", len(tune.ready), tune.format)
	}
	made := &tune.ready[0]
	tune.Preload()
	if &tune.ready[0] != made {
		t.Error("a second Preload made the tune again")
	}
	if err := tune.Err(); err != nil {
		t.Errorf("Err after Preload: %v", err)
	}
	if err := takeError(); err != nil {
		t.Errorf("Preload of a good tune reported %v", err)
	}

	broken := NewTune(TuneSpec{Voices: []Voice{{Notes: "c4 q1"}}})
	broken.Preload()
	if err := takeError(); err == nil || !strings.Contains(err.Error(), "golib.NewTune") {
		t.Errorf("Preload of a broken tune reported %v; want its mistake, for Run", err)
	}
	if broken.Err() == nil {
		t.Error("Err is nil after Preload found a mistake")
	}

	missing := NewMusic("music/missing.ogg")
	missing.Preload()
	if err := takeError(); err == nil || !strings.Contains(err.Error(), "golib.NewMusic") {
		t.Errorf("Preload of a missing file reported %v", err)
	}

	// With a sound device, Play streams what Preload made.
	if audio.isReady() {
		return
	}
	SetVolume(0)
	audio.open()
	t.Cleanup(func() {
		audio.close()
		SetVolume(1)
	})
	if !audio.isReady() {
		return // this machine has no sound device
	}
	tune.Play()
	if err := takeError(); err != nil {
		t.Fatal(err)
	}
	if !tune.Playing() || tune.ready != nil {
		t.Errorf("after Play: playing %v, %d bytes still waiting; want it playing what Preload made", tune.Playing(), len(tune.ready))
	}
	tune.Stop()
}
