package main

import (
	"math"

	"golib"
)

// The build menu: a radial of blueprints around the clicked cell. It is
// view, not state: the scene remembers the cell it opened on, and the
// options lay out around the cell's projected center every frame, so the
// menu follows the view while it pans or zooms. Picking an option raises
// that blueprint right on the cell - the click that opened the menu
// already chose where.

// The blueprints the menu offers, in the order they sit around it.
var blueprintOrder = []BuildingKind{
	BuildingFactory,
	BuildingCharger,
	BuildingSilo,
	BuildingWarehouse,
	BuildingProtector,
}

// Radial tuning, in screen pixels.
const (
	radialRadius = 62 // from the cell's center to an option's center
	radialItemR  = 15 // an option circle's radius
)

// radialItem is one option of the open menu, laid out on the screen.
type radialItem struct {
	kind        BuildingKind
	x, y        float32 // the circle's center
	ready       bool    // the colony may raise this kind on this cell
}

// radialLayout lays the open menu's options out around its cell's
// center, evenly spread, the first at the top.
func radialLayout(s *playScene) []radialItem {
	cx, cy := projectBuilding(Building{Col: s.radialCol, Row: s.radialRow})
	center := s.camera.ToScreen(golib.Vector2{X: cx, Y: cy})
	items := make([]radialItem, 0, len(blueprintOrder))
	for i, kind := range blueprintOrder {
		angle := -math.Pi/2 + float64(i)*2*math.Pi/float64(len(blueprintOrder))
		items = append(items, radialItem{
			kind: kind,
			x:    center.X + radialRadius*float32(math.Cos(angle)),
			y:    center.Y + radialRadius*float32(math.Sin(angle)),
			ready: canPlace(s.state, kind, s.radialCol, s.radialRow) &&
				canAfford(s.state, kind),
		})
	}
	return items
}

// radialHover returns the option under the pointer, if any.
func radialHover(items []radialItem, mx, my float32) (radialItem, bool) {
	for _, item := range items {
		if d := math.Hypot(float64(mx-item.x), float64(my-item.y)); d <= radialItemR+4 {
			return item, true
		}
	}
	return radialItem{}, false
}

// drawRadial paints the open menu: a marker on the cell's center and one
// circle per blueprint, ringed in its color and labeled under it, the
// option under the pointer lit and the ones the colony can't raise
// dimmed.
func drawRadial(s *playScene, screen *golib.Screen, mx, my float32) {
	cx, cy := projectBuilding(Building{Col: s.radialCol, Row: s.radialRow})
	center := s.camera.ToScreen(golib.Vector2{X: cx, Y: cy})
	screen.DrawCircle(center.X, center.Y, 3, pickedTileColor)
	items := radialLayout(s)
	hovered, _ := radialHover(items, mx, my)
	for _, item := range items {
		info := catalogInfo(buildingType(item.kind))
		lit := item.ready && item.kind == hovered.kind
		edge := info.Color
		if !item.ready {
			edge = golib.WithOpacity(info.Color, 80)
		}
		fill := panelColor
		if lit {
			fill, edge = info.Color, panelTextColor
		}
		screen.DrawCircle(item.x, item.y, radialItemR, fill)
		screen.DrawCircleOutline(item.x, item.y, radialItemR, 1.5, edge)
		color := edge
		if lit {
			color = panelTextColor
		}
		screen.DrawText(string(item.kind), item.x, item.y+radialItemR+5, 10, color,
			golib.TextOptions{Align: golib.AlignCenter})
	}
}
