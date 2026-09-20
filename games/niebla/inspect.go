package main

import (
	"fmt"

	"golib"
)

// Tooltip tuning, in screen pixels. The panel is one dark plate with a row
// per line: the tile's header, one card title per thing with its headline
// at the right, the expanded cards' details and, for deposits, a button
// that sends the card's robot to it, or recalls it.
const (
	tooltipWidth  = 260
	tooltipPad    = 12
	titleSize     = 12
	textSize      = 10
	titleRow      = 22 // line height of a card's title
	textRow       = 14 // line height of the header and of a detail line
	buttonRow     = 22 // line height of a button, with air around it
	buttonWidth   = 96
	barWidth      = 3  // the color bar on a card's left
	detailLabelW  = 68 // where a detail's value starts, past its label
	detailIndent  = 12 // how far details sit inside their card
	tooltipGap    = 14 // from the tile's corner to the panel
	tooltipMargin = 8  // kept between the panel and the screen's edges
)

// The labels a deposit card's button carries. Update reads them to know
// which action to apply, so treat them as identifiers, not prose.
const (
	buttonSend   = "send robot"
	buttonRecall = "recall robot"
)

// tooltipRow is one line the panel draws: the header, a card's title, an
// expanded card's detail, or a button.
type tooltipRow struct {
	thing   Thing  // the card the row belongs to; empty on the header
	detail  Detail // on a detail row
	header  bool
	title   bool
	open    bool    // the card is expanded
	summary string  // the title's headline, right-aligned
	button  string  // the label on a button row, "" otherwise
	bx, by  float32 // the button's rectangle on the screen
	bw, bh  float32 //
}

// tooltip is the picked tile's panel, laid out: where it stands on the
// screen and the rows it draws. Update hit-tests it and Draw paints it, so
// both see the same geometry.
type tooltip struct {
	col, row int
	x, y     float32
	w, h     float32
	rows     []tooltipRow
}

// tooltipLayout lays the picked tile's panel out. The camera decides where
// the tile's corner lands, so the panel follows the tile while the view
// moves. Deposit cards that still hold something get their robot button.
func tooltipLayout(
	s *State,
	camera *golib.Camera,
	col, row int,
	expanded map[string]bool,
) tooltip {
	t := tooltip{col: col, row: row, w: tooltipWidth}
	t.rows = append(t.rows, tooltipRow{header: true})
	for _, thing := range thingsAt(s, col, row) {
		info := catalogInfo(thing.Type)
		t.rows = append(t.rows, tooltipRow{
			thing:   thing,
			title:   true,
			open:    expanded[thing.ID],
			summary: info.summarize(thing),
		})
		if !expanded[thing.ID] {
			continue
		}
		for _, detail := range info.Details(s, thing) {
			t.rows = append(t.rows, tooltipRow{thing: thing, detail: detail})
		}
		if thing.Type != TypeOil && thing.Type != TypeLilac {
			continue
		}
		if thing.Amount <= 0 {
			continue // a dry deposit has nobody to send
		}
		label := buttonSend
		if _, owned := postOwner(s, col, row); owned {
			label = buttonRecall
		}
		t.rows = append(t.rows, tooltipRow{thing: thing, button: label})
	}
	// Lay the rows out inside the plate: heights first, then the buttons'
	// rectangles relative to it, then everything at its final place.
	y := float32(tooltipPad)
	for i := range t.rows {
		r := &t.rows[i]
		if r.button != "" {
			r.bx, r.by = tooltipPad+detailIndent, y+2
			r.bw, r.bh = buttonWidth, buttonRow-4
		}
		y += rowHeight(r)
	}
	t.h = y + tooltipPad
	sx, sy := projectTile(float32(col), float32(row))
	corner := camera.ToScreen(golib.Vector2{X: sx, Y: sy})
	t.x = corner.X + tileW/2 + tooltipGap
	if t.x+t.w > screenWidth-tooltipMargin {
		t.x = corner.X - tileW/2 - tooltipGap - t.w
	}
	t.x = clampf(t.x, tooltipMargin, screenWidth-tooltipMargin-t.w)
	t.y = clampf(corner.Y-titleRow/2, tooltipMargin, screenHeight-tooltipMargin-t.h)
	for i := range t.rows {
		t.rows[i].bx += t.x
		t.rows[i].by += t.y
	}
	return t
}

// rowHeight returns the line height of a panel row.
func rowHeight(r *tooltipRow) float32 {
	switch {
	case r.title:
		return titleRow
	case r.button != "":
		return buttonRow
	}
	return textRow
}

// contains reports whether the screen point is anywhere on the panel.
func (t tooltip) contains(x, y float32) bool {
	return x >= t.x && x <= t.x+t.w && y >= t.y && y <= t.y+t.h
}

// cardAt returns the thing whose card title holds the screen point, to
// expand or fold it.
func (t tooltip) cardAt(x, y float32) (Thing, bool) {
	if x < t.x || x > t.x+t.w {
		return Thing{}, false
	}
	lineY := t.y + tooltipPad
	for i := range t.rows {
		r := &t.rows[i]
		height := rowHeight(r)
		if r.title && y >= lineY && y < lineY+height {
			return r.thing, true
		}
		lineY += height
	}
	return Thing{}, false
}

// buttonAt returns the thing whose button holds the screen point, with
// the button's label.
func (t tooltip) buttonAt(x, y float32) (Thing, string, bool) {
	if x < t.x || x > t.x+t.w {
		return Thing{}, "", false
	}
	for i := range t.rows {
		r := &t.rows[i]
		if r.button == "" {
			continue
		}
		if x >= r.bx && x <= r.bx+r.bw && y >= r.by && y <= r.by+r.bh {
			return r.thing, r.button, true
		}
	}
	return Thing{}, "", false
}

// drawTooltip paints the panel: a dark plate, the tile's header, and one
// card per thing, named and colored by the catalog, with its details and
// button when expanded. The pointer's place lights the button it is over.
func drawTooltip(screen *golib.Screen, t tooltip, mx, my float32) {
	plate := golib.Rectangle{X: t.x, Y: t.y, Width: t.w, Height: t.h}
	screen.DrawRectangle(plate, panelColor)
	screen.DrawRectangleOutline(plate, 1, panelEdgeColor)
	x := t.x + tooltipPad
	y := t.y + tooltipPad
	for i := range t.rows {
		r := &t.rows[i]
		switch {
		case r.header:
			drawMarkup(screen, fmt.Sprintf("[dim]tile %d, %d[/]", t.col, t.row),
				x, y, textSize, panelTextColor)
		case r.title:
			info := catalogInfo(r.thing.Type)
			screen.DrawRectangle(
				golib.Rectangle{X: x, Y: y + 1, Width: barWidth, Height: titleRow - 6},
				info.Color)
			tx := x + barWidth + 6
			mark := "+ "
			if r.open {
				mark = "- "
			}
			tx = drawMarkup(screen, mark, tx, y, titleSize, panelDimColor)
			screen.DrawText(info.Name, tx, y, titleSize, info.Color)
			screen.DrawText(r.summary, t.x+t.w-tooltipPad, y, textSize, panelDimColor,
				golib.TextOptions{Align: golib.AlignRight})
		case r.button != "":
			fill := buttonColor
			if x, y := r.bx, r.by; mx >= x && mx <= x+r.bw && my >= y && my <= y+r.bh {
				fill = buttonHoverColor
			}
			rect := golib.Rectangle{X: r.bx, Y: r.by, Width: r.bw, Height: r.bh}
			screen.DrawRectangle(rect, fill)
			screen.DrawRectangleOutline(rect, 1, buttonEdgeColor)
			screen.DrawText(r.button, r.bx+8, r.by+4, textSize, panelTextColor)
		default:
			screen.DrawText(r.detail.Label, x+detailIndent, y, textSize, panelDimColor)
			drawMarkup(screen, r.detail.Value, x+detailIndent+detailLabelW, y,
				textSize, panelTextColor)
		}
		y += rowHeight(r)
	}
}

// findButton returns the row of the first button labeled so, or nil.
func (t tooltip) findButton(label string) *tooltipRow {
	for i := range t.rows {
		if t.rows[i].button == label {
			return &t.rows[i]
		}
	}
	return nil
}

// drawTileHighlight outlines a tile, in world space.
func drawTileHighlight(screen *golib.Screen, col, row int, thickness float32, color golib.Color) {
	x, y := projectTile(float32(col), float32(row))
	screen.DrawPolygonOutline(tileDiamond(x, y), thickness, color)
}

// clampf keeps v inside low and high.
func clampf(v, low, high float32) float32 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
