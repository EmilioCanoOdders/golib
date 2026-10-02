package golib

import (
	"errors"
	"math"
	"strings"
	"sync/atomic"

	"golib/internal/device"
)

// The window's place and pace, for a game's graphics settings: which monitor
// it is on, how many of the screen's pixels its units are, its size, how many frames a second it draws, how many it does,
// and opening a link. Like SetFullscreen, what a game asks for is applied at
// the start of the next frame.

// Monitor is a monitor connected to the machine: its name, as the system
// gives it, its size in pixels, and how many times a second it refreshes.
type Monitor struct {
	Name        string
	Width       int
	Height      int
	RefreshRate int // 0 where the system doesn't say, such as in a browser
}

// What the game asked for, and what Run last saw, kept in step at the start of
// every frame.
var (
	monitorWanted    atomic.Int32  // the monitor SetMonitor asked for, plus one; 0 for none
	windowSizeWanted atomic.Uint64 // the width and height SetWindowSize asked for, packed; 0 for none
	frameRateWanted  atomic.Int32  // the frames a second SetFrameRate asked for; 0 for no change
	monitorsSeen     atomic.Pointer[[]Monitor]
	monitorNow       atomic.Int32
	displayScale     atomic.Uint32 // math.Float32bits of DisplayScale's answer; 0 before Run looks
	framesPerSecond  atomic.Int32
)

// The frame rates SetFrameRate takes, and the one a game starts at.
const (
	minFrameRate = 15
	maxFrameRate = 360
)

// Monitors returns the monitors connected to the machine, as Run last looked,
// once a second. A game shows them in its settings, to choose one with
// SetMonitor. In a browser there is one, the screen the page is on. Under golib
// shot and in tests there is none.
func Monitors() []Monitor {
	if seen := monitorsSeen.Load(); seen != nil {
		return append([]Monitor(nil), (*seen)...)
	}
	return nil
}

// CurrentMonitor returns the monitor the game's window is on, as an index into
// Monitors: 0 with one monitor, in a browser, and where there is none.
func CurrentMonitor() int {
	return int(monitorNow.Load())
}

// DisplayScale returns how many of the screen's own pixels, across, one unit
// of Monitors' sizes, SetWindowSize and the window's size is, on the monitor
// the window is on, as Run last looked, once a second: 2 on a Mac's Retina
// screen, where macOS measures in points two pixels wide, and 1 on other
// Macs, on Windows and on Linux, which measure in pixels; in a browser, its
// devicePixelRatio. A game's settings multiply by it to show a window's size
// in the screen's pixels:
//
//	scale := golib.DisplayScale()
//	label := fmt.Sprintf("%d × %d", int(1920*scale), int(1080*scale)) // "3840 × 2160" on a Retina Mac
//
// The screen a game draws on keeps its size either way. It is 1 under golib
// shot and in tests.
func DisplayScale() float32 {
	if bits := displayScale.Load(); bits != 0 {
		return math.Float32frombits(bits)
	}
	return 1
}

// SetMonitor moves the game's window to monitor i of Monitors, from the next
// frame: in a window, centered on it at the size it has; in fullscreen,
// covering it. On macOS, whose fullscreen is the system's own, a game in
// fullscreen leaves it, moves, and enters it again on that monitor. An index
// no monitor has is ignored, and so is the call in a browser, where the page
// stays where it is.
func SetMonitor(i int) {
	if i >= 0 {
		monitorWanted.Store(int32(i) + 1)
	}
}

// SetWindowSize makes the game's window width by height pixels, the area it
// draws in, from the next frame, centered on its monitor and no larger than
// it. In fullscreen it is the size the window comes back to. The screen the
// game draws on keeps its size, and Run scales it to fit the window, unless
// Config.WindowScale makes it this size divided by the scale, the way to offer
// resolutions, or Config.FillWindow gives it the window's shape.
// A size below 1 is ignored, and so is the call in a browser, where the canvas
// follows the page.
//
// The system may make the window smaller than asked, to fit it between its
// menu bar, its taskbar or Dock, and the window's own title bar: on a 2560 by
// 1440 Mac, a window asked 1440 high gets 1296. With Config.PixelArt, the
// screen then drops to the next whole size, with black borders around it. A
// game that offers window sizes keeps them to about four fifths of the
// monitor, as Run's first window is.
func SetWindowSize(width, height int) {
	if width >= 1 && height >= 1 {
		windowSizeWanted.Store(uint64(width)<<32 | uint64(uint32(height)))
	}
}

// SetFrameRate draws fps frames a second at most, from the next frame, from
// 15 to 360; a game starts at 60. Updates stay at 60 a second of game time
// whatever the frame rate: at 30, every frame runs two updates; above 60,
// some frames run none and draw the same state again. A frame rate higher
// than the monitor refreshes wastes nothing but power.
func SetFrameRate(fps int) {
	frameRateWanted.Store(int32(max(minFrameRate, min(maxFrameRate, fps))))
}

// FPS returns how many frames Run drew in the last second, for a game to show:
//
//	if showFPS {
//		screen.DrawText(fmt.Sprint(golib.FPS(), " FPS"), 4, 4, 10, golib.White)
//	}
//
// It is 0 under golib shot and in tests, and during the game's first second.
func FPS() int {
	return int(framesPerSecond.Load())
}

// errBadURL is why OpenURL refuses a link.
var errBadURL = errors.New("golib.OpenURL: only links that start with https://, http:// or mailto: open, with no spaces or quotes")

// OpenURL opens url in the program the player's system opens such links
// with: a web page in the browser, or a mail to write, for a mailto: link,
// such as a game's contact in its About screen:
//
//	golib.OpenURL("mailto:someone@example.com")
//
// Only links that start with https://, http:// or mailto: open, and with no
// spaces or quotes, so a game can't run anything else by mistake: it returns
// an error for any other. In a browser, a web page opens in a new tab. Under
// golib shot and in tests, nothing opens.
func OpenURL(url string) error {
	if !validURL(url) {
		return errBadURL
	}
	if !windowOpen.Load() {
		return nil
	}
	device.OpenURL(url)
	return nil
}

// validURL reports whether url is a link OpenURL opens.
func validURL(url string) bool {
	ok := false
	for _, scheme := range []string{"https://", "http://", "mailto:"} {
		if strings.HasPrefix(strings.ToLower(url), scheme) && len(url) > len(scheme) {
			ok = true
		}
	}
	return ok && !strings.ContainsAny(url, " \t\n\"'`\\")
}

// windowOpen is whether Run has a window open for the game, where OpenURL
// can open a link.
var windowOpen atomic.Bool

// frameCounter counts the frames Run draws, for FPS.
type frameCounter struct {
	since  float64 // when the second being counted began
	frames int
}

// count counts a frame drawn at now, in seconds, and every second hands what
// it counted to FPS.
func (c *frameCounter) count(now float64) {
	if c.since == 0 {
		c.since = now // the frame that starts the count
		return
	}
	c.frames++
	if elapsed := now - c.since; elapsed >= 1 {
		framesPerSecond.Store(int32(float64(c.frames)/elapsed + 0.5))
		c.since, c.frames = now, 0
	}
}

// lookAtMonitors notes the monitors connected and the one the window is on,
// for Monitors and CurrentMonitor.
func lookAtMonitors() {
	count := device.MonitorCount()
	monitors := make([]Monitor, 0, count)
	for i := range count {
		name, _, _, width, height, refresh := device.Monitor(i)
		monitors = append(monitors, Monitor{Name: name, Width: width, Height: height, RefreshRate: refresh})
	}
	monitorsSeen.Store(&monitors)
	monitorNow.Store(int32(max(0, min(device.CurrentMonitor(), count-1))))
	if scale := device.DisplayScale(); scale > 0 {
		displayScale.Store(math.Float32bits(scale))
	}
}

// centered returns where a window of width by height pixels goes on a monitor
// whose top-left corner is at x, y and whose size is monitorWidth by
// monitorHeight: in its middle, no larger than it.
func centered(width, height, x, y, monitorWidth, monitorHeight int) (int, int, int, int) {
	width, height = min(width, monitorWidth), min(height, monitorHeight)
	return x + (monitorWidth-width)/2, y + (monitorHeight-height)/2, width, height
}

// placeWindow applies what SetFrameRate, SetWindowSize and SetMonitor asked
// for since the last frame, to the window w.
func (w *window) placeWindow(held bool) {
	if fps := frameRateWanted.Swap(0); fps > 0 {
		device.SetTargetFPS(int(fps))
	}
	if size := windowSizeWanted.Swap(0); size != 0 {
		width, height := int(size>>32), int(uint32(size))
		if w.fullscreen {
			w.width, w.height = width, height // for when it leaves fullscreen
		} else {
			x, y, monitorWidth, monitorHeight := device.MonitorBounds()
			x, y, width, height = centered(width, height, x, y, monitorWidth, monitorHeight)
			device.SetWindowSize(width, height)
			device.SetWindowPosition(x, y)
		}
	}
	if wanted := int(monitorWanted.Swap(0)) - 1; wanted >= 0 && wanted < device.MonitorCount() && wanted != device.CurrentMonitor() {
		w.moveTo = wanted + 1
	}
	if w.moveTo == 0 {
		return
	}
	// macOS's own fullscreen can't move: leave it first, and move once the
	// window is back, after which apply enters it again.
	if w.fullscreen && device.HasSystemFullscreen() {
		if !held {
			device.SetSystemFullscreen(false)
			w.fullscreen = false
		}
		return
	}
	_, x, y, monitorWidth, monitorHeight, _ := device.Monitor(w.moveTo - 1)
	if w.fullscreen {
		device.SetWindowPosition(x, y)
		device.SetWindowSize(monitorWidth, monitorHeight)
		// Leaving fullscreen puts the window back on this monitor.
		w.x, w.y, w.width, w.height = centered(w.width, w.height, x, y, monitorWidth, monitorHeight)
	} else {
		width, height := device.WindowSize()
		x, y, width, height = centered(width, height, x, y, monitorWidth, monitorHeight)
		device.SetWindowPosition(x, y)
		device.SetWindowSize(width, height)
	}
	w.moveTo = 0
}
