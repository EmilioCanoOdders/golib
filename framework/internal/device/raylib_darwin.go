//go:build !js

package device

import (
	"github.com/ebitengine/purego/objc"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// macOS needs two things of the raylib backend that Windows and Linux don't,
// both about the window. They were found on a Retina iMac, 2560 by 1440
// points on a 5120 by 2880 screen:
//
//   - While the window opens, and golib.Run enlarges it, AppKit can measure
//     it in Retina pixels, twice its size in points, and raylib keeps that
//     size although it draws in points. The screen then shows twice too
//     large, only its bottom-left quarter in the window, until the player
//     resizes the window. MeasureWindow makes raylib measure it again.
//   - A borderless window the size of the monitor, GoLib's fullscreen
//     elsewhere, sits below the menu bar here, so its bottom falls off the
//     monitor. macOS has a fullscreen of its own, the one the window's green
//     button enters: a space of its own, without the menu bar or the Dock,
//     that Cmd+Tab leaves like any other. SystemFullscreen uses it.
//
// Both ask AppKit about the window through purego's Objective-C calls, as
// raylib-go reaches raylib through purego.

// The Objective-C messages the backend sends the window, registered once.
var (
	selContentView          = objc.RegisterName("contentView")
	selFrame                = objc.RegisterName("frame")
	selConvertRectToBacking = objc.RegisterName("convertRectToBacking:")
	selStyleMask            = objc.RegisterName("styleMask")
	selToggleFullScreen     = objc.RegisterName("toggleFullScreen:")
)

// nsWindowStyleMaskFullScreen is the bit of an NSWindow's styleMask that is
// on while the window is in macOS's fullscreen.
const nsWindowStyleMaskFullScreen = 1 << 14

// nsRect is AppKit's NSRect.
type nsRect struct{ X, Y, Width, Height float64 }

// windowMeasured is whether MeasureWindow has seen raylib keep the window's
// own size. It checks no more after that: the wrong size only comes while
// the window opens.
var windowMeasured bool

// MeasureWindow makes raylib measure the window again while the size it keeps
// isn't the window's own, which happens on macOS as the window opens (see the
// top of this file). It waits for AppKit to measure the window in points,
// since raylib would measure it wrong again before that, and for the window
// to leave macOS's fullscreen, whose own resize measures it again. Run calls
// it at the start of every frame.
func MeasureWindow() {
	if windowMeasured || inSystemFullscreen() {
		return
	}
	view := nsWindow().Send(selContentView)
	points := objc.Send[nsRect](view, selFrame)
	pixels := objc.Send[nsRect](view, selConvertRectToBacking, points)
	if pixels.Width != points.Width || pixels.Height != points.Height {
		return // AppKit still counts Retina pixels
	}
	width, height := int(points.Width), int(points.Height)
	if rl.GetScreenWidth() == width && rl.GetScreenHeight() == height &&
		rl.GetRenderWidth() == width && rl.GetRenderHeight() == height {
		windowMeasured = true
		return
	}
	// What the player's own resize does: GLFW measures the window again at
	// every new size, one point wider and then the size it has.
	rl.SetWindowSize(width+1, height)
	rl.SetWindowSize(width, height)
}

// SystemFullscreen enters or leaves macOS's own fullscreen, and reports true:
// golib uses it instead of a borderless window (see the top of this file).
// The window slides into place over the next frames. A hidden window, as in
// the framework's tests, keeps the borderless one, and it reports false.
func SystemFullscreen(on bool) bool {
	if rl.IsWindowHidden() {
		return false
	}
	if on != inSystemFullscreen() {
		nsWindow().Send(selToggleFullScreen, objc.ID(0))
	}
	return true
}

// FullscreenLost reports whether the window is out of macOS's fullscreen
// while golib had put it there: the player can leave it themselves, with the
// window's green button.
func FullscreenLost() bool {
	return !rl.IsWindowHidden() && !inSystemFullscreen()
}

// inSystemFullscreen reports whether the window is in macOS's fullscreen, or
// sliding into it.
func inSystemFullscreen() bool {
	return objc.Send[uint](nsWindow(), selStyleMask)&nsWindowStyleMaskFullScreen != 0
}

// nsWindow returns the game's window as AppKit's NSWindow.
func nsWindow() objc.ID {
	return objc.ID(uintptr(rl.GetWindowHandle()))
}
