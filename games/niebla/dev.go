package main

import "golib"

// The dev tools: a strip of buttons under the HUD for whoever works on
// the game. Holding Control and clicking the game's name twice opens and
// closes it. The strip is view; what its buttons do goes through actions
// (DevHoldSwell, DevSpawnRobot), like everything else that touches the
// state.

const (
	devDoubleClickTicks = 24 // updates between the two clicks, at the most

	devButtonWidth  = 120
	devButtonHeight = 18
	devButtonGap    = 8
	devStripX       = 16
	devStripY       = 68

	devSwellButton = 0
	devRobotButton = 1
)

// devOpen outlives the play scene, so a trip to the menu keeps the strip.
var devOpen bool

// devTitle is where the game's name stands on the play scene's HUD.
var devTitle = golib.Rectangle{X: 16, Y: 12, Width: 84, Height: 28}

// devTools is the strip's view state.
type devTools struct {
	updates   int64
	clickedAt int64 // the update of the last Control click on the name
	placing   bool  // the next click on the region puts a robot there
}

func devButtonBounds(index int) golib.Rectangle {
	return golib.Rectangle{
		X:      devStripX + float32(index)*(devButtonWidth+devButtonGap),
		Y:      devStripY,
		Width:  devButtonWidth,
		Height: devButtonHeight,
	}
}

// update handles the strip's clicks and reports whether it took this
// update's left click, so nothing under it acts on the same click.
func (d *devTools) update(s *playScene, input *golib.Input) bool {
	d.updates++
	if !devOpen {
		d.placing = false
	}
	if d.placing && input.MousePressed(golib.MouseRight) {
		d.placing = false
	}
	if !input.MousePressed(golib.MouseLeft) {
		return false
	}
	mx, my := input.MousePosition()
	control := input.KeyDown(golib.KeyLeftControl) ||
		input.KeyDown(golib.KeyRightControl)
	if control && devTitle.Contains(mx, my) {
		if d.clickedAt > 0 && d.updates-d.clickedAt <= devDoubleClickTicks {
			devOpen = !devOpen
			d.clickedAt = 0
		} else {
			d.clickedAt = d.updates
		}
		return true
	}
	if !devOpen {
		return false
	}
	if devButtonBounds(devSwellButton).Contains(mx, my) {
		Apply(s.state, DevHoldSwell{On: !s.state.Fog.Held})
		return true
	}
	if devButtonBounds(devRobotButton).Contains(mx, my) {
		d.placing = !d.placing
		return true
	}
	if d.placing {
		world := s.camera.ToWorld(mx, my)
		x, y := unitsAtWorld(float64(world.X), float64(world.Y))
		Apply(s.state, DevSpawnRobot{X: x, Y: y})
		return true
	}
	return false
}

// unitsAtWorld undoes project: the world units under a projected point.
func unitsAtWorld(px, py float64) (x, y float64) {
	a := (px - float64(regionOriginX)) / float64(unitW/2)
	b := (py - float64(regionOriginY)) / float64(unitH/2)
	return (a + b) / 2, (b - a) / 2
}

func (d *devTools) draw(s *playScene, screen *golib.Screen) {
	if !devOpen {
		return
	}
	swell := "hold a swell"
	if s.state.Fog.Held {
		swell = "let the swell go"
	}
	robot := "place robots"
	if d.placing {
		robot = "placing: click ground"
	}
	labels := []string{devSwellButton: swell, devRobotButton: robot}
	lit := []bool{devSwellButton: s.state.Fog.Held, devRobotButton: d.placing}
	for i, label := range labels {
		rect := devButtonBounds(i)
		fill := buttonColor
		if lit[i] || rect.Contains(s.mouse.X, s.mouse.Y) {
			fill = buttonHoverColor
		}
		screen.DrawRectangle(rect, fill)
		screen.DrawRectangleOutline(rect, 1, buttonEdgeColor)
		screen.DrawText(label, rect.X+8, rect.Y+3, 10, panelTextColor, uiText)
	}
	screen.DrawText("dev", devTitle.X+devTitle.Width, devTitle.Y+10, 12,
		panelDimColor, uiText)
}
