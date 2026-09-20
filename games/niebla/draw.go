package main

import (
	"math"
	"sort"

	"golib"
)

// propZoom is the zoom from which rocks and bushes are drawn: zoomed out
// they are a speck of noise, and their growing as you zoom in is the
// detail the zoom is for.
const propZoom = 4

// dotRadius returns the drawing radius, in projected pixels, of a world
// radius given in units, never smaller than minPx pixels on the screen:
// things show as points while the view is far out and grow to their true
// size as it closes in.
func dotRadius(units, zoom, minPx float32) float32 {
	r := units * unitW
	if min := minPx / zoom; r < min {
		r = min
	}
	return r
}

// drawRegion paints the still region and the robots on it: the ground
// inside the fog line, the core and its bubble, and the fog standing
// beyond the line. It reads the state and changes nothing.
func drawRegion(s *State, screen *golib.Screen, zoom float32) {
	screen.Clear(fogColor)
	drawGround(s, screen, zoom)
	drawDeposits(s, screen)
	drawRobots(s, screen, zoom)
	drawFogCover(screen)
	drawFogLine(screen, zoom)
	drawCoreAndBubble(screen, zoom)
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

// patchCorners returns the center and half sizes of a patch's union
// diamond, the shape its tiles make together on the screen.
func patchCorners(d Deposit) (cx, cy, halfW, halfH float32) {
	x, y := projectTile(float32(d.Col), float32(d.Row))
	cx = x + float32(d.Cols-d.Rows)*tileW/4
	cy = y + float32(d.Cols+d.Rows)*tileH/4
	halfW = float32(d.Cols+d.Rows) * tileW / 4
	halfH = float32(d.Cols+d.Rows) * tileH / 4
	return cx, cy, halfW, halfH
}

// patchDiamond returns a patch-shaped diamond around a center.
func patchDiamond(cx, cy, halfW, halfH float32) []golib.Vector2 {
	return []golib.Vector2{
		{X: cx, Y: cy - halfH},
		{X: cx + halfW, Y: cy},
		{X: cx, Y: cy + halfH},
		{X: cx - halfW, Y: cy},
	}
}

// drawDeposits paints each patch as one continuous body: a pool as one
// sheet of oil, a vein as one shelf of lilac rock with crystal clusters
// over its tiles. The body shrinks as the robots drain the patch, and a
// dry patch leaves one big scar.
func drawDeposits(s *State, screen *golib.Screen) {
	for _, d := range regionDeposits {
		cx, cy, halfW, halfH := patchCorners(d)
		fill := float32(s.Drain[depositKey(d)] / depositFull(d))
		if fill <= 0 {
			screen.DrawPolygon(patchDiamond(cx, cy, halfW*0.3, halfH*0.3), scarColor)
			continue
		}
		k := 0.45 + 0.55*fill
		if d.Kind == kindOil {
			screen.DrawPolygon(patchDiamond(cx, cy, halfW*0.82*k, halfH*0.82*k), oilDarkColor)
			screen.DrawPolygon(patchDiamond(cx, cy, halfW*0.6*k, halfH*0.6*k), oilColor)
			continue
		}
		screen.DrawPolygon(patchDiamond(cx, cy, halfW*0.78*k, halfH*0.78*k), lilacDarkColor)
		for row := d.Row; row < d.Row+d.Rows; row++ {
			for col := d.Col; col < d.Col+d.Cols; col++ {
				drawCrystals(screen, col, row, k)
			}
		}
	}
}

// drawCrystals draws a vein tile's crystal clusters, scaled by the
// patch's fill, off the tile's middle by a stable jitter so the clusters
// spread over the shelf instead of marching in step.
func drawCrystals(screen *golib.Screen, col, row int, k float32) {
	x, y := projectTile(float32(col), float32(row))
	cx := x + jitter(col, row, 5)*6
	cy := y + tileH/2 + jitter(col, row, 6)*4
	screen.DrawTriangle(cx-5*k, cy+3*k, cx-2*k, cy-11*k, cx+2*k, cy+3*k, lilacColor)
	screen.DrawTriangle(cx+2*k, cy+3*k, cx+5*k, cy-6*k, cx+8*k, cy+3*k, lilacLightColor)
}

// drawRock draws a rock 6 u (6 m) across, at the middle of its tile but
// for the jitter. Two facets, dark against the light; world-sized, so it
// grows from a pebble to a boulder as the view closes in.
func drawRock(screen *golib.Screen, x, y float32, col, row int) {
	cx := x + jitter(col, row, 1)*3*unitW
	cy := y + tileH/2 + jitter(col, row, 2)*2*unitH
	u := 3 * unitW // the rock's half width
	screen.DrawPolygon([]golib.Vector2{
		{X: cx - u, Y: cy + 0.6*u},
		{X: cx - 0.4*u, Y: cy - 1.2*u},
		{X: cx + 0.8*u, Y: cy - 0.6*u},
		{X: cx + u, Y: cy + 0.6*u},
	}, rockColor)
	screen.DrawPolygon([]golib.Vector2{
		{X: cx - 0.4*u, Y: cy - 1.2*u},
		{X: cx + 0.8*u, Y: cy - 0.6*u},
		{X: cx + 0.2*u, Y: cy + 0.2*u},
		{X: cx - 0.4*u, Y: cy - 0.4*u},
	}, rockLightColor)
}

// drawBush draws a bush of three circles 6 u across, at the middle of its
// tile but for the jitter. World-sized, like the rock.
func drawBush(screen *golib.Screen, x, y float32, col, row int) {
	cx := x + jitter(col, row, 3)*3*unitW
	cy := y + tileH/2 + jitter(col, row, 4)*2*unitH
	r := 1.5 * unitW
	screen.DrawCircle(cx-r, cy, r, bushColor)
	screen.DrawCircle(cx+r, cy, r, bushColor)
	screen.DrawCircle(cx, cy-r, r, bushLightColor)
}

// drawRobots paints the robots, back to front, each a dark body with a
// lit top and a glowing eye, its cargo as a colored pack on its back. A
// robot is 6 u across and the region is 5 km, so far out it is a point:
// dotRadius keeps it a few pixels wide until the view closes in.
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
		R := dotRadius(3, zoom, 3) // the body's radius, in projected pixels
		screen.DrawCircle(p.X, p.Y+R*0.5, R*1.1, robotShadowColor)
		screen.DrawCircle(p.X, p.Y, R, robotDarkColor)
		screen.DrawCircle(p.X, p.Y, R*0.68, robotColor)
		if r.Carry > 0 {
			cargo := lilacColor
			if r.Cargo == TypeOil {
				cargo = oilColor
			}
			screen.DrawCircle(p.X+R*0.84, p.Y+R*0.24, R*0.44, cargo)
		}
		screen.DrawCircle(p.X, p.Y-R*0.56, R*0.28, coreGlowColor)
	}
}

func drawCoreAndBubble(screen *golib.Screen, zoom float32) {
	cx, cy := projectTile(coreCol, coreRow)
	cy += tileH / 2
	// The pad is the core's whole tile; the pole's glow is world-sized,
	// a point of light until the view closes in. The outlines keep their
	// thickness on the screen, not in the world: inside the bubble they
	// would grow into roads.
	fillEllipse(screen, cx, cy, coreBubbleRadius, bubbleColor)
	ellipseOutline(screen, cx, cy, coreBubbleRadius, 3/zoom, bubbleEdgeColor)
	screen.DrawPolygon(scaledDiamond(cx, cy, 1), coreColor)
	glow := dotRadius(4, zoom, 3.5)
	screen.DrawCircle(cx, cy, glow, coreGlowColor)
	screen.DrawCircle(cx, cy, glow*0.5, coreColor)
}

func drawFogLine(screen *golib.Screen, zoom float32) {
	cx, cy := projectTile(coreCol, coreRow)
	cy += tileH / 2
	ellipseOutline(screen, cx, cy, fogLineRadius, 3/zoom,
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
