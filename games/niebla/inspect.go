package main

import (
	"fmt"

	"golib"
)

// Tooltip tuning, in screen pixels. The panel is one dark plate with a row
// per line: the tile's header, one card title per thing with its headline
// at the right, and the expanded cards' details.
const (
	tooltipWidth  = 260
	tooltipPad    = 12
	titleSize     = 12
	textSize      = 10
	titleRow      = 22 // line height of a card's title
	textRow       = 14 // line height of the header and of a detail line
	barWidth      = 3  // the color bar on a card's left
	detailLabelW  = 68 // where a detail's value starts, past its label
	detailIndent  = 12 // how far details sit inside their card
	tooltipGap    = 14 // from the tile's corner to the panel
	tooltipMargin = 8  // kept between the panel and the screen's edges
)

// tooltipRow is one line the panel draws: the header, a card's title, or an
// expanded card's detail.
type tooltipRow struct {
	thing   Thing  // the card the row belongs to; empty on the header
	detail  Detail // on a detail row
	header  bool
	title   bool
	open    bool   // the card is expanded
	summary string // the title's headline, right-aligned
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
// moves.
func tooltipLayout(camera *golib.Camera, col, row int, expanded map[string]bool) tooltip {
	t := tooltip{col: col, row: row, w: tooltipWidth}
	t.rows = append(t.rows, tooltipRow{header: true})
	t.h = tooltipPad*2 + textRow
	for _, thing := range thingsAt(col, row) {
		info := catalogInfo(thing.Type)
		t.rows = append(t.rows, tooltipRow{
			thing:   thing,
			title:   true,
			open:    expanded[thing.ID],
			summary: info.summarize(thing.Amount),
		})
		t.h += titleRow
		if !expanded[thing.ID] {
			continue
		}
		for _, detail := range info.Details(thing) {
			t.rows = append(t.rows, tooltipRow{thing: thing, detail: detail})
			t.h += textRow
		}
	}
	sx, sy := projectTile(float32(col), float32(row))
	corner := camera.ToScreen(golib.Vector2{X: sx, Y: sy})
	t.x = corner.X + tileW/2 + tooltipGap
	if t.x+t.w > screenWidth-tooltipMargin {
		t.x = corner.X - tileW/2 - tooltipGap - t.w
	}
	t.x = clampf(t.x, tooltipMargin, screenWidth-tooltipMargin-t.w)
	t.y = clampf(corner.Y-titleRow/2, tooltipMargin, screenHeight-tooltipMargin-t.h)
	return t
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
	for _, r := range t.rows {
		height := float32(textRow)
		if r.title {
			height = titleRow
		}
		if r.title && y >= lineY && y < lineY+height {
			return r.thing, true
		}
		lineY += height
	}
	return Thing{}, false
}

// drawTooltip paints the panel: a dark plate, the tile's header, and one
// card per thing, named and colored by the catalog, with its details when
// expanded.
func drawTooltip(screen *golib.Screen, t tooltip) {
	plate := golib.Rectangle{X: t.x, Y: t.y, Width: t.w, Height: t.h}
	screen.DrawRectangle(plate, panelColor)
	screen.DrawRectangleOutline(plate, 1, panelEdgeColor)
	x := t.x + tooltipPad
	y := t.y + tooltipPad
	for _, r := range t.rows {
		switch {
		case r.header:
			drawMarkup(screen, fmt.Sprintf("[dim]tile %d, %d[/]", t.col, t.row),
				x, y, textSize, panelTextColor)
			y += textRow
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
			y += titleRow
		default:
			screen.DrawText(r.detail.Label, x+detailIndent, y, textSize, panelDimColor)
			drawMarkup(screen, r.detail.Value, x+detailIndent+detailLabelW, y,
				textSize, panelTextColor)
			y += textRow
		}
	}
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
