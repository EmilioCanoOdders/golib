package main

import (
	"math"

	"golib"
)

const (
	facingRight uint8 = iota
	facingDownRight
	facingDown
	facingDownLeft
	facingLeft
	facingUpLeft
	facingUp
	facingUpRight
)

// facingFromMovement quantizes the flat isometric projection, not world axes.
func facingFromMovement(dx, dy float64) uint8 {
	screenX := (dx - dy) * float64(unitW) / 2
	screenY := (dx + dy) * float64(unitH) / 2
	sector := int(math.Round(math.Atan2(screenY, screenX) / (math.Pi / 4)))
	return uint8((sector + 8) % 8)
}

func facingAngle(facing uint8) float32 {
	return float32(facing%8) * 45
}

func facingPoint(
	center golib.Vector2,
	forward, side, size float32,
	facing uint8,
) golib.Vector2 {
	local := golib.Vector2{X: forward * size, Y: side * size}
	return center.Add(local.Rotate(facingAngle(facing)))
}

func facingPoints(
	center golib.Vector2,
	size float32,
	facing uint8,
	shape []golib.Vector2,
) []golib.Vector2 {
	points := make([]golib.Vector2, len(shape))
	angle := facingAngle(facing)
	for i, point := range shape {
		points[i] = center.Add(point.Scale(size).Rotate(angle))
	}
	return points
}
