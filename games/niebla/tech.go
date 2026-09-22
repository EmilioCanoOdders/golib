package main

import (
	"math"
	"strings"

	"golib"
)

// The schematics' view: the badge that glows over the core while an
// unopened drop waits, and the callout that opens on its click. All of
// it reads the state and draws — the badge breathes with the state's
// tick, so it freezes with the game — and nothing here is state; the
// click that opens a drop goes through AckTech like any action.

// Tuning, in screen pixels and ticks.
const (
	techBadgeR     = 22.0 // the badge's radius
	techBadgeGap   = 8.0  // air between the badge and the monolith's top
	techPulseTicks = 90   // one breath of the glow: 1.5 s

	techCalloutW    = 280.0 // the callout plate's width
	techCalloutPad  = 12.0  //
	techCalloutRow  = 17.0  // a wrapped line's height
	techCalloutHead = 22.0  // the title's line height
	techCalloutSize = 13.0  // the body's text size
)

// techBreath is the pulse the glow rides, 0 to 1, from the state's tick
// alone: no clock of its own, so a pause holds its breath.
func techBreath(s *State) float32 {
	a := float64(s.Ticks%techPulseTicks) / float64(techPulseTicks)
	return float32((math.Sin(a*2*math.Pi) + 1) / 2)
}

// techBadgeAt returns where the badge sits on the screen: over the top
// of the core's monolith, at a size that ignores the zoom.
func techBadgeAt(s *playScene) (x, y float32) {
	cx, cy := projectCore()
	k := buildingIcon((coreSlabWide+coreSlabDeep)/2, coreHeight, s.zoom)
	sum := (coreSlabDeep + coreSlabWide) / 2 * k
	at := s.camera.ToScreen(golib.Vector2{X: cx, Y: cy})
	top := at.Y - coreHeight*k*unitH - sum*unitH/2
	return at.X, top - techBadgeGap - techBadgeR
}

// techBadgeHolds reports whether the screen point is on the badge, with
// some slack around it.
func (s *playScene) techBadgeHolds(mx, my float32) bool {
	x, y := techBadgeAt(s)
	return math.Hypot(float64(mx-x), float64(my-y)) <= float64(techBadgeR)+6
}

// techInk is the color a drop is written in: its group's for a group's
// arrival, its blueprint's own for a single one.
func techInk(id string) golib.Color {
	switch id {
	case techInfraID:
		return groupColor(groupLogistics)
	case techGuardID:
		return groupColor(groupMilitary)
	case techIndustryID:
		return groupColor(groupIndustry)
	}
	for i := range techLadder {
		d := &techLadder[i]
		if d.id == id && len(d.kinds) > 0 {
			return catalogInfo(buildingType(d.kinds[0])).Color
		}
	}
	return panelTextColor
}

// drawTechMark paints the badge's middle: a group's glyph where a group
// arrived, the blueprint's own body where one building did.
func drawTechMark(screen *golib.Screen, id string, cx, cy, box float32) {
	switch id {
	case techInfraID:
		drawGroupGlyph(screen, groupLogistics, cx, cy, box, techInk(id))
	case techGuardID:
		drawGroupGlyph(screen, groupMilitary, cx, cy, box, techInk(id))
	case techIndustryID:
		drawGroupGlyph(screen, groupIndustry, cx, cy, box, techInk(id))
	case techFrontierID:
		drawBlueprintIcon(screen, BuildingProtector, cx, cy, box)
	case techMobileID:
		drawBlueprintIcon(screen, BuildingWarFactory, cx, cy, box)
	case techArtilleryID:
		drawBlueprintIcon(screen, BuildingArtillery, cx, cy, box)
	}
}

// drawTechBadge paints the badge over the core while the oldest
// unopened drop waits: two halos that breathe with the clock, the
// plate, its ring in the drop's ink and the mark inside.
func drawTechBadge(s *playScene, screen *golib.Screen) {
	id := techPending(s.state)
	if id == "" {
		return
	}
	x, y := techBadgeAt(s)
	breath := techBreath(s.state)
	ink := techInk(id)
	screen.DrawCircle(x, y, techBadgeR*(1.5+0.25*breath),
		golib.WithOpacity(ink, 0.10+0.10*breath))
	screen.DrawCircle(x, y, techBadgeR*(1.2+0.15*breath),
		golib.WithOpacity(coreGlowColor, 0.14+0.12*breath))
	screen.DrawCircle(x, y, techBadgeR, panelColor)
	screen.DrawCircleOutline(x, y, techBadgeR, 1.5, ink)
	drawTechMark(screen, id, x, y, techBadgeR*1.5)
}

// techWords is what a drop's callout says: its title, one sentence of
// body, and the dim list of what it brings.
func techWords(id string) (title, body, list string) {
	switch id {
	case techInfraID:
		return "infrastructure",
			"Schematics received. Click empty ground to raise a building.",
			"silo, warehouse, charger"
	case techGuardID:
		return "guard post",
			"They are at the tanks: the mark they leave is their claim. A guard post would stop the next visit.",
			"guard"
	case techFrontierID:
		return "frontier kit",
			"Grow the safe ground and draw oil without legs. Pipes come with them.",
			"protector, pump, pipes"
	case techIndustryID:
		return "robot factory",
			"More hands for the colony.",
			"factory"
	case techMobileID:
		return "war factory",
			"Troopers are its squad. Keys 1-9 give the order.",
			"war factory"
	case techArtilleryID:
		return "artillery",
			"It shells what the colony sees. Every shell costs lilac and oil.",
			"artillery"
	}
	return "schematics", "The core received schematics.", ""
}

// techWrap breaks a callout's body into lines that fit the plate.
func techWrap(screen *golib.Screen, text string, width float32) []string {
	lines, line := []string{}, ""
	for _, word := range strings.Fields(text) {
		try := word
		if line != "" {
			try = line + " " + word
		}
		if line != "" && screen.TextWidth(try, techCalloutSize, uiText) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line = try
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// drawTechCallout paints the open drop's teaching, anchored to the
// badge and kept on the screen. It stays until the next click, which
// updateTech swallows.
func drawTechCallout(s *playScene, screen *golib.Screen) {
	if s.techCallout == "" {
		return
	}
	title, body, list := techWords(s.techCallout)
	bx, by := techBadgeAt(s)
	inner := float32(techCalloutW - 2*techCalloutPad)
	lines := techWrap(screen, body, inner)
	h := techCalloutPad + techCalloutHead +
		float32(len(lines))*techCalloutRow + techCalloutPad
	if list != "" {
		h += techCalloutRow
	}
	x, y := bx+techBadgeR+10, by-h/2
	if x+techCalloutW > screenWidth-8 {
		x = bx - techBadgeR - 10 - techCalloutW
	}
	x = clampf(x, 8, screenWidth-8-techCalloutW)
	y = clampf(y, 8, screenHeight-8-h)
	plate := golib.Rectangle{X: x, Y: y, Width: techCalloutW, Height: h}
	screen.DrawRectangle(plate, panelColor)
	screen.DrawRectangleOutline(plate, 1.5, techInk(s.techCallout))
	tx, ty := x+techCalloutPad, y+techCalloutPad
	screen.DrawText(title, tx, ty, titleSize, panelTextColor, uiTextBold)
	ty += techCalloutHead
	for _, line := range lines {
		screen.DrawText(line, tx, ty, techCalloutSize, panelTextColor, uiText)
		ty += techCalloutRow
	}
	if list != "" {
		drawMarkup(screen, "[dim]"+list+"[/]", tx, ty, techCalloutSize,
			panelTextColor)
	}
}
