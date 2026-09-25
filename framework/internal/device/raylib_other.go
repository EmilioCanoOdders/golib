//go:build !js && !darwin

package device

// On Windows and Linux the window needs none of what macOS needs in
// raylib_darwin.go.

// MeasureWindow is nothing here: raylib measures the window right.
func MeasureWindow() {}

// SystemFullscreen reports false: fullscreen here is a borderless window
// that golib sizes to cover the monitor, which keeps its resolution.
func SystemFullscreen(on bool) bool {
	return false
}

// FullscreenLost is always false here: fullscreen is a window GoLib sizes
// itself, and nothing but the game takes it away again. A browser is where a
// player leaves fullscreen on their own, with Esc or a phone's gesture, and
// macOS, with the window's green button.
func FullscreenLost() bool {
	return false
}
