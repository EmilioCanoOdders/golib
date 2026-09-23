package main

import (
	"math"
	"sort"

	"golib"
)

var (
	coreHull = []golib.Vector2{
		{X: 1.25}, {X: 0.5, Y: -0.38}, {X: -0.6, Y: -0.42},
		{X: -1.05, Y: -0.72}, {X: -0.85}, {X: -1.05, Y: 0.72},
		{X: -0.6, Y: 0.42}, {X: 0.5, Y: 0.38},
	}
	carrierHull = []golib.Vector2{
		{X: 1, Y: -0.5}, {X: 0.6, Y: -0.8},
		{X: -0.9, Y: -0.8}, {X: -1.1, Y: -0.48},
		{X: -1.1, Y: 0.48}, {X: -0.9, Y: 0.8},
		{X: 0.6, Y: 0.8}, {X: 1, Y: 0.5},
	}
	trooperHull = []golib.Vector2{
		{X: 0.95, Y: -0.55}, {X: 0.4, Y: -0.9},
		{X: -0.85, Y: -0.9}, {X: -1.1, Y: -0.6},
		{X: -1.1, Y: 0.6}, {X: -0.85, Y: 0.9},
		{X: 0.4, Y: 0.9}, {X: 0.95, Y: 0.55},
	}
	trooperTurret = []golib.Vector2{
		{X: 0.68}, {X: 0.25, Y: -0.38},
		{X: -0.48, Y: -0.38}, {X: -0.68},
		{X: -0.48, Y: 0.38}, {X: 0.25, Y: 0.38},
	}
)

// drawUnitModel draws one chassis at any size, facing and pixel scale.
// The world and screen-space unit icons use the same silhouettes.
func drawUnitModel(
	screen *golib.Screen, kind RobotKind, center golib.Vector2,
	radius float32, facing uint8, pixel float32, lampColor golib.Color,
) {
	point := func(forward, side float32) golib.Vector2 {
		return facingPoint(center, forward, side, radius, facing)
	}
	line := func(fromX, fromY, toX, toY, thick float32, color golib.Color) {
		from, to := point(fromX, fromY), point(toX, toY)
		screen.DrawLine(from.X, from.Y, to.X, to.Y, thick, color)
	}
	switch kind {
	case RobotBuilt:
		screen.DrawPolygon(facingPoints(center, radius, facing, carrierHull),
			robotDarkColor)
		screen.DrawPolygon(facingPoints(center, radius*0.83, facing, carrierHull),
			factoryColor)
		for _, side := range []float32{-0.7, 0.7} {
			line(-0.9, side, 0.85, side, radius*0.18, factoryDark)
		}
		pack := point(-0.33, 0)
		screen.DrawPolygon(facingPoints(pack, radius*0.4, facing,
			carrierHull), factoryDark)
		screen.DrawCircle(pack.X, pack.Y, radius*0.22, oilColor)
		lamp := point(0.86, 0)
		screen.DrawCircle(lamp.X, lamp.Y, radius*0.21, lampColor)
	case RobotCombat:
		drawTrooperModel(screen, center, radius, facing, pixel,
			guardColor, guardDark, lampColor)
	default:
		screen.DrawPolygon(facingPoints(center, radius, facing, coreHull),
			robotDarkColor)
		screen.DrawPolygon(facingPoints(center, radius*0.79, facing, coreHull),
			robotColor)
		for _, side := range []float32{-0.62, 0.62} {
			fin := point(-0.86, side)
			screen.DrawCircle(fin.X, fin.Y, radius*0.18, coreGlowColor)
		}
		lamp := point(0.92, 0)
		screen.DrawCircle(lamp.X, lamp.Y, radius*0.25, lampColor)
	}
}

func drawTrooperModel(
	screen *golib.Screen, center golib.Vector2,
	radius float32, facing uint8, pixel float32,
	body, dark, lampColor golib.Color,
) {
	screen.DrawPolygon(facingPoints(center, radius, facing, trooperHull), dark)
	screen.DrawPolygon(facingPoints(center, radius*0.82, facing, trooperHull),
		body)
	for _, side := range []float32{-0.74, 0.74} {
		from := facingPoint(center, -0.85, side, radius, facing)
		to := facingPoint(center, 0.48, side, radius, facing)
		screen.DrawLine(from.X, from.Y, to.X, to.Y, radius*0.26, dark)
	}
	from := facingPoint(center, 0.15, 0, radius, facing)
	to := facingPoint(center, 1.65, 0, radius, facing)
	barrelWidth := max(pixel*1.6, radius*0.22)
	screen.DrawLine(from.X, from.Y, to.X, to.Y, barrelWidth*1.6, dark)
	screen.DrawLine(from.X, from.Y, to.X, to.Y, barrelWidth*0.65,
		mid(body, coreGlowColor))
	screen.DrawPolygon(facingPoints(center, radius*0.7, facing,
		trooperTurret), dark)
	screen.DrawPolygon(facingPoints(center, radius*0.5, facing,
		trooperTurret), body)
	screen.DrawCircle(to.X, to.Y, radius*0.16, lampColor)
}

func drawRobots(s *State, screen *golib.Screen, zoom float32) {
	type spot struct {
		id int64
		p  golib.Vector2
	}
	spots := make([]spot, 0, len(s.Robots))
	for _, id := range sortedRobotIDs(s) {
		r := s.Robots[id]
		x, y := project(float32(r.X), float32(r.Y))
		spots = append(spots, spot{id, golib.Vector2{X: x, Y: y}})
	}
	sort.Slice(spots, func(i, j int) bool { return spots[i].p.Y < spots[j].p.Y })
	for _, sp := range spots {
		r := s.Robots[sp.id]
		p := sp.p
		radius := dotRadius(3, zoom, 3)
		screen.DrawCircle(p.X, p.Y+radius*0.5, radius*1.1,
			robotShadowColor)
		lamp := unitLamp(r.Kind)
		if r.tanked() && chargeStatus(s, r) == "refueling" &&
			s.Ticks/20%2 == 0 {
			lamp = robotDarkColor
		}
		drawUnitModel(screen, r.Kind, p, radius, r.Facing, 1/zoom, lamp)
		if r.Carry > 0 {
			cargo := lilacColor
			if r.Cargo == TypeOil {
				cargo = oilColor
			}
			pack := facingPoint(p, -0.78, 0.38, radius, r.Facing)
			screen.DrawCircle(pack.X, pack.Y, radius*0.32, cargo)
		}
		if r.Kind == RobotCombat && r.Health < trooperHealth {
			bar := golib.Rectangle{
				X: p.X - radius, Y: p.Y + radius*1.5,
				Width: 2 * radius, Height: 2 / zoom,
			}
			screen.DrawRectangle(bar, fillBarColor)
			bar.Width *= float32(math.Max(0, r.Health) / trooperHealth)
			screen.DrawRectangle(bar, guardColor)
		}
	}
}

func unitLamp(kind RobotKind) golib.Color {
	if kind == RobotBuilt {
		return chargerColor
	}
	return coreGlowColor
}
