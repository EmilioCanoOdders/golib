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
	drawBuildings(s, screen, zoom)
	drawRobots(s, screen, zoom)
	drawFogCover(screen)
	drawFogLine(screen, zoom)
	drawJobs(s, screen, zoom)
	drawBubbles(s, screen, zoom)
}

// mid blends two colors halfway, for a box face between light and shade.
func mid(a, b golib.Color) golib.Color {
	mix := func(x, y uint8) uint8 { return uint8((int(x) + int(y)) / 2) }
	return golib.Color{
		R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: a.A,
	}
}

// drawBuildings paints the colony's structures, back to front, each an
// isometric body standing on its cell. Like the robots, they never
// shrink under an icon size on the screen: far out they are little
// marked blocks, and the view closes in on their true 40 u.
func drawBuildings(s *State, screen *golib.Screen, zoom float32) {
	type spot struct {
		id int64
		y  float32
	}
	spots := make([]spot, 0, len(s.Buildings))
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		_, y := projectBuilding(b)
		spots = append(spots, spot{id, y})
	}
	sort.Slice(spots, func(i, j int) bool { return spots[i].y < spots[j].y })
	for _, sp := range spots {
		b := s.Buildings[sp.id]
		gx, gy := projectBuilding(b)
		across, height := buildingSize(b.Kind)
		k := buildingIcon(across, height, zoom)
		drawBuilding(screen, b, gx, gy, across*k, height*k)
	}
}

// buildingIcon returns the factor that lifts a body's true size until it
// clears the icon minimum on the screen — the same law the robots obey —
// easing back to one as the view closes in.
func buildingIcon(across, height, zoom float32) float32 {
	const (
		iconW = 9.0 // the smallest a building reads on the screen
		iconH = 6.0 //
	)
	k := float32(1)
	if w := across * unitW; w < iconW/zoom {
		k = iconW / zoom / w
	}
	if h := height * unitH; h*k < iconH/zoom {
		k = iconH / zoom / h
	}
	return k
}

// projectBuilding returns where a building's ground center lands on the
// screen.
func projectBuilding(b Building) (gx, gy float32) {
	x, y := cellCenterUnits(b.Col, b.Row)
	return project(float32(x), float32(y))
}

// drawJobs paints the build sites: the part already built rises from the
// ground in solid colors, wrapped in the wireframe of the whole body, so
// the building grows bottom-up inside its scaffold. The work's progress
// bar sits under the cell. It runs after the fog, so a site in the mist
// stays visible — the colony's work shows through.
func drawJobs(s *State, screen *golib.Screen, zoom float32) {
	for _, job := range s.Jobs {
		gx, gy := projectBuilding(Building{Col: job.Col, Row: job.Row})
		info := catalogInfo(buildingType(job.Kind))
		// The cell's ground diamond, in the kind's color.
		screen.DrawPolygonOutline(scaledDiamond(gx, gy, buildingCell/unitsPerTile),
			1.5/zoom, info.Color)
		across, height := buildingSize(job.Kind)
		k := buildingIcon(across, height, zoom)
		across, height = across*k, height*k
		// The built part, from the ground up.
		done := golib.Clamp(1-float32(job.Left)/float32(buildingWorkTicks), 0, 1)
		if done > 0 {
			drawBuilding(screen, Building{Kind: job.Kind}, gx, gy,
				across, height*done)
		}
		// The scaffold: the whole body, in wireframe.
		hw := across * unitW / 2
		hh := across * unitH / 2
		hy := height * unitH
		wire := func(points []golib.Vector2) {
			screen.DrawPolygonOutline(points, 1.5/zoom, info.Color)
		}
		wire([]golib.Vector2{
			{X: gx, Y: gy - hy - hh}, {X: gx + hw, Y: gy - hy},
			{X: gx, Y: gy - hy + hh}, {X: gx - hw, Y: gy - hy},
		})
		wire([]golib.Vector2{
			{X: gx, Y: gy - hy + hh}, {X: gx + hw, Y: gy - hy},
			{X: gx + hw, Y: gy}, {X: gx, Y: gy + hh},
		})
		wire([]golib.Vector2{
			{X: gx - hw, Y: gy - hy}, {X: gx, Y: gy - hy + hh},
			{X: gx, Y: gy + hh}, {X: gx - hw, Y: gy},
		})
		// The progress bar: the site's width, filling with the work done.
		w := across * unitW
		y := gy + hh + 4/zoom
		screen.DrawRectangle(
			golib.Rectangle{X: gx - w/2, Y: y, Width: w, Height: 3 / zoom}, scarColor)
		if done > 0 {
			screen.DrawRectangle(
				golib.Rectangle{X: gx - w/2, Y: y, Width: w * done, Height: 3 / zoom},
				info.Color)
		}
	}
}

// buildingSize returns a kind's body: its footprint across and its
// height, both in world units. Every footprint fits the 25 u cell.
func buildingSize(kind BuildingKind) (across, height float32) {
	switch kind {
	case BuildingFactory:
		return 24, 14
	case BuildingCharger:
		return 16, 12
	case BuildingSilo:
		return 22, 22
	case BuildingWarehouse:
		return 25, 13
	case BuildingProtector:
		return 8, 24
	}
	return 20, 10
}

func drawBuilding(
	screen *golib.Screen,
	b Building,
	gx, gy, across, height float32,
) {
	switch b.Kind {
	case BuildingFactory:
		isoBox(screen, gx, gy, across, height,
			factoryColor, mid(factoryColor, factoryDark), factoryDark)
		// The chimney, a narrow stack at the works' corner.
		isoBox(screen, gx+across*unitW*0.22, gy-height*unitH*0.22,
			across*0.22, height*1.6, factoryColor, factoryDark, factoryDark)
	case BuildingCharger:
		isoBox(screen, gx, gy, across, height*0.6,
			chargerColor, mid(chargerColor, chargerDark), chargerDark)
		glow := across * unitW * 0.3
		screen.DrawCircle(gx, gy-height*unitH*0.8, glow, chargerColor)
		screen.DrawCircle(gx, gy-height*unitH*0.8, glow*0.5, oilColor)
	case BuildingSilo:
		// Three stacked tiers, each narrower: a silo.
		tier := across
		for i := 0; i < 3; i++ {
			h := height / 3
			isoBox(screen, gx, gy-float32(i)*h*unitH, tier, h,
				siloColor, mid(siloColor, siloDark), siloDark)
			tier *= 0.72
		}
	case BuildingWarehouse:
		isoBox(screen, gx, gy, across, height*0.75,
			warehouseColor, mid(warehouseColor, warehouseDark), warehouseDark)
		isoBox(screen, gx, gy-height*0.75*unitH, across*0.45, height*0.35,
			warehouseColor, warehouseDark, warehouseDark)
	case BuildingProtector:
		isoBox(screen, gx, gy, across, height,
			protectorColor, mid(protectorColor, protectorDark), protectorDark)
		glow := across * unitW * 0.7 // world-sized, like the core's
		screen.DrawCircle(gx, gy-(height+2)*unitH, glow, protectorColor)
		screen.DrawCircle(gx, gy-(height+2)*unitH, glow*0.5, bubbleEdgeColor)
	}
}

// isoBox draws an isometric box of a square footprint, across units on a
// side and height units tall, standing on the ground point gx, gy: a lit
// top, a side in shade and one halfway between.
func isoBox(
	screen *golib.Screen,
	gx, gy, across, height float32,
	top, right, left golib.Color,
) {
	hw := across * unitW / 2
	hh := across * unitH / 2
	hy := height * unitH
	screen.DrawPolygon([]golib.Vector2{
		{X: gx, Y: gy - hy - hh},
		{X: gx + hw, Y: gy - hy},
		{X: gx, Y: gy - hy + hh},
		{X: gx - hw, Y: gy - hy},
	}, top)
	screen.DrawPolygon([]golib.Vector2{
		{X: gx, Y: gy - hy + hh}, {X: gx + hw, Y: gy - hy},
		{X: gx + hw, Y: gy}, {X: gx, Y: gy + hh},
	}, right)
	screen.DrawPolygon([]golib.Vector2{
		{X: gx - hw, Y: gy - hy}, {X: gx, Y: gy - hy + hh},
		{X: gx, Y: gy + hh}, {X: gx - hw, Y: gy},
	}, left)
}

// drawBubbles paints the safe ground: the core's bubble last, so its
// glow sits over everything, and a dimmer pocket per shadow protector,
// drawn over the fog it holds back.
func drawBubbles(s *State, screen *golib.Screen, zoom float32) {
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		if b.Kind != BuildingProtector {
			continue
		}
		cx, cy := projectTile(float32(b.Col), float32(b.Row))
		cy += tileH / 2
		fillEllipse(screen, cx, cy, protectorBubbleTiles, protectorBubbleColor)
		ellipseOutline(screen, cx, cy, protectorBubbleTiles, 2/zoom, protectorEdgeColor)
	}
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
		// The eye says the model: the core's warm white for its own,
		// the charger's amber for a built one, blinking while it drinks.
		eye := coreGlowColor
		if r.Kind == RobotBuilt {
			eye = chargerColor
			if chargeStatus(s, r) == "refueling" && s.Ticks/20%2 == 0 {
				eye = robotDarkColor
			}
		}
		screen.DrawCircle(p.X, p.Y-R*0.56, R*0.28, eye)
	}
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
