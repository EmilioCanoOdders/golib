package main

import (
	"math"

	"golib"
)

// Camera tuning. Zoom rests on whole steps so shapes and, later, pixel art
// stay square, and glides between them; at zoomIn a robot's 1 u fills about
// 40 screen pixels. Panning keeps its speed on the screen, not in the world,
// so it glides the same at every zoom.
const (
	zoomOut = 1
	zoomIn  = 4

	panSpeed = 480 // screen pixels per second

	zoomGlide = 0.1 // seconds for the zoom to close most of the way to its next step
)

// playScene shows the region through a camera. The camera is view, not
// state: it lives here, outside the simulation, and never gets serialized.
type playScene struct {
	glow, crt, soft *golib.Shader
	filterOn        bool
	camera          *golib.Camera
	zoom            float32       // the zoom on screen, gliding toward zoomLevel
	zoomLevel       float32       // the whole-step zoom the wheel last asked for
	anchorWorld     golib.Vector2 // while gliding, the point kept under the cursor
	anchorScreen    golib.Vector2
	dragging        bool          // the right button is down and moving the view
	dragFrom        golib.Vector2 // where the cursor stood at the last drag update

	picked       bool // a tile is selected and shows its panel
	pickedCol    int  // the selected tile
	pickedRow    int
	expanded     map[string]bool // which cards stand open, by thing ID
	hovering     bool            // the pointer is over the region
	hoverCol     int             // the tile under the pointer
	hoverRow     int
	rightWasDown bool
	rightFrom    golib.Vector2 // where the right button went down
}

// newPlayScene turns the monitor filters on: the glow runs first, so the CRT
// scans the glowing picture, and the soft rounding goes last, so it rounds
// the whole result.
func newPlayScene() *playScene {
	s := &playScene{
		glow:     golib.NewShader(glowSource),
		crt:      golib.NewShader(crtSource),
		soft:     golib.NewShader(softSource),
		expanded: map[string]bool{},
	}
	s.glow.SetUniform("strength", glowStrength)
	s.crt.SetUniform("curvature", crtCurvature)
	s.soft.SetUniform("amount", 0.35)
	s.setFilters(true)
	s.camera = golib.NewCamera(screenWidth, screenHeight)
	s.camera.Bounds = regionOnScreen()
	s.zoom, s.zoomLevel = zoomOut, zoomOut
	s.camera.Zoom = s.zoom
	s.camera.Snap()
	return s
}

func (s *playScene) setFilters(on bool) {
	s.filterOn = on
	if on {
		golib.SetPostProcess(s.glow, s.crt, s.soft)
	} else {
		golib.SetPostProcess()
	}
}

func (s *playScene) Update(input *golib.Input, dt float32) {
	// No key quits by itself, not even Esc: the game calls golib.Quit when
	// it wants to end.
	if input.KeyPressed(golib.KeyEscape) || input.GamepadPressed(0, golib.GamepadBack) {
		golib.Quit()
		return
	}
	// F11 or Alt+Enter switches fullscreen. The screen keeps its size: GoLib
	// scales it to fit.
	altEnter := (input.KeyDown(golib.KeyLeftAlt) || input.KeyDown(golib.KeyRightAlt)) &&
		input.KeyPressed(golib.KeyEnter)
	if input.KeyPressed(golib.KeyF11) || altEnter {
		golib.SetFullscreen(!golib.IsFullscreen())
	}
	// F2 turns the monitor filters on and off.
	if input.KeyPressed(golib.KeyF2) {
		s.setFilters(!s.filterOn)
	}
	s.updateCamera(input, dt)
	s.updateInspection(input)
}

// updateCamera pans with WASD, the arrows or the left stick, drags with the
// right button, glides the zoom in whole steps with the wheel, and keeps the
// view inside the region.
func (s *playScene) updateCamera(input *golib.Input, dt float32) {
	s.zoomCamera(input, dt)
	s.dragCamera(input)
	s.panCamera(input, dt)
	s.camera.Update(dt)
}

func (s *playScene) panCamera(input *golib.Input, dt float32) {
	dx, dy := float32(0), float32(0)
	if input.KeyDown(golib.KeyW) || input.KeyDown(golib.KeyUp) {
		dy--
	}
	if input.KeyDown(golib.KeyS) || input.KeyDown(golib.KeyDown) {
		dy++
	}
	if input.KeyDown(golib.KeyA) || input.KeyDown(golib.KeyLeft) {
		dx--
	}
	if input.KeyDown(golib.KeyD) || input.KeyDown(golib.KeyRight) {
		dx++
	}
	if stickX, stickY := input.GamepadLeftStick(0); stickX != 0 || stickY != 0 {
		dx, dy = stickX, stickY
	}
	walked := math.Hypot(float64(dx), float64(dy))
	if walked == 0 {
		return
	}
	step := panSpeed / s.zoom * dt
	s.camera.Target.X += dx / float32(walked) * step
	s.camera.Target.Y += dy / float32(walked) * step
}

// zoomCamera glides the zoom to the whole step the wheel asks for, keeping
// the point under the cursor under it while it moves. The in-between zooms
// only exist while gliding: at rest the zoom is a whole number again.
func (s *playScene) zoomCamera(input *golib.Input, dt float32) {
	if notches := input.MouseWheel(); notches != 0 {
		level := golib.Clamp(s.zoomLevel+float32(int(notches)), zoomOut, zoomIn)
		if level != s.zoomLevel {
			s.zoomLevel = level
			mx, my := input.MousePosition()
			s.anchorWorld = s.camera.ToWorld(mx, my)
			s.anchorScreen = golib.Vector2{X: mx, Y: my}
		}
	}
	if s.zoom == s.zoomLevel {
		return
	}
	keep := float32(math.Exp(float64(-dt / zoomGlide)))
	s.zoom += (s.zoomLevel - s.zoom) * (1 - keep)
	if math.Abs(float64(s.zoomLevel-s.zoom)) < 0.001 {
		s.zoom = s.zoomLevel
	}
	s.camera.Zoom = s.zoom
	// Where the center must sit for the anchor to stay under the cursor:
	// the anchor minus the cursor's offset from the middle.
	s.camera.Target = golib.Vector2{
		X: s.anchorWorld.X - (s.anchorScreen.X-screenWidth/2)/s.zoom,
		Y: s.anchorWorld.Y - (s.anchorScreen.Y-screenHeight/2)/s.zoom,
	}
}

// dragCamera moves the view with the right button, the way a hand drags a
// map: the grab moves the ground with the cursor's opposite.
func (s *playScene) dragCamera(input *golib.Input) {
	if !input.MouseDown(golib.MouseRight) {
		s.dragging = false
		return
	}
	mx, my := input.MousePosition()
	if !s.dragging {
		s.dragging = true
		s.dragFrom = golib.Vector2{X: mx, Y: my}
		return
	}
	// Shifting the anchor with the view keeps the zoom's glide from
	// fighting the drag when both move at once.
	dx := (mx - s.dragFrom.X) / s.zoom
	dy := (my - s.dragFrom.Y) / s.zoom
	s.camera.Target.X -= dx
	s.camera.Target.Y -= dy
	s.anchorWorld.X -= dx
	s.anchorWorld.Y -= dy
	s.dragFrom = golib.Vector2{X: mx, Y: my}
}

// updateInspection picks the tile under the pointer with the left button,
// cancels with a right click that never became a drag, and expands or folds
// a card when a click lands on its title. The camera has already moved, so
// the hover follows the view the frame it changes.
func (s *playScene) updateInspection(input *golib.Input) {
	mx, my := input.MousePosition()
	world := s.camera.ToWorld(mx, my)
	col, row, inside := tileAtWorld(world.X, world.Y)
	s.hovering = inside
	s.hoverCol, s.hoverRow = col, row

	if input.MousePressed(golib.MouseLeft) {
		if s.picked {
			panel := tooltipLayout(s.camera, s.pickedCol, s.pickedRow, s.expanded)
			if panel.contains(mx, my) {
				if thing, ok := panel.cardAt(mx, my); ok {
					s.expanded[thing.ID] = !s.expanded[thing.ID]
				}
				return
			}
		}
		s.picked = inside
		s.pickedCol, s.pickedRow = col, row
	}

	// A right click is a press and a release within a few pixels; anything
	// more was a drag, and drags don't deselect.
	down := input.MouseDown(golib.MouseRight)
	if down && !s.rightWasDown {
		s.rightFrom = golib.Vector2{X: mx, Y: my}
	}
	if s.rightWasDown && !down &&
		math.Abs(float64(mx-s.rightFrom.X)) < 4 && math.Abs(float64(my-s.rightFrom.Y)) < 4 {
		s.picked = false
	}
	s.rightWasDown = down
}

// regionOnScreen returns where the region's diamond lands on the screen, the
// rectangle the camera's view stays inside.
func regionOnScreen() golib.Rectangle {
	width := float32(regionCols+regionRows) * tileW / 2
	height := float32(regionCols+regionRows) * tileH / 2
	return golib.Rectangle{
		X:      regionOriginX - width/2,
		Y:      regionOriginY,
		Width:  width,
		Height: height,
	}
}

// Draw draws the region through the camera, and the text over it in screen
// pixels. It reads the state and never changes it.
func (s *playScene) Draw(screen *golib.Screen) {
	screen.SetCamera(s.camera)
	drawRegion(screen, s.zoom)
	if s.hovering && (!s.picked || s.hoverCol != s.pickedCol || s.hoverRow != s.pickedRow) {
		drawTileHighlight(screen, s.hoverCol, s.hoverRow, 1, hoveredTileColor)
	}
	if s.picked {
		drawTileHighlight(screen, s.pickedCol, s.pickedRow, 2, pickedTileColor)
	}
	screen.SetCamera(nil)
	screen.DrawText("niebla", 16, 16, 20, textColor)
	screen.DrawText(
		"wheel zooms, WASD or arrows or right-drag pans, left-click inspects a tile, Esc quits, F11 fullscreen, F2 filter",
		16, float32(screen.Height())-30, 10, textColor,
	)
	if s.picked {
		drawTooltip(screen, tooltipLayout(s.camera, s.pickedCol, s.pickedRow, s.expanded))
	}
}
