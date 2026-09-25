//go:build !js && !darwin

package device

// On Windows and Linux the window needs none of what macOS needs in
// raylib_darwin.go.

// MeasureWindow is nothing here: raylib measures the window right.
func MeasureWindow() {}

// HasSystemFullscreen reports false: fullscreen here is a borderless window
// that golib sizes to cover the monitor, which keeps its resolution.
func HasSystemFullscreen() bool {
	return false
}

// SetSystemFullscreen is never called here: see HasSystemFullscreen.
func SetSystemFullscreen(on bool) {}

// SetAppIcon is nothing here: golib puts the icon inside Windows
// executables, and Linux builds don't carry one yet.
func SetAppIcon(png []byte) {}

// FullscreenLost is always false here: fullscreen is a window GoLib sizes
// itself, and nothing but the game takes it away again. A browser is where a
// player leaves fullscreen on their own, with Esc or a phone's gesture, and
// macOS, with the window's green button.
func FullscreenLost() bool {
	return false
}
