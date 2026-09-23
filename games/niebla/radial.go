package main

import (
	"math"

	"golib"
)

// The build menu: a radial of blueprints around the clicked cell. It is
// view, not state: the scene remembers the cell it opened on, and the
// options lay out around the cell's projected center every frame, so the
// menu follows the view while it pans or zooms. Picking a blueprint
// raises it right on the cell - the click that opened the menu already
// chose where.
//
// The menu is two rings deep. The first offers the build groups, the
// second the blueprints of the group picked; a right click goes back a
// ring and closes the menu from the first.

// buildGroup names a family of blueprints the menu groups together.
type buildGroup string

const (
	groupIndustry  buildGroup = "industry"
	groupMilitary  buildGroup = "military"
	groupLogistics buildGroup = "logistics"
)

// buildGroups is the first ring, in the order it sits around the cell.
var buildGroups = []buildGroup{
	groupIndustry,
	groupMilitary,
	groupLogistics,
}

// groupMembers is the second ring of each group, in the order its
// blueprints sit around the cell. The pump is not here: it rises from an
// oil pool's card, never from the menu.
var groupMembers = map[buildGroup][]BuildingKind{
	groupIndustry: {BuildingFactory, BuildingWarFactory},
	groupMilitary: {BuildingGuard, BuildingArtillery},
	groupLogistics: {
		BuildingCharger, BuildingSilo, BuildingWarehouse, BuildingProtector,
	},
}

// groupColor is the ink a group's ring and glyph read in.
func groupColor(group buildGroup) golib.Color {
	switch group {
	case groupIndustry:
		return factoryColor
	case groupMilitary:
		return guardColor
	case groupLogistics:
		return warehouseColor
	}
	return panelTextColor
}

// Radial tuning, in screen pixels.
const (
	radialRadius = 62 // from the cell's center to an option's center
	radialItemR  = 15 // an option circle's radius
)

// radialCenter is where the menu's cell lands on the screen.
func radialCenter(s *playScene) golib.Vector2 {
	cx, cy := projectBuilding(Building{Col: s.radialCol, Row: s.radialRow})
	return s.camera.ToScreen(golib.Vector2{X: cx, Y: cy})
}

// radialSpot is where the i-th of n options sits on the ring around a
// center, the first at the top and the rest clockwise.
func radialSpot(n, i int, cx, cy float32) (x, y float32) {
	angle := -math.Pi/2 + float64(i)*2*math.Pi/float64(n)
	return cx + radialRadius*float32(math.Cos(angle)),
		cy + radialRadius*float32(math.Sin(angle))
}

// radialHit reports whether the pointer is on an option's circle.
func radialHit(x, y, mx, my float32) bool {
	d := math.Hypot(float64(mx-x), float64(my-y))
	return d <= radialItemR+4
}

// radialGroupItem is one group on the first ring.
type radialGroupItem struct {
	group buildGroup
	x, y  float32
}

// radialLeafItem is one blueprint on the second ring.
type radialLeafItem struct {
	kind BuildingKind
	x, y float32
}

// radialReady reports whether a blueprint may be raised on the menu's
// cell right now. The menu offers nothing else: what the schematics,
// the ground, the fog or the stores refuse is not on the rings at all,
// so every option a click could land on is one that would really rise.
func radialReady(s *playScene, kind BuildingKind) bool {
	return kindUnlocked(s.state, kind) &&
		canPlace(s.state, kind, s.radialCol, s.radialRow) &&
		canAfford(s.state, kind)
}

// radialGroupLayout lays the first ring out around the menu's cell:
// every group that holds a blueprint the colony could raise here.
func radialGroupLayout(s *playScene) []radialGroupItem {
	center := radialCenter(s)
	groups := make([]buildGroup, 0, len(buildGroups))
	for _, group := range buildGroups {
		for _, kind := range groupMembers[group] {
			if radialReady(s, kind) {
				groups = append(groups, group)
				break
			}
		}
	}
	items := make([]radialGroupItem, 0, len(groups))
	for i, group := range groups {
		x, y := radialSpot(len(groups), i, center.X, center.Y)
		items = append(items, radialGroupItem{group, x, y})
	}
	return items
}

// radialLeafLayout lays the picked group's raiseable blueprints out
// around the menu's cell.
func radialLeafLayout(s *playScene) []radialLeafItem {
	center := radialCenter(s)
	kinds := make([]BuildingKind, 0, len(groupMembers[s.radialGroup]))
	for _, kind := range groupMembers[s.radialGroup] {
		if radialReady(s, kind) {
			kinds = append(kinds, kind)
		}
	}
	items := make([]radialLeafItem, 0, len(kinds))
	for i, kind := range kinds {
		x, y := radialSpot(len(kinds), i, center.X, center.Y)
		items = append(items, radialLeafItem{kind, x, y})
	}
	return items
}

func radialGroupHover(
	items []radialGroupItem,
	mx, my float32,
) (radialGroupItem, bool) {
	for _, item := range items {
		if radialHit(item.x, item.y, mx, my) {
			return item, true
		}
	}
	return radialGroupItem{}, false
}

func radialLeafHover(
	items []radialLeafItem,
	mx, my float32,
) (radialLeafItem, bool) {
	for _, item := range items {
		if radialHit(item.x, item.y, mx, my) {
			return item, true
		}
	}
	return radialLeafItem{}, false
}

// pickRadial acts on a click while the menu stands open. A group opens
// its ring - the ring holds only what the colony could raise, so any
// group on it opens. A blueprint is marked on the cell, and a click
// anywhere else puts the menu away.
func (s *playScene) pickRadial(mx, my float32) {
	if s.radialLevel == 0 {
		item, hit := radialGroupHover(radialGroupLayout(s), mx, my)
		if !hit {
			s.closeRadial()
			return
		}
		s.radialGroup = item.group
		s.radialLevel = 1
		s.au.ui(1)
		return
	}
	item, hit := radialLeafHover(radialLeafLayout(s), mx, my)
	if !hit {
		s.closeRadial()
		return
	}
	Apply(s.state, MarkBuilding{
		Kind: item.kind, Col: s.radialCol, Row: s.radialRow,
	})
	s.au.ui(0.95)
	s.closeRadial()
}

// openRadial opens the build menu on a cell, on its first ring.
func (s *playScene) openRadial(col, row int) {
	s.radial = true
	s.radialLevel = 0
	s.radialCol, s.radialRow = col, row
	s.au.ui(1.05)
}

// closeRadial puts the build menu away.
func (s *playScene) closeRadial() {
	s.radial = false
	s.radialLevel = 0
}

// backRadial takes one ring back, and closes the menu from its first.
func (s *playScene) backRadial() {
	if s.radialLevel > 0 {
		s.radialLevel = 0
		return
	}
	s.closeRadial()
}

// drawRadial paints the open menu: a marker on the cell's center and one
// circle per option of the ring that stands open, ringed in its color
// and labeled under it, the option under the pointer lit. Every option
// on the rings is one the colony could raise; nothing else is offered.
// A blueprint's circle carries its body in miniature; a group's carries
// its mark, and the group picked stands on the cell so the player knows
// which ring they are in.
func drawRadial(s *playScene, screen *golib.Screen, mx, my float32) {
	center := radialCenter(s)
	if s.radialLevel == 0 {
		screen.DrawCircle(center.X, center.Y, 3, pickedTileColor)
		drawRadialGroups(s, screen, center, mx, my)
		return
	}
	drawGroupGlyph(screen, s.radialGroup, center.X, center.Y, 18,
		groupColor(s.radialGroup))
	drawRadialLeaves(s, screen, center, mx, my)
}

func drawRadialGroups(
	s *playScene,
	screen *golib.Screen,
	center golib.Vector2,
	mx, my float32,
) {
	items := radialGroupLayout(s)
	hovered, over := radialGroupHover(items, mx, my)
	for _, item := range items {
		ink := groupColor(item.group)
		lit := over && item.group == hovered.group
		fill, edge, label := panelColor, ink, ink
		if lit {
			fill = mid(panelColor, groupColor(item.group))
			edge, label = panelTextColor, panelTextColor
		}
		screen.DrawCircle(item.x, item.y, radialItemR, fill)
		screen.DrawCircleOutline(item.x, item.y, radialItemR, 1.5, edge)
		drawGroupGlyph(screen, item.group, item.x, item.y,
			radialItemR*1.6, label)
		screen.DrawText(string(item.group), item.x, item.y+radialItemR+5,
			13, label,
			golib.TextOptions{Font: uiFont, Align: golib.AlignCenter})
	}
}

func drawRadialLeaves(
	s *playScene,
	screen *golib.Screen,
	center golib.Vector2,
	mx, my float32,
) {
	items := radialLeafLayout(s)
	hovered, over := radialLeafHover(items, mx, my)
	for _, item := range items {
		info := catalogInfo(buildingType(item.kind))
		ink := info.Color
		lit := over && item.kind == hovered.kind
		fill, edge, label := panelColor, ink, ink
		if lit {
			fill = mid(panelColor, info.Color)
			edge, label = panelTextColor, panelTextColor
		}
		screen.DrawCircle(item.x, item.y, radialItemR, fill)
		screen.DrawCircleOutline(item.x, item.y, radialItemR, 1.5, edge)
		drawBlueprintIcon(screen, item.kind, item.x, item.y, radialItemR*1.5)
		screen.DrawText(string(item.kind), item.x, item.y+radialItemR+5,
			13, label,
			golib.TextOptions{Font: uiFont, Align: golib.AlignCenter})
	}
}
