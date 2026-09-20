// niebla is a game made with GoLib. DESIGN.md says what it is.
//
// Read it in this order:
//
//   - main.go (this file) starts the game: main calls golib.Run with the
//     first scene. The screen's size and the game's colors are here.
//   - play.go is the play scene: Update turns input into actions and
//     sends one Tick per update, Draw draws the region through the
//     camera, and the camera and the selection live here, never
//     serialized.
//   - state.go holds the simulation's state, the one serializable value
//     the whole game is, and newGame, which deals the starting region.
//   - actions.go holds the actions (Tick, SendRobot, RecallRobot,
//     MarkBuilding, QueueRobot) and Apply, the only door into the state.
//   - sim_robots.go holds the robots' rules and their tuning: what a
//     robot does each tick — carry home, mind the tank, finish loading,
//     build jobs first, its post second.
//   - sim_buildings.go holds the buildings' rules and their tuning:
//     blueprints, placement and safe zones, storage caps, refueling,
//     the factories' robot works.
//   - region.go holds the hand-made region and the isometric projection,
//     as plain Go with no drawing, so that region_test.go can test it.
//   - things.go holds what a tile holds: the things' snapshot out of the
//     state and the SI units, as plain Go with no drawing.
//   - catalog.go is the entity database: per thing type, its name, its
//     color and its card's lines, with a stable-color fallback for types
//     it has no entry for yet.
//   - markup.go writes text in colors: the "[name]...[/]" markup.
//   - inspect.go lays out and paints the tile inspection panel, with the
//     cards' buttons.
//   - draw.go paints the region and the robots.
//   - world_test.go drives the simulation directly, no window needed.
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
	scarColor        = golib.Color{R: 54, G: 60, B: 72, A: 255}
	robotColor       = golib.Color{R: 208, G: 216, B: 228, A: 255}
	robotDarkColor   = golib.Color{R: 58, G: 66, B: 80, A: 255}
	robotShadowColor = golib.Color{R: 40, G: 46, B: 58, A: 90}
	coreColor        = golib.Color{R: 32, G: 36, B: 46, A: 255}
	coreGlowColor    = golib.Color{R: 255, G: 244, B: 214, A: 255}
	bubbleColor      = golib.Color{R: 168, G: 216, B: 255, A: 46}
	bubbleEdgeColor  = golib.Color{R: 190, G: 226, B: 255, A: 130}
	textColor        = golib.Color{R: 40, G: 44, B: 54, A: 255}

	// The buildings. Each kind has a light face for the sun side and a
	// dark one for the shade, in the colony's cold palette; the
	// protector keeps the bubble's blue.
	factoryColor   = golib.Color{R: 92, G: 188, B: 174, A: 255}
	factoryDark    = golib.Color{R: 52, G: 118, B: 110, A: 255}
	chargerColor   = golib.Color{R: 240, G: 202, B: 96, A: 255}
	chargerDark    = golib.Color{R: 150, G: 120, B: 44, A: 255}
	siloColor      = golib.Color{R: 204, G: 164, B: 100, A: 255}
	siloDark       = golib.Color{R: 128, G: 100, B: 56, A: 255}
	warehouseColor = golib.Color{R: 168, G: 156, B: 208, A: 255}
	warehouseDark  = golib.Color{R: 104, G: 96, B: 140, A: 255}

	protectorColor       = golib.Color{R: 150, G: 202, B: 246, A: 255}
	protectorDark        = golib.Color{R: 80, G: 120, B: 168, A: 255}
	protectorBubbleColor = golib.Color{R: 168, G: 216, B: 255, A: 26}
	protectorEdgeColor   = golib.Color{R: 168, G: 216, B: 255, A: 90}

	// The inspection panel: a dark plate with light text, so the cards'
	// colors read over any ground. The picked tile keeps the core's warm
	// white.
	panelColor       = golib.Color{R: 24, G: 27, B: 35, A: 235}
	panelEdgeColor   = golib.Color{R: 190, G: 226, B: 255, A: 70}
	panelTextColor   = golib.Color{R: 232, G: 236, B: 244, A: 255}
	panelDimColor    = golib.Color{R: 148, G: 156, B: 172, A: 255}
	pickedTileColor  = golib.Color{R: 255, G: 244, B: 214, A: 255}
	hoveredTileColor = golib.Color{R: 255, G: 255, B: 255, A: 90}
	buttonColor      = golib.Color{R: 34, G: 39, B: 50, A: 255}
	buttonEdgeColor  = golib.Color{R: 190, G: 226, B: 255, A: 120}
	buttonHoverColor = golib.Color{R: 52, G: 60, B: 76, A: 255}
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
