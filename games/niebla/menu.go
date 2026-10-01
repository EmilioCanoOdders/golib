package main

import (
	"golib"
)

// The menu is the game's front door: it says who is playing — the
// machine's number — and Play carries the player to their base as they
// left it, or deals a new one. Leaving the play scene saves and lands
// here again.

// The menu's buttons, as indexes into menuScene.buttons.
const (
	playButton = iota
	quitButton
)

// The buttons' shape, on the screen's middle line.
const (
	menuButtonWidth  = 200
	menuButtonHeight = 38
	menuButtonStep   = 52
)

// menuScene is the title screen.
type menuScene struct {
	buttons  []menuButton
	selected int
}

// newMenuScene lays the menu out and puts the monitor filters on, the
// same ones the play scene runs under.
func newMenuScene() *menuScene {
	setFilters(true)
	s := &menuScene{selected: playButton}
	activeResize = s.resize
	s.resize(screenWidth, screenHeight)
	return s
}

func (s *menuScene) resize(width, height int) {
	x := (width - menuButtonWidth) / 2
	mid := float32(height) / 2
	buttonY := mid + 30
	if height < 360 {
		buttonY = mid + 12
	}
	s.buttons = []menuButton{
		playButton: {label: "Play", bounds: golib.Rectangle{
			X: float32(x), Y: buttonY, Width: menuButtonWidth, Height: menuButtonHeight,
		}},
		quitButton: {label: "Quit", bounds: golib.Rectangle{
			X: float32(x), Y: buttonY + menuButtonStep,
			Width: menuButtonWidth, Height: menuButtonHeight,
		}},
	}
}

// Update moves the selection with the up and down keys, the d-pad, the
// wheel or the pointer, and confirms it with Enter, Space, A, Start or a
// click. Esc quits, as the only screen where it does.
func (s *menuScene) Update(input *golib.Input, dt float32) {
	step := 0
	if input.KeyPressed(golib.KeyUp) || input.KeyPressed(golib.KeyW) ||
		input.GamepadPressed(0, golib.GamepadUp) || input.MouseWheel() > 0 {
		step--
	}
	if input.KeyPressed(golib.KeyDown) || input.KeyPressed(golib.KeyS) ||
		input.GamepadPressed(0, golib.GamepadDown) || input.MouseWheel() < 0 {
		step++
	}
	s.selected = moveSelection(s.selected, step, len(s.buttons))

	x, y := input.MousePosition()
	pointed := buttonAt(s.buttons, x, y)
	clicked := input.MousePressed(golib.MouseLeft) && pointed >= 0
	// The pointer picks the button under it only when it moves or clicks,
	// so a resting pointer doesn't fight the keys.
	if pointed >= 0 && (input.MouseMoved() || clicked) {
		s.selected = pointed
	}

	confirm := clicked || input.KeyPressed(golib.KeyEnter) || input.KeyPressed(golib.KeySpace) ||
		input.GamepadPressed(0, golib.GamepadA) || input.GamepadPressed(0, golib.GamepadStart)
	if input.KeyPressed(golib.KeyEscape) || input.GamepadPressed(0, golib.GamepadBack) ||
		(confirm && s.selected == quitButton) {
		golib.Quit()
	} else if confirm && s.selected == playButton {
		s.play()
	}
}

// play carries the player to their base: the one they left, or a new
// one. A save that can't be read starts from nothing rather than stand
// in the way; the database would have to be broken by hand for it.
func (s *menuScene) play() {
	state, found, err := resumeState()
	if err != nil || !found {
		state = nil
	}
	golib.SwitchScene(newPlayScene(state))
}

// Draw paints the fog, the game's name, the player's number and the
// menu. It reads and never changes anything.
func (s *menuScene) Draw(screen *golib.Screen) {
	screen.Clear(fogColor)
	drawCentered(screen, "niebla", float32(screenHeight)/2-118, 64, textColor)
	drawCentered(screen, "player "+playerDisplay(player), float32(screenHeight)/2-20, 16, textColor)
	if saveWarning != "" {
		drawCentered(screen, saveWarning, float32(screenHeight)/2+6, 11, groundColor)
	}
	for i, button := range s.buttons {
		button.draw(screen, i == s.selected)
	}
	help := "Enter or click to play, up, down or the wheel to choose, Esc quits"
	if screenWidth < 600 {
		help = "Enter / click: play   Esc: quit"
	}
	drawCentered(screen, help, float32(screenHeight)-18, 13, groundColor)
}

// drawCentered draws text centered on the screen's width.
func drawCentered(screen *golib.Screen, text string, y, size float32, color golib.Color) {
	x := (float32(screenWidth) - screen.TextWidth(text, size, uiText)) / 2
	screen.DrawText(text, x, y, size, color, uiText)
}

// menuButton is one on-screen button of the menu.
type menuButton struct {
	label  string
	bounds golib.Rectangle
}

// buttonAt returns the index of the button holding the point x, y, or -1.
func buttonAt(buttons []menuButton, x, y float32) int {
	for i, button := range buttons {
		if button.bounds.Contains(x, y) {
			return i
		}
	}
	return -1
}

// moveSelection moves the selected button index by step, staying within
// count buttons.
func moveSelection(selected, step, count int) int {
	return max(0, min(selected+step, count-1))
}

// draw paints the button, lit when it is the selected one.
func (b menuButton) draw(screen *golib.Screen, selected bool) {
	fill := buttonColor
	if selected {
		fill = buttonHoverColor
	}
	screen.DrawRectangle(b.bounds, fill)
	screen.DrawRectangleOutline(b.bounds, 2, buttonEdgeColor)
	const size = 16
	x := b.bounds.X + (b.bounds.Width-screen.TextWidth(b.label, size, uiText))/2
	y := b.bounds.Y + (b.bounds.Height-size)/2
	screen.DrawText(b.label, x, y, size, panelTextColor, uiText)
}
