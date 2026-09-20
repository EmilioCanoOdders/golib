// niebla is a game made with GoLib. DESIGN.md says what it is.
//
// Read it in this order:
//
//   - main.go (this file) starts the game: main calls golib.Run with the
//     first scene. The screen's size and the game's colors are here.
//   - play.go is the play scene: Update answers the keys and drives the
//     camera, Draw draws the region through it.
//   - region.go holds the hand-made region and the isometric projection, as
//     plain Go with no drawing, so that region_test.go can test it.
//   - draw.go paints the region.
//
// games/platformer is a complete example, with a title, pause and win scenes,
// and a world larger than the screen, seen through a golib.Camera.
package main

import (
	_ "embed"
	"log"

	"golib"
)

// The screen's size in pixels: half of 2K, so a 2K monitor scales it by two
// whole numbers. With PixelArt the scaling stays in whole numbers and
// without smoothing, so shapes and text keep their pixels. GoLib scales it
// to any window; 2K/4 (640x360) would be the step for pixel-art sprites.
const (
	screenWidth  = 1280
	screenHeight = 720
)

// The game's colors, in one place so the look is easy to change: cold ground,
// amber oil, lilac veins, pale fog.
var (
	fogColor         = golib.Color{R: 186, G: 192, B: 200, A: 255}
	fogBandColor     = golib.Color{R: 225, G: 228, B: 234, A: 255}
	groundColor      = golib.Color{R: 74, G: 84, B: 98, A: 255}
	groundShadeColor = golib.Color{R: 66, G: 76, B: 90, A: 255}
	oilColor         = golib.Color{R: 224, G: 156, B: 58, A: 255}
	oilDarkColor     = golib.Color{R: 166, G: 112, B: 42, A: 255}
	lilacColor       = golib.Color{R: 186, G: 148, B: 255, A: 255}
	lilacDarkColor   = golib.Color{R: 138, G: 102, B: 208, A: 255}
	lilacLightColor  = golib.Color{R: 214, G: 188, B: 255, A: 255}
	rockColor        = golib.Color{R: 120, G: 126, B: 138, A: 255}
	rockLightColor   = golib.Color{R: 144, G: 150, B: 162, A: 255}
	bushColor        = golib.Color{R: 92, G: 106, B: 92, A: 255}
	bushLightColor   = golib.Color{R: 114, G: 130, B: 110, A: 255}
	coreColor        = golib.Color{R: 32, G: 36, B: 46, A: 255}
	coreGlowColor    = golib.Color{R: 255, G: 244, B: 214, A: 255}
	bubbleColor      = golib.Color{R: 168, G: 216, B: 255, A: 46}
	bubbleEdgeColor  = golib.Color{R: 190, G: 226, B: 255, A: 130}
	textColor        = golib.Color{R: 40, G: 44, B: 54, A: 255}
)

// The monitor filters, run over the whole picture after every Draw, in this
// order: a glow, a whisper of a tube screen and a soft rounding of the
// pixels. Both the glow and the CRT are adapted from games/asteroids, turned
// down. The player switches all three with F2.
var (
	//go:embed shaders/glow.fs
	glowSource string

	//go:embed shaders/crt.fs
	crtSource string

	//go:embed shaders/soft.fs
	softSource string
)

// The screen effect settings, sent to the shaders as uniforms. Softer than
// games/asteroids' 2.4 and 0.2.
const (
	glowStrength = 0.9  // how bright the halo around the core and bubble is
	crtCurvature = 0.08 // how much the picture bulges, like a tube; 0 is flat
)

func main() {
	config := golib.Config{
		Title: "niebla", Width: screenWidth, Height: screenHeight,
		PixelArt: true,
		// The game waits while the player is in another program. Take it out
		// for a game that should keep playing in the background.
		PauseUnfocused: true,
	}
	if err := golib.Run(newPlayScene(), config); err != nil {
		log.Fatal(err)
	}
}
