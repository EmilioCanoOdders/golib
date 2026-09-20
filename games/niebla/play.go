package main

import "golib"

// playScene shows the region. The slice is still: nothing moves yet, so
// Update only answers the keys that leave, resize or filter the game.
type playScene struct {
	glow, crt, soft *golib.Shader
	filterOn        bool
}

// newPlayScene turns the monitor filters on: the glow runs first, so the CRT
// scans the glowing picture, and the soft rounding goes last, so it rounds
// the whole result.
func newPlayScene() *playScene {
	s := &playScene{
		glow: golib.NewShader(glowSource),
		crt:  golib.NewShader(crtSource),
		soft: golib.NewShader(softSource),
	}
	s.glow.SetUniform("strength", glowStrength)
	s.crt.SetUniform("curvature", crtCurvature)
	s.soft.SetUniform("amount", 0.35)
	s.setFilters(true)
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
}

// Draw draws the region. It reads the state and never changes it.
func (s *playScene) Draw(screen *golib.Screen) {
	drawRegion(screen)
	screen.DrawText("niebla", 16, 16, 20, textColor)
	screen.DrawText(
		"slice 1: the region. Esc quits, F11 fullscreen, F2 filter",
		16, float32(screen.Height())-30, 10, textColor,
	)
}
