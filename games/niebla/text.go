package main

import "golib"

// The font every text in the game is drawn with, and the options that
// choose it.
var (
	uiFont = golib.NewFont("fonts/FiraSans-Regular.ttf")
	uiText = golib.TextOptions{Font: uiFont}
)
