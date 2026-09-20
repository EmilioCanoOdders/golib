package main

import (
	"fmt"
	"math"

	"golib"
)

// Camera tuning, after R.U.S.E.: the wheel moves between whole stops and
// each stop doubles the zoom, so the range is huge — stop 0 shows the
// whole region with its deposits as icons, stop 5 shows about 80 m of
// ground. The zoom glides from stop to stop; only at rest is it a whole
// power of two, so shapes stay square. Panning keeps its speed on the
// screen, not in the world, so it glides the same at every zoom.
const (
	zoomOut = 0 // the stop that shows the whole region
	zoomIn  = 5 // the stop that shows the ground

	panSpeed = 480 // screen pixels per second

	zoomGlide = 0.1 // seconds for the zoom to close most of the way to its next stop
)

// zoomOfStop returns the zoom a stop stands for: two to the stop's power.
func zoomOfStop(stop float32) float32 {
	return float32(math.Exp2(float64(stop)))
}

// playScene shows the region through a camera. The camera and the
// selection are view, not state: they live here, outside the
// simulation, and never get serialized. The simulation's state does:
// Update turns input into actions and sends one Tick per update, and
// Draw renders the state and changes nothing.
type playScene struct {
	state *State

	glow, crt, soft *golib.Shader
	filterOn        bool
	camera          *golib.Camera
	zoom            float32       // the zoom on screen, gliding toward zoomOfStop(zoomStop)
	zoomStop        float32       // the whole stop the wheel last asked for
	anchorWorld     golib.Vector2 // while gliding, the point kept under the cursor
	anchorScreen    golib.Vector2
	dragging        bool          // the right button is down and moving the view
	dragFrom        golib.Vector2 // where the cursor stood at the last drag update
	mouse           golib.Vector2 // where the pointer stands, to light buttons

	picked       bool // a tile is selected and shows its panel
	pickedCol    int  // the selected tile
	pickedRow    int
	expanded     map[string]bool // which cards stand open, by thing ID
	radial       bool            // the build menu stands open on a cell
	radialCol    int             // the cell the menu opened on
	radialRow    int
	hovering     bool // the pointer is over the region
	hoverCol     int  // the tile under the pointer
	hoverRow     int
	hoverCellCol int  // the cell under the pointer, the cursor
	hoverCellRow int  //
	hoverCell    bool // the pointer is over a cell
	rightWasDown bool
	rightFrom    golib.Vector2 // where the right button went down
}

// newPlayScene turns the monitor filters on: the glow runs first, so the CRT
// scans the glowing picture, and the soft rounding goes last, so it rounds
// the whole result.
func newPlayScene() *playScene {
	s := &playScene{
		state:    newGame(),
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
	s.zoomStop, s.zoom = zoomOut, zoomOfStop(zoomOut)
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
	mx, my := input.MousePosition()
	s.mouse = golib.Vector2{X: mx, Y: my}
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
	// The loop is the clock: one tick of simulation per update.
	Apply(s.state, Tick{})
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

// zoomCamera glides the zoom to the stop the wheel asks for, keeping the
// point under the cursor under it while it moves. The in-between zooms
// only exist while gliding: at rest the zoom is a whole power of two.
func (s *playScene) zoomCamera(input *golib.Input, dt float32) {
	if notches := input.MouseWheel(); notches != 0 {
		stop := golib.Clamp(s.zoomStop+float32(int(notches)), zoomOut, zoomIn)
		if stop != s.zoomStop {
			s.zoomStop = stop
			mx, my := input.MousePosition()
			s.anchorWorld = s.camera.ToWorld(mx, my)
			s.anchorScreen = golib.Vector2{X: mx, Y: my}
		}
	}
	if target := zoomOfStop(s.zoomStop); s.zoom != target {
		keep := float32(math.Exp(float64(-dt / zoomGlide)))
		s.zoom += (target - s.zoom) * (1 - keep)
		if math.Abs(float64(target-s.zoom)) < 0.001 {
			s.zoom = target
		}
		s.camera.Zoom = s.zoom
		// Where the center must sit for the anchor to stay under the
		// cursor: the anchor minus the cursor's offset from the middle.
		s.camera.Target = golib.Vector2{
			X: s.anchorWorld.X - (s.anchorScreen.X-screenWidth/2)/s.zoom,
			Y: s.anchorWorld.Y - (s.anchorScreen.Y-screenHeight/2)/s.zoom,
		}
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
// cancels with a right click that never became a drag, expands or folds
// a card when a click lands on its title, and acts when a click lands on
// a card's button. A click on empty ground opens the build menu instead,
// a radial around the tile; picking one of its options marks that
// blueprint, and the click that confirms it lands on a cell. While a
// blueprint is marked, the panel and the menu stand down. The camera has
// already moved, so the hover follows the view the frame it changes.
func (s *playScene) updateInspection(input *golib.Input) {
	mx, my := input.MousePosition()
	world := s.camera.ToWorld(mx, my)
	col, row, inside := tileAtWorld(world.X, world.Y)
	s.hovering = inside
	s.hoverCol, s.hoverRow = col, row
	// The cursor is the cell, the grid's last subdivision, whatever the
	// scene is doing with it.
	if cc, cr, inCell := cellAtWorld(float64(world.X), float64(world.Y)); inCell {
		s.hoverCell, s.hoverCellCol, s.hoverCellRow = true, cc, cr
	} else {
		s.hoverCell = false
	}

	if input.MousePressed(golib.MouseLeft) {
		// The panel, while it stands, wins over whatever sits under it:
		// its buttons act even where it covers buildable ground.
		if s.picked {
			panel := tooltipLayout(s.state, s.camera, s.pickedCol, s.pickedRow, s.expanded)
			if panel.contains(mx, my) {
				if thing, label, ok := panel.buttonAt(mx, my); ok {
					s.pressButton(thing, label)
					return
				}
				if thing, ok := panel.cardAt(mx, my); ok {
					// A click folds or opens from where the card stands:
					// the default for its kind, or the last click's choice.
					open := cardOpen(catalogInfo(thing.Type), panel.lone, s.expanded, thing.ID)
					s.expanded[thing.ID] = !open
				}
				return
			}
		}
		if s.radial {
			// A pick raises the blueprint right on the menu's cell; a
			// click anywhere else puts the menu away.
			if item, hit := radialHover(radialLayout(s), mx, my); hit && item.ready {
				Apply(s.state, MarkBuilding{
					Kind: item.kind, Col: s.radialCol, Row: s.radialRow,
				})
				s.radial = false
			} else if !hit {
				s.radial = false
			}
		} else if s.buildableCell(s.hoverCellCol, s.hoverCellRow) {
			s.radial = true
			s.radialCol, s.radialRow = s.hoverCellCol, s.hoverCellRow
			s.picked = false
		} else {
			s.picked = inside
			s.pickedCol, s.pickedRow = col, row
		}
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
		s.radial = false
	}
	s.rightWasDown = down
}

// buildableCell reports whether a cell may ask for the build menu:
// buildable ground, nothing raised or rising there, no robot standing on
// it.
func (s *playScene) buildableCell(col, row int) bool {
	tcol, trow := cellTile(col, row)
	if tileAt(tcol, trow) != kindGround {
		return false
	}
	if _, occupied := buildingAt(s.state, col, row); occupied {
		return false
	}
	for _, job := range s.state.Jobs {
		if job.Col == col && job.Row == row {
			return false
		}
	}
	for _, id := range sortedRobotIDs(s.state) {
		r := s.state.Robots[id]
		if int(r.X/buildingCell) == col && int(r.Y/buildingCell) == row {
			return false
		}
	}
	return true
}

// pressButton applies the action a card's button asks for on the picked
// tile.
func (s *playScene) pressButton(thing Thing, label string) {
	switch label {
	case buttonSend:
		Apply(s.state, SendRobot{Col: s.pickedCol, Row: s.pickedRow})
	case buttonRecall:
		Apply(s.state, RecallRobot{Col: s.pickedCol, Row: s.pickedRow})
	case buttonBuildRobot:
		Apply(s.state, QueueRobot{Building: thing.Ref})
	}
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
	drawRegion(s.state, screen, s.zoom)
	// The cursor is the cell under the pointer, the grid's last
	// subdivision, about four robots across. Far out it lifts to a
	// readable size on the screen.
	if s.hoverCell {
		x, y := cellCenterUnits(s.hoverCellCol, s.hoverCellRow)
		gx, gy := project(float32(x), float32(y))
		scale := float32(buildingCell) / unitsPerTile
		if min := 12 / s.zoom / tileW; scale < min {
			scale = min
		}
		cursor := scaledDiamond(gx, gy, scale)
		screen.DrawPolygonOutline(cursor, 2/s.zoom, hoveredTileColor)
		screen.DrawCircle(gx, gy, 2.5/s.zoom, hoveredTileColor)
	}
	if s.picked {
		drawTileHighlight(screen, s.pickedCol, s.pickedRow, 2/s.zoom, pickedTileColor)
	}
	screen.SetCamera(nil)
	screen.DrawText("niebla", 16, 16, 20, textColor)
	drawMarkup(screen, s.hudLine(), 16, 44, 12, textColor)
	screen.DrawText(
		"click empty ground for the build menu, wheel zooms, WASD or arrows or right-drag pans, left-click inspects a tile, Esc quits, F11 fullscreen, F2 filter",
		16, float32(screen.Height())-30, 10, textColor,
	)
	if s.picked && !s.radial {
		panel := tooltipLayout(s.state, s.camera, s.pickedCol, s.pickedRow, s.expanded)
		drawTooltip(screen, panel, s.mouse.X, s.mouse.Y)
	}
	if s.radial {
		drawRadial(s, screen, s.mouse.X, s.mouse.Y)
	}
}

// hudLine is the strip of stores and hands under the game's name, each
// store against the room the colony has for it.
func (s *playScene) hudLine() string {
	return fmt.Sprintf("[oil]%s / %s[/]   [lilac]%s / %s[/]   [dim]%d robots[/]",
		si(s.state.Stock.Oil, "L"), si(oilCap(s.state), "L"),
		si(s.state.Stock.Lilac, "kg"), si(lilacCap(s.state), "kg"),
		len(s.state.Robots))
}
