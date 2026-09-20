package main

import (
	"math"
	"sort"

	"golib"
)

// propZoom is the zoom from which rocks and bushes are drawn: at whole
// region view they are a speck of noise, and their appearing as you zoom in
// is the detail the zoom is for.
const propZoom = 2

// drawRegion paints the still region and the robots on it: the ground
// inside the fog line, the core and its bubble, and the fog standing
// beyond the line. It reads the state and changes nothing.
func drawRegion(s *State, screen *golib.Screen, zoom float32) {
	screen.Clear(fogColor)
	drawGround(s, screen, zoom)
	drawRobots(s, screen)
	drawFogCover(screen)
	drawFogLine(screen)
	drawCoreAndBubble(screen)
}

func drawGround(s *State, screen *golib.Screen, zoom float32) {
	for row := 0; row < regionRows; row++ {
		for col := 0; col < regionCols; col++ {
			distance := tileDistance(col, row)
			if distance > fogLineRadius-0.7+fogFadeTiles {
				continue
			}
			x, y := projectTile(float32(col), float32(row))
			color := groundColor
			if (col+row)%2 == 0 {
				color = groundShadeColor
			}
			screen.DrawPolygon(tileDiamond(x, y), color)
			switch tileAt(col, row) {
			case kindOil:
				if remainingAt(s, col, row) > 0 {
					drawOil(screen, x, y)
				} else {
					drawScar(screen, x, y)
				}
			case kindLilac:
				if remainingAt(s, col, row) > 0 {
					drawLilac(screen, x, y)
				} else {
					drawScar(screen, x, y)
				}
			case kindRock:
				if zoom >= propZoom {
					drawRock(screen, x, y, col, row)
				}
			case kindBush:
				if zoom >= propZoom {
					drawBush(screen, x, y, col, row)
				}
			}
		}
	}
}

// drawFogCover lays the fog over everything, tile by tile, fading in
// past the line. It runs after the robots, so whatever walks into the
// fog is swallowed by it.
func drawFogCover(screen *golib.Screen) {
	for row := 0; row < regionRows; row++ {
		for col := 0; col < regionCols; col++ {
			cover := fogCover(tileDistance(col, row))
			if cover <= 0 {
				continue
			}
			x, y := projectTile(float32(col), float32(row))
			screen.DrawPolygon(tileDiamond(x, y),
				golib.WithOpacity(fogColor, cover))
		}
	}
}

// fogCover returns how much fog sits on a tile, from 0 to 1: none inside the
// line, then fading in across fogFadeTiles until the tile is fog, one with
// the fog around the region. Fading per tile keeps the front hugging the
// tiles, with no gaps between the tiles and a smooth band.
func fogCover(distance float32) float32 {
	start := float32(fogLineRadius - 0.7)
	return golib.Clamp((distance-start)/fogFadeTiles, 0, 1)
}

// tileDiamond returns the four corners of a tile's top face, from its
// projected top corner.
func tileDiamond(x, y float32) []golib.Vector2 {
	return []golib.Vector2{
		{X: x, Y: y},
		{X: x + tileW/2, Y: y + tileH/2},
		{X: x, Y: y + tileH},
		{X: x - tileW/2, Y: y + tileH/2},
	}
}

// scaledDiamond returns a tile-shaped diamond, shrunk around a center.
func scaledDiamond(cx, cy, scale float32) []golib.Vector2 {
	w := tileW / 2 * scale
	h := tileH / 2 * scale
	return []golib.Vector2{
		{X: cx, Y: cy - h},
		{X: cx + w, Y: cy},
		{X: cx, Y: cy + h},
		{X: cx - w, Y: cy},
	}
}

func drawOil(screen *golib.Screen, x, y float32) {
	cx, cy := x, y+tileH/2
	screen.DrawPolygon(scaledDiamond(cx, cy, 0.64), oilDarkColor)
	screen.DrawPolygon(scaledDiamond(cx, cy, 0.46), oilColor)
}

func drawLilac(screen *golib.Screen, x, y float32) {
	cx, cy := x, y+tileH/2
	screen.DrawPolygon(scaledDiamond(cx, cy, 0.55), lilacDarkColor)
	screen.DrawTriangle(cx-5, cy+3, cx-2, cy-11, cx+2, cy+3, lilacColor)
	screen.DrawTriangle(cx+2, cy+3, cx+5, cy-6, cx+8, cy+3, lilacLightColor)
}

// drawRock draws a rock about one unit across, at the middle of its tile but
// for the jitter. Two facets, dark against the light.
func drawRock(screen *golib.Screen, x, y float32, col, row int) {
	cx := x + jitter(col, row, 1)*3*unitW
	cy := y + tileH/2 + jitter(col, row, 2)*2*unitH
	screen.DrawPolygon([]golib.Vector2{
		{X: cx - 5, Y: cy + 3},
		{X: cx - 2, Y: cy - 6},
		{X: cx + 4, Y: cy - 3},
		{X: cx + 5, Y: cy + 3},
	}, rockColor)
	screen.DrawPolygon([]golib.Vector2{
		{X: cx - 2, Y: cy - 6},
		{X: cx + 4, Y: cy - 3},
		{X: cx + 1, Y: cy + 1},
		{X: cx - 2, Y: cy - 2},
	}, rockLightColor)
}

// drawBush draws a bush of three circles about a unit and a half across, at
// the middle of its tile but for the jitter.
func drawBush(screen *golib.Screen, x, y float32, col, row int) {
	cx := x + jitter(col, row, 3)*3*unitW
	cy := y + tileH/2 + jitter(col, row, 4)*2*unitH
	screen.DrawCircle(cx-3, cy, 3, bushColor)
	screen.DrawCircle(cx+3, cy, 3, bushColor)
	screen.DrawCircle(cx, cy-3, 3, bushLightColor)
}

// drawScar marks a deposit tile the robots emptied: a shallow dark
// patch where the pool or the crystals used to be.
func drawScar(screen *golib.Screen, x, y float32) {
	cx, cy := x, y+tileH/2
	screen.DrawPolygon(scaledDiamond(cx, cy, 0.3), scarColor)
}

// drawRobots paints the robots, back to front, each a dark body with a
// lit top and a glowing eye, its cargo as a colored pack on its back.
func drawRobots(s *State, screen *golib.Screen) {
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
		screen.DrawCircle(p.X, p.Y+unitH*0.4, unitW*0.55, robotShadowColor)
		screen.DrawCircle(p.X, p.Y, unitW*0.5, robotDarkColor)
		screen.DrawCircle(p.X, p.Y, unitW*0.34, robotColor)
		if r.Carry > 0 {
			cargo := lilacColor
			if r.Cargo == TypeOil {
				cargo = oilColor
			}
			screen.DrawCircle(p.X+unitW*0.42, p.Y+unitH*0.12, unitW*0.22, cargo)
		}
		screen.DrawCircle(p.X, p.Y-unitW*0.28, unitW*0.14, coreGlowColor)
	}
}

func drawCoreAndBubble(screen *golib.Screen) {
	cx, cy := projectTile(coreCol, coreRow)
	cy += tileH / 2
	// The pole is 5 u across: one whole tile.
	fillEllipse(screen, cx, cy, coreBubbleRadius, bubbleColor)
	ellipseOutline(screen, cx, cy, coreBubbleRadius, 3, bubbleEdgeColor)
	screen.DrawPolygon(scaledDiamond(cx, cy, 1), coreColor)
	screen.DrawCircle(cx, cy, unitW, coreGlowColor)
	screen.DrawCircle(cx, cy, unitW/2, coreColor)
}

func drawFogLine(screen *golib.Screen) {
	cx, cy := projectTile(coreCol, coreRow)
	cy += tileH / 2
	ellipseOutline(screen, cx, cy, fogLineRadius, 3,
		golib.WithOpacity(fogBandColor, 0.55))
}

// ellipsePoints returns the corners of a world circle of a radius in tiles
// around a screen point, flattened by the projection.
func ellipsePoints(cx, cy, radius float32) []golib.Vector2 {
	const points = 192
	halfW, halfH := ellipseSemiAxes(radius)
	shape := make([]golib.Vector2, points)
	for i := range shape {
		angle := float64(i) / points * 2 * math.Pi
		shape[i] = golib.Vector2{
			X: cx + halfW*float32(math.Cos(angle)),
			Y: cy + halfH*float32(math.Sin(angle)),
		}
	}
	return shape
}

func fillEllipse(screen *golib.Screen, cx, cy, radius float32, color golib.Color) {
	screen.DrawPolygon(ellipsePoints(cx, cy, radius), color)
}

func ellipseOutline(
	screen *golib.Screen,
	cx, cy, radius, thickness float32,
	color golib.Color,
) {
	screen.DrawPolygonOutline(ellipsePoints(cx, cy, radius), thickness, color)
}
