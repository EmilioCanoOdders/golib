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

	techCalloutW           = 280.0 // the callout plate's width
	techCalloutPad         = 12.0  //
	techCalloutRow         = 17.0  // a wrapped line's height
	techCalloutHead        = 22.0  // the title's line height
	techCalloutSize        = 13.0  // the body's text size
	techCalloutMaxBodyRows = 4     // rows reserved in the dismissal hitbox

	// One square per thing a drop brings, drawn like the workers'
	// portraits a deposit's card holds: the icon of its construction
	// above, its name below.
	techSquareW       = 80.0 // three fill the plate's inside, gaps and all
	techSquareH       = 72.0
	techSquareGap     = 8.0
	techSquareIcon    = 40.0 // the box an icon is fitted into
	techSquareName    = 9.0  // the name's text size
	techSquareNameRow = 11.0 // and its lines' height
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

// techItem is one thing a drop brings: a blueprint's body and name, or
// the pipes, which build no building and carry a mark of their own.
type techItem struct {
	name  string
	kind  BuildingKind
	pipes bool
}

// techBrings lists what a drop brought in, one per square: the
// blueprints of its rung in the ladder and, with the frontier kit, the
// pipes whose LayPipe asks for that drop by name.
func techBrings(id string) []techItem {
	for i := range techLadder {
		d := &techLadder[i]
		if d.id != id {
			continue
		}
		items := make([]techItem, 0, len(d.kinds)+1)
		for _, kind := range d.kinds {
			items = append(items, techItem{
				name: catalogInfo(buildingType(kind)).Name,
				kind: kind,
			})
		}
		if id == techFrontierID {
			items = append(items, techItem{name: "Pipes", pipes: true})
		}
		return items
	}
	return nil
}

// techWords is what a drop's callout says: its title and one sentence of
// what it is for. The squares under them say what it brings.
func techWords(id string) (title, body string) {
	switch id {
	case techInfraID:
		return "infrastructure",
			"Schematics received. Click empty ground to raise a building."
	case techGuardID:
		return "guard post",
			"They are at the tanks: the mark they leave is their claim. A guard post would stop the next visit."
	case techFrontierID:
		return "frontier kit",
			"Grow the safe ground and draw oil without legs. Pipes come with them."
	case techIndustryID:
		return "robot factory",
			"More hands for the colony."
	case techMobileID:
		return "war factory",
			"Troopers are its squad. Keys 1-9 give the order."
	case techArtilleryID:
		return "artillery",
			"It shells what the colony sees. Every shell costs lilac and oil."
	}
	return "schematics", "The core received schematics."
}

// techWrap breaks a text into the lines that fit the width at its size.
func techWrap(screen *golib.Screen, text string, width, size float32) []string {
	lines, line := []string{}, ""
	for _, word := range strings.Fields(text) {
		try := word
		if line != "" {
			try = line + " " + word
		}
		if line != "" && screen.TextWidth(try, size, uiText) > width {
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

func techCalloutRect(s *playScene, height float32) golib.Rectangle {
	bx, by := techBadgeAt(s)
	x := bx + techBadgeR + 10
	if x+techCalloutW > screenWidth-8 {
		x = bx - techBadgeR - 10 - techCalloutW
	}
	x = clampf(x, 8, screenWidth-8-techCalloutW)
	y := clampf(by-height/2, 8, screenHeight-8-height)
	return golib.Rectangle{X: x, Y: y, Width: techCalloutW, Height: height}
}

// techCalloutBounds is the callout's hitbox: as tall as its tallest
// picture - four body rows and the squares' row - whatever the body
// wrapped into, so no click on the plate falls through to the region.
func techCalloutBounds(s *playScene) golib.Rectangle {
	height := techCalloutPad*2 + techCalloutHead +
		float32(techCalloutMaxBodyRows)*techCalloutRow +
		techSquareGap + techSquareH
	return techCalloutRect(s, height)
}

func (s *playScene) dismissTechCallout(mx, my float32) bool {
	s.techCallout = ""
	return techCalloutBounds(s).Contains(mx, my)
}

// drawTechCallout paints the open drop's teaching and the squares of
// what it brought, anchored to the badge and kept on the screen.
// Clicking it closes the callout; a click elsewhere closes it and acts
// on the region.
func drawTechCallout(s *playScene, screen *golib.Screen) {
	if s.techCallout == "" {
		return
	}
	title, body := techWords(s.techCallout)
	inner := float32(techCalloutW - 2*techCalloutPad)
	lines := techWrap(screen, body, inner, techCalloutSize)
	brings := techBrings(s.techCallout)
	h := techCalloutPad + techCalloutHead +
		float32(len(lines))*techCalloutRow + techCalloutPad
	if len(brings) > 0 {
		h += techSquareGap + techSquareH
	}
	plate := techCalloutRect(s, h)
	screen.DrawRectangle(plate, panelColor)
	screen.DrawRectangleOutline(plate, 1.5, techInk(s.techCallout))
	tx, ty := plate.X+techCalloutPad, plate.Y+techCalloutPad
	screen.DrawText(title, tx, ty, titleSize, panelTextColor, uiTextBold)
	ty += techCalloutHead
	for _, line := range lines {
		screen.DrawText(line, tx, ty, techCalloutSize, panelTextColor, uiText)
		ty += techCalloutRow
	}
	if len(brings) > 0 {
		drawTechSquares(screen, brings, tx, ty+techSquareGap)
	}
}

// drawTechSquares paints one square per thing a drop brought in, side
// by side from x, y: the plate of a deposit's worker portrait, the icon
// of its construction above and its name below.
func drawTechSquares(
	screen *golib.Screen,
	brings []techItem,
	x, y float32,
) {
	nameTop := y + 3 + techSquareIcon + 5
	nameBand := float32(2 * techSquareNameRow)
	for i, item := range brings {
		area := golib.Rectangle{
			X:      x + float32(i)*(techSquareW+techSquareGap),
			Y:      y,
			Width:  techSquareW,
			Height: techSquareH,
		}
		screen.DrawRectangle(area, buttonColor)
		screen.DrawRectangleOutline(area, 1, buttonEdgeColor)
		cx := area.X + techSquareW/2
		cy := area.Y + 3 + techSquareIcon/2
		if item.pipes {
			drawPipeIcon(screen, cx, cy, techSquareIcon)
		} else {
			drawBlueprintIcon(screen, item.kind, cx, cy, techSquareIcon)
		}
		names := techWrap(screen, item.name, techSquareW-12, techSquareName)
		ny := nameTop + (nameBand-float32(len(names))*techSquareNameRow)/2
		for _, line := range names {
			screen.DrawText(line, cx, ny, techSquareName, panelTextColor,
				golib.TextOptions{Font: uiFont, Align: golib.AlignCenter})
			ny += techSquareNameRow
		}
	}
}
