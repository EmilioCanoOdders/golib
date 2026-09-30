package golib

import (
	"math"
	"testing"
)

// OpenURL opens web pages and mails to write, and refuses anything else,
// which a game could otherwise run by mistake.
func TestOpenURLOpensOnlyLinks(t *testing.T) {
	for url, ok := range map[string]bool{
		"https://example.com/page":     true,
		"http://example.com":           true,
		"mailto:someone@example.com":   true,
		"MAILTO:someone@example.com":   true,
		"https://":                     false,
		"file:///etc/passwd":           false,
		"calc.exe":                     false,
		"https://example.com/a b":      false,
		`https://example.com/"; rm -r`: false,
		"":                             false,
	} {
		if err := OpenURL(url); (err == nil) != ok {
			t.Errorf("OpenURL(%q): %v, want it to open: %v", url, err, ok)
		}
	}
}

// A window moved to another monitor sits in its middle, no larger than it.
func TestAWindowSitsCenteredOnItsMonitor(t *testing.T) {
	if x, y, w, h := centered(1280, 720, 1920, 0, 2560, 1440); x != 1920+640 || y != 360 || w != 1280 || h != 720 {
		t.Errorf("a 1280 by 720 window on a 2560 by 1440 monitor at 1920, 0: %d, %d, %d by %d", x, y, w, h)
	}
	if x, y, w, h := centered(4000, 3000, 0, 0, 1920, 1080); x != 0 || y != 0 || w != 1920 || h != 1080 {
		t.Errorf("a window larger than its monitor: %d, %d, %d by %d, want the monitor", x, y, w, h)
	}
}

// FPS counts the frames of the last second.
func TestFPSCountsTheFramesOfTheLastSecond(t *testing.T) {
	framesPerSecond.Store(0)
	var c frameCounter
	for i := range 91 {
		c.count(10 + float64(i)/90)
	}
	if got := FPS(); got != 90 {
		t.Errorf("90 frames in a second: FPS %d", got)
	}
}

// The window's requests keep within their limits, and are ignored when they
// make no sense.
func TestTheWindowsRequestsKeepWithinLimits(t *testing.T) {
	SetFrameRate(1000)
	if got := frameRateWanted.Swap(0); got != maxFrameRate {
		t.Errorf("SetFrameRate(1000) asks for %d", got)
	}
	SetFrameRate(5)
	if got := frameRateWanted.Swap(0); got != minFrameRate {
		t.Errorf("SetFrameRate(5) asks for %d", got)
	}
	SetWindowSize(0, 600)
	SetMonitor(-1)
	if windowSizeWanted.Load() != 0 || monitorWanted.Load() != 0 {
		t.Error("a window 0 wide or monitor -1 was asked for")
	}
	SetWindowSize(1280, 720)
	if size := windowSizeWanted.Swap(0); size>>32 != 1280 || uint32(size) != 720 {
		t.Errorf("SetWindowSize(1280, 720) asks for %d by %d", size>>32, uint32(size))
	}
	if Monitors() != nil && len(Monitors()) > 0 && !windowOpen.Load() {
		t.Error("monitors without a window")
	}
}

func TestDisplayScaleIsOneUntilRunLooks(t *testing.T) {
	if got := DisplayScale(); got != 1 {
		t.Errorf("DisplayScale() is %v with no window, want 1", got)
	}
	displayScale.Store(math.Float32bits(2))
	defer displayScale.Store(0)
	if got := DisplayScale(); got != 2 {
		t.Errorf("DisplayScale() is %v after a Retina screen was seen, want 2", got)
	}
}
