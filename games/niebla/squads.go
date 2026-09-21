package main

import (
	"fmt"
	"math"

	"golib"
)

// The squads on the screen: the pointer's mode that gives a squad its
// order, the mark of what each squad is at, and the shots that fly
// between troopers and rivals. The order itself goes through OrderSquad.

const orderPickPx = 16 // screen pixels around a vehicle that still pick it

// enemyUnder returns the rival vehicle under the pointer, the nearest on
// the screen within orderPickPx.
func (s *playScene) enemyUnder(mx, my float32) (Enemy, bool) {
	var best Enemy
	found, bestGap := false, float64(orderPickPx)
	for _, id := range sortedEnemyIDs(s.state) {
		e := s.state.Enemies[id]
		gx, gy := project(float32(e.X), float32(e.Y))
		at := s.camera.ToScreen(golib.Vector2{X: gx, Y: gy})
		if gap := math.Hypot(float64(at.X-mx), float64(at.Y-my)); gap <= bestGap {
			best, found, bestGap = e, true, gap
		}
	}
	return best, found
}

// updateOrdering is the pointer while it gives a squad its order: a left
// click on a rival vehicle sends the squad after its party, that vehicle
// first; a left click anywhere else on the region posts the squad there;
// a right click puts the pointer away. One click, one order.
func (s *playScene) updateOrdering(input *golib.Input, clickTaken, rightClick bool) {
	if _, stands := s.state.Buildings[s.ordering]; !stands || rightClick {
		s.ordering = 0
		return
	}
	if !input.MousePressed(golib.MouseLeft) || clickTaken {
		return
	}
	mx, my := input.MousePosition()
	order := OrderSquad{Squad: s.ordering}
	if e, ok := s.enemyUnder(mx, my); ok {
		order.Enemy = e.ID
	} else if !s.hoverCell {
		return
	}
	world := s.camera.ToWorld(mx, my)
	order.X, order.Y = unitsAtWorld(float64(world.X), float64(world.Y))
	Apply(s.state, order)
	s.ordering = 0
}

// drawOrderingLabel writes by the pointer what a click would order.
func (s *playScene) drawOrderingLabel(screen *golib.Screen) {
	words := "guard here"
	if e, ok := s.enemyUnder(s.mouse.X, s.mouse.Y); ok {
		words = "attack, " + string(e.Kind) + " first"
	}
	width := screen.TextWidth(words, textSize, uiText)
	x, y := s.mouse.X+16, s.mouse.Y-textRow/2
	screen.DrawRectangle(golib.Rectangle{
		X: x - 6, Y: y - 4, Width: width + 12, Height: textRow + 6,
	}, panelColor)
	screen.DrawText(words, x, y, textSize, guardColor, uiText)
}

// drawSquadMarks paints what each squad is at, over the fog: a small
// pennant on the spot it guards, or a ring around the vehicle it is to
// shoot first.
func drawSquadMarks(s *State, screen *golib.Screen, zoom float32) {
	for _, id := range sortedBuildingIDs(s) {
		if s.Buildings[id].Kind != BuildingWarFactory {
			continue
		}
		if len(squadMembers(s, id)) == 0 {
			continue
		}
		sq := squadOf(s, id)
		if e, ok := s.Enemies[sq.Focus]; ok && sq.Order == OrderAttack {
			gx, gy := project(float32(e.X), float32(e.Y))
			r := dotRadius(14, zoom, 9)
			screen.DrawCircleOutline(gx, gy-2*unitH, r, 1.5/zoom, guardColor)
			continue
		}
		gx, gy := project(float32(sq.X), float32(sq.Y))
		pole := dotRadius(14, zoom, 10)
		screen.DrawLine(gx, gy, gx, gy-pole, 1.5/zoom, guardColor)
		screen.DrawTriangle(gx, gy-pole, gx, gy-pole*0.55,
			gx+pole*0.6, gy-pole*0.78, guardColor)
	}
}

// squadWords says a squad in a few words: how many it is and what it is
// at.
func squadWords(s *State, home int64) string {
	count := len(squadMembers(s, home))
	if count == 0 {
		return "no troopers yet"
	}
	sq := squadOf(s, home)
	doing := "guarding"
	if sq.Order == OrderAttack {
		doing = "attacking"
		if e, ok := s.Enemies[sq.Focus]; ok {
			doing = fmt.Sprintf("attacking, %s first", e.Kind)
		}
	}
	return fmt.Sprintf("squad of %d, %s", count, doing)
}
