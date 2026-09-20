package main

import (
	"fmt"
	"testing"

	"golib"
)

func TestParseMarkupPlain(t *testing.T) {
	spans := parseMarkup("no colors here", panelTextColor)
	if len(spans) != 1 || spans[0].text != "no colors here" {
		t.Errorf("plain text parsed into %v, want one span", spans)
	}
	if spans[0].color != panelTextColor {
		t.Errorf("plain text took color %v, want the base one", spans[0].color)
	}
}

func TestParseMarkupColorsSpans(t *testing.T) {
	spans := parseMarkup("an [oil]oil pool[/] holds", panelTextColor)
	want := []markupSpan{
		{"an ", panelTextColor},
		{"oil pool", oilColor},
		{" holds", panelTextColor},
	}
	if len(spans) != len(want) {
		t.Fatalf("parsed %v, want %v", spans, want)
	}
	for i, w := range want {
		if spans[i] != w {
			t.Errorf("span %d is %v, want %v", i, spans[i], w)
		}
	}
}

func TestParseMarkupNests(t *testing.T) {
	spans := parseMarkup("[dim]a [oil]b[/] c[/]", panelTextColor)
	want := []markupSpan{
		{"a ", panelDimColor},
		{"b", oilColor},
		{" c", panelDimColor},
	}
	if len(spans) != len(want) {
		t.Fatalf("parsed %v, want %v", spans, want)
	}
	for i, w := range want {
		if spans[i] != w {
			t.Errorf("span %d is %v, want %v", i, spans[i], w)
		}
	}
}

func TestParseMarkupKeepsUnknownTags(t *testing.T) {
	spans := parseMarkup("[nope]x", panelTextColor)
	if len(spans) != 1 || spans[0].text != "[nope]x" {
		t.Errorf("an unknown tag parsed into %v, want it printed as it is", spans)
	}
}

func TestParseMarkupUnclosed(t *testing.T) {
	spans := parseMarkup("a [oil]b", panelTextColor)
	want := []markupSpan{
		{"a ", panelTextColor},
		{"b", oilColor},
	}
	if len(spans) != len(want) {
		t.Fatalf("parsed %v, want %v", spans, want)
	}
	for i, w := range want {
		if spans[i] != w {
			t.Errorf("span %d is %v, want %v", i, spans[i], w)
		}
	}
}

func TestTooltipLayoutRowsAndCards(t *testing.T) {
	s := newGame()
	camera := golib.NewCamera(screenWidth, screenHeight)
	panel := tooltipLayout(s, camera, coreCol, coreRow, map[string]bool{})
	titles := 0
	for _, r := range panel.rows {
		if r.title {
			titles++
		}
	}
	if titles != 1 {
		t.Fatalf("the core tile laid %d card titles, want 1", titles)
	}
	open := map[string]bool{"core@12,12": true}
	panel = tooltipLayout(s, camera, coreCol, coreRow, open)
	details := 0
	for _, r := range panel.rows {
		if r.title {
			if r.summary != "r = 800 m" {
				t.Errorf("the core card's headline is %q, want \"r = 800 m\"", r.summary)
			}
		} else if !r.header && r.button == "" {
			details++
		}
	}
	if details != 8 {
		t.Errorf("the expanded core card shows %d details, want 8", details)
	}
	if !panel.contains(panel.x+1, panel.y+1) || panel.contains(panel.x-1, panel.y) {
		t.Errorf("contains answers wrongly around the panel's edges")
	}
	thing, ok := panel.cardAt(panel.x+10, panel.y+tooltipPad+textRow+titleRow/2)
	if !ok || thing.Type != TypeCore {
		t.Errorf("cardAt the title found %v, %v, want the core thing", thing, ok)
	}
	if _, ok := panel.cardAt(panel.x+10, panel.y+tooltipPad+textRow/2); ok {
		t.Errorf("cardAt the header found a card, want none")
	}
}

func TestPrimaryCardsStartOpen(t *testing.T) {
	camera := golib.NewCamera(screenWidth, screenHeight)
	s := newGame()
	// The core's card starts open with no clicks: its details are there.
	panel := tooltipLayout(s, camera, coreCol, coreRow, map[string]bool{})
	details := 0
	for _, r := range panel.rows {
		if !r.header && !r.title && r.button == "" {
			details++
		}
	}
	if details != 8 {
		t.Errorf("the core's card starts with %d details shown, want 8", details)
	}

	// A deposit's card starts open, button included, with no clicks.
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	panel = tooltipLayout(s, camera, col, row, map[string]bool{})
	if panel.findButton(buttonSend) == nil {
		t.Error("the oil patch's card starts folded, want it open with its button")
	}

	// A robot alone on a tile starts open; the same robot beside a
	// deposit starts folded, for the deposit is the card that opens.
	r := s.Robots[1]
	col, row = robotTile(r)
	panel = tooltipLayout(s, camera, col, row, map[string]bool{})
	if panel.findButton(buttonSend) != nil {
		t.Fatal("the robot's tile has a deposit, want bare ground to test with")
	}
	if len(panel.rows) < 3 {
		t.Fatal("the lone robot's card starts folded, want it open")
	}
	oilCol, oilRow, _ := nearestTileOf(kindOil)
	cx, cy := tileCenterUnits(oilCol, oilRow)
	r.X, r.Y = cx, cy
	s.Robots[1] = r
	panel = tooltipLayout(s, camera, oilCol, oilRow, map[string]bool{})
	for _, thing := range panel.rows {
		if thing.title && thing.open && thing.thing.Type == TypeRobot {
			t.Error("the robot's card starts open beside a deposit, want it folded")
		}
	}
}

func TestTooltipOffersRobotButtons(t *testing.T) {
	camera := golib.NewCamera(screenWidth, screenHeight)
	s := newGame()
	col, row, ok := nearestTileOf(kindOil)
	if !ok {
		t.Fatal("the region has no oil to test with")
	}
	// The card belongs to the whole patch, so its ID carries the patch's
	// top corner tile, not the tile clicked.
	patch, _ := depositAt(col, row)
	open := map[string]bool{fmt.Sprintf("oil@%d,%d", patch.Col, patch.Row): true}
	panel := tooltipLayout(s, camera, col, row, open)
	if thing, label, ok := panel.buttonAt(panel.x+1, panel.y+1); ok {
		t.Errorf("the panel's corner hit %v's %q button, want nothing", thing, label)
	}
	button := panel.findButton(buttonSend)
	if button == nil {
		t.Fatal("the expanded oil card offers no send button")
	}
	mid := golib.Vector2{X: button.bx + button.bw/2, Y: button.by + button.bh/2}
	if _, label, ok := panel.buttonAt(mid.X, mid.Y); !ok || label != buttonSend {
		t.Errorf("buttonAt its own button gave %q, %v", label, ok)
	}
	// Sent for, the same card asks for the robot back.
	Apply(s, SendRobot{Col: col, Row: row})
	panel = tooltipLayout(s, camera, col, row, open)
	if panel.findButton(buttonSend) != nil {
		t.Error("an occupied post still offers to send a robot")
	}
	if panel.findButton(buttonRecall) == nil {
		t.Error("an occupied post offers no recall")
	}
	// A dry deposit has nobody to send.
	drained := newGame()
	drained.Drain[depositKey(patch)] = 0
	panel = tooltipLayout(drained, camera, col, row, open)
	if panel.findButton(buttonSend) != nil || panel.findButton(buttonRecall) != nil {
		t.Error("a dry deposit still offers a robot button")
	}
}
