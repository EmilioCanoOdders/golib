package main

import (
	"fmt"
	"math"
	"sort"

	"golib"
)

// The rivals' picture: the marks their scouts leave on the ground, their
// vehicles under their repulsors' pockets, the guard posts' shots, and
// the words that tell the player what they are up to. All of it reads
// the state (sim_enemies.go) and changes nothing.

const (
	reportShowTicks = 15 * 60 // ticks a report stays on the screen: 15 s

	markScale = 1.3 // the scout's doodle, about 40 u long
)

// drawMarks paints what the scouts left on the ground, in spray paint:
// the oldest drawing there is, flat on the ground, so everything walks
// over it.
func drawMarks(s *State, screen *golib.Screen, zoom float32) {
	thick := 2 * dotRadius(0.9, zoom, 0.75)
	for _, id := range sortedMarkIDs(s) {
		m := s.Marks[id]
		for _, stroke := range markStrokes() {
			for i := 1; i < len(stroke); i++ {
				ax, ay := project(
					float32(m.X+stroke[i-1].X*markScale),
					float32(m.Y+stroke[i-1].Y*markScale))
				bx, by := project(
					float32(m.X+stroke[i].X*markScale),
					float32(m.Y+stroke[i].Y*markScale))
				screen.DrawLine(ax, ay, bx, by, thick, markColor)
			}
		}
	}
}

// markStrokes returns the doodle's strokes, in units around its middle:
// two rounds and a long shape between them, closed by an arc.
func markStrokes() [][]PipePoint {
	arc := func(cx, cy, r, from, to float64) []PipePoint {
		const steps = 12
		points := make([]PipePoint, 0, steps+1)
		for i := 0; i <= steps; i++ {
			angle := from + (to-from)*float64(i)/steps
			points = append(points,
				PipePoint{cx + math.Cos(angle)*r, cy + math.Sin(angle)*r})
		}
		return points
	}
	shaft := []PipePoint{{-6, -4}}
	shaft = append(shaft, arc(16, 0, 4, -math.Pi/2, math.Pi/2)...)
	shaft = append(shaft, PipePoint{-6, 4})
	return [][]PipePoint{
		arc(-10, -5, 5, 0, 2*math.Pi),
		arc(-10, 5, 5, 0, 2*math.Pi),
		shaft,
	}
}

// vehicleIcon returns the factor that lifts a vehicle's true size until
// it is minPx across on the screen, the robots' law.
func vehicleIcon(across, zoom, minPx float32) float32 {
	if w := across * unitW; w < minPx/zoom {
		return minPx / zoom / w
	}
	return 1
}

// drawEnemies paints the rivals over the fog, so a party reads from far
// out as what it is: a pocket of clear air moving through the mist. The
// pockets go first, then the structures and vehicles back to front, then
// the squads' marks (squads.go).
func drawEnemies(s *State, screen *golib.Screen, zoom float32) {
	type spot struct {
		e    Enemy
		x, y float32
	}
	spots := make([]spot, 0, len(s.Enemies))
	for _, id := range sortedEnemyIDs(s) {
		e := s.Enemies[id]
		x, y := project(float32(e.X), float32(e.Y))
		spots = append(spots, spot{e, x, y})
		if reach := enemySpecOf(e.Kind).bubble; reach > 0 {
			tiles := float32(reach / unitsPerTile)
			ellipseOutline(screen, x, y, tiles, 2/zoom, enemyEdgeColor)
		}
	}
	sort.SliceStable(spots, func(i, j int) bool { return spots[i].y < spots[j].y })
	for _, sp := range spots {
		if sp.e.Kind == EnemyBase {
			drawCityBuilding(s, screen, sp.e, sp.x, sp.y, zoom)
			continue
		}
		if sp.e.City != 0 && sp.e.Party == 0 {
			drawCityBuilding(s, screen, sp.e, sp.x, sp.y, zoom)
			continue
		}
		drawVehicle(screen, sp.e, sp.x, sp.y, zoom)
	}
	drawCityConstruction(s, screen, zoom)
	drawSquadMarks(s, screen, zoom)
}

func drawCityConstruction(s *State, screen *golib.Screen, zoom float32) {
	for _, id := range sortedCityIDs(s) {
		city := s.Cities[id]
		if city.Stage >= len(cityBuildOrder) {
			continue
		}
		x, y := cityBuildingPosition(city, city.Stage)
		gx, gy := project(float32(x), float32(y))
		gray := golib.Color{R: 158, G: 169, B: 172, A: 255}
		w := dotRadius(11, zoom, 5)
		screen.DrawCircleOutline(gx, gy, w, 1/zoom, gray)
		bar := golib.Rectangle{
			X: gx - w, Y: gy + w + 2/zoom,
			Width: 2 * w, Height: 2 / zoom,
		}
		screen.DrawRectangle(bar, scarColor)
		progress := 1 - float32(city.Work)/cityBuildTicks
		bar.Width *= golib.Clamp(progress, 0, 1)
		screen.DrawRectangle(bar, gray)
	}
}

func drawCityBuilding(
	s *State,
	screen *golib.Screen,
	e Enemy,
	gx, gy, zoom float32,
) {
	gray := golib.Color{R: 119, G: 132, B: 137, A: 255}
	dark := golib.Color{R: 55, G: 65, B: 69, A: 255}
	light := golib.Color{R: 168, G: 178, B: 180, A: 255}
	k := buildingIcon(24, 18, zoom)
	switch e.Kind {
	case EnemyBase:
		isoSlab(screen, gx, gy, 30*k, 9*k, 15*k, gray, mid(gray, dark), dark)
		isoBox(screen, gx, gy-15*k*unitH, 6*k, 14*k, light, gray, dark)
	case EnemyCityCrawler:
		isoSlab(screen, gx, gy, 16*k, 9*k, 7*k, gray, mid(gray, dark), dark)
		isoBox(screen, gx, gy-7*k*unitH, 5*k, 13*k, light, gray, dark)
	case EnemyCityRepulsor:
		isoBox(screen, gx, gy, 10*k, 24*k, gray, mid(gray, dark), dark)
		screen.DrawCircle(gx, gy-26*k*unitH, 4*k*unitW, light)
	case EnemyCityOilworks:
		isoBox(screen, gx, gy, 20*k, 12*k, gray, mid(gray, dark), dark)
		isoBox(screen, gx, gy-12*k*unitH, 5*k, 16*k, light, gray, dark)
		for bead := 0; bead < 3; bead++ {
			phase := math.Mod(float64(s.Ticks%90)/90+float64(bead)/3, 1)
			fade := float32(math.Sin(math.Pi * phase))
			y := gy - (30+float32(phase)*18)*k*unitH
			color := golib.WithOpacity(oilColor, fade*0.34)
			screen.DrawCircle(gx+float32(bead-1)*4*k*unitW,
				y, 2*k*unitW, color)
		}
	case EnemyCityMine:
		isoBox(screen, gx, gy, 23*k, 10*k, gray, mid(gray, dark), dark)
		screen.DrawLine(gx-7*k*unitW, gy-8*k*unitH,
			gx+7*k*unitW, gy-8*k*unitH, 2/zoom, light)
		for crystal := float32(-1); crystal <= 1; crystal++ {
			x := gx + crystal*6*k*unitW
			screen.DrawLine(x, gy-11*k*unitH,
				x+2*k*unitW, gy-15*k*unitH, 2/zoom, lilacColor)
		}
	case EnemyCityFactory:
		isoBox(screen, gx, gy, 26*k, 14*k, gray, mid(gray, dark), dark)
		isoBox(screen, gx+5*k*unitW, gy-14*k*unitH,
			7*k, 18*k, gray, dark, dark)
	}
	drawHealthBar(screen, gx, gy, 24*k, zoom,
		e.Health, enemySpecOf(e.Kind).health, dangerColor)
}

// drawHealthBar paints what is left of something hurt under its foot;
// nothing for what is whole.
func drawHealthBar(
	screen *golib.Screen,
	gx, gy, across, zoom float32,
	health, full float64,
	color golib.Color,
) {
	if health >= full {
		return
	}
	w := across * unitW
	bar := golib.Rectangle{
		X: gx - w/2, Y: gy + across*unitH/2 + 2/zoom, Width: w, Height: 2 / zoom,
	}
	screen.DrawRectangle(bar, fillBarColor)
	bar.Width *= float32(math.Max(0, health) / full)
	screen.DrawRectangle(bar, color)
}

// drawVehicle paints one rival vehicle on its ground point: the crawler
// a long hull under its repulsor's lamp, the raider a tanker whose drum
// shows once there is oil in it, the scout a small hull with a lamp.
func drawVehicle(screen *golib.Screen, e Enemy, gx, gy, zoom float32) {
	var across float32
	switch e.Kind {
	case EnemyArtillery:
		k := vehicleIcon(16, zoom, 8)
		across = 16 * k
		isoSlab(screen, gx, gy, 16*k, 10*k, 6*k,
			enemyDark, mid(enemyDark, enemyColor), enemyColor)
		screen.DrawLine(gx, gy-7*k*unitH,
			gx+10*k*unitW, gy-10*k*unitH, 2/zoom, enemyLampColor)
	case EnemyCrawler:
		k := vehicleIcon(16, zoom, 9)
		across = 16 * k
		isoSlab(screen, gx, gy, 16*k, 9*k, 6*k,
			enemyColor, mid(enemyColor, enemyDark), enemyDark)
		screen.DrawCircle(gx, gy-8*k*unitH, 3*k*unitW, enemyLampColor)
	case EnemyScout:
		k := vehicleIcon(6, zoom, 5)
		across = 6 * k
		isoBox(screen, gx, gy, 6*k, 3.5*k,
			enemyColor, mid(enemyColor, enemyDark), enemyDark)
		screen.DrawCircle(gx, gy-5*k*unitH, 1.6*k*unitW, enemyLampColor)
	default:
		k := vehicleIcon(8, zoom, 5)
		across = 8 * k
		isoBox(screen, gx, gy, 8*k, 4*k,
			enemyColor, mid(enemyColor, enemyDark), enemyDark)
		if e.Oil > 0 {
			screen.DrawCircle(gx, gy-5*k*unitH, 2.4*k*unitW, oilColor)
		}
	}
	drawHealthBar(screen, gx, gy, across, zoom, e.Health,
		enemySpecOf(e.Kind).health, dangerColor)
}

// compassWord names the way from the core to a spot as the screen shows
// it: north is up.
func compassWord(x, y float64) string {
	cx, cy := tileCenterUnits(coreCol, coreRow)
	dx, dy := x-cx, y-cy
	angle := math.Atan2(-(dx+dy)/2, dx-dy)
	words := []string{
		"east", "north-east", "north", "north-west",
		"west", "south-west", "south", "south-east",
	}
	sector := int(math.Round(angle/(math.Pi/4))+8) % 8
	return words[sector]
}

// threatWords says what the rivals in the region are doing, for the
// HUD; "" while there are none.
func threatWords(s *State) string {
	for _, id := range sortedPartyIDs(s) {
		p := s.Parties[id]
		members := partyMembers(s, id)
		if len(members) == 0 {
			continue
		}
		lead := members[0]
		where := compassWord(lead.X, lead.Y)
		switch p.Stage {
		case StageApproach:
			return "something moves in the mist, " + where
		case StageCamp:
			left := (p.Wait + 59) / 60
			return fmt.Sprintf("raiders camped %s, moving in %d:%02d",
				where, left/60, left%60)
		case StageRaid:
			if lead.Kind == EnemyScout {
				return "an intruder, " + where
			}
			return "raid under way, " + where
		case StageSettled:
			continue
		default:
			return "rivals leaving, " + where
		}
	}
	for _, id := range sortedCityIDs(s) {
		city := s.Cities[id]
		where := compassWord(city.X, city.Y)
		if city.Stage < len(cityBuildOrder) {
			return fmt.Sprintf("rival city %s, building %s",
				where, cityBuildingName(cityBuildOrder[city.Stage]))
		}
		left := city.NextSortie - s.Ticks
		if left < 0 {
			left = 0
		}
		return fmt.Sprintf("rival city %s, next force in %d:%02d",
			where, left/3600, left/60%60)
	}
	return ""
}

// reportWords words a report for the player.
func reportWords(r Report) string {
	where := compassWord(r.X, r.Y)
	switch r.Kind {
	case ReportScout:
		return fmt.Sprintf("A scout siphoned [oil]%s[/] off your tanks and left its mark. "+
			"They know you are here. A guard post would stop the next visit.",
			si(math.Round(r.Oil), "L"))
	case ReportCamp:
		return fmt.Sprintf("[danger]Raiders have camped to the %s.[/] "+
			"They are getting ready: so should you.", where)
	case ReportRaid:
		return "[danger]The raiders are moving in[/], for the nearest tank with oil in it."
	case ReportLeft:
		if r.Oil < 1 {
			return "The rivals left empty-handed."
		}
		return fmt.Sprintf("The raiders got away with [oil]%s[/].",
			si(math.Round(r.Oil), "L"))
	case ReportDestroyed:
		return "The rivals are gone to the last vehicle. What they carried lies where they fell."
	case ReportSettled:
		return fmt.Sprintf("[danger]A rival city is establishing to the %s.[/] "+
			"Its extractors and war factory will make forces here.", where)
	case ReportGun:
		return fmt.Sprintf("A rival building to the %s is complete.", where)
	case ReportBaseDown:
		return fmt.Sprintf("The rival city to the %s has fallen.", where)
	case ReportSortie:
		return fmt.Sprintf("[danger]A rival battalion is assembling to the %s.[/]",
			where)
	case ReportCityIncoming:
		return fmt.Sprintf("[danger]A crawler approaches from the %s.[/] "+
			"It is looking for a place to establish a city.", where)
	case ReportCityBuilding:
		index := int(r.Stage)
		if index >= 0 && index < len(cityBuildOrder) {
			return fmt.Sprintf("The rival city to the %s completed its %s.",
				where, cityBuildingName(cityBuildOrder[index]))
		}
	case ReportRazed:
		return fmt.Sprintf("[danger]A shell brought a building down, %s.[/] Half of it lies there as a pile.", where)
	}
	return ""
}

// drawReport shows the latest report for a while, on a plate under the
// HUD, in the middle of the screen.
func drawReport(s *State, screen *golib.Screen) {
	if len(s.Reports) == 0 {
		return
	}
	r := s.Reports[len(s.Reports)-1]
	if s.Ticks-r.Tick > reportShowTicks {
		return
	}
	words := reportWords(r)
	width := float32(0)
	for _, span := range parseMarkup(words, panelTextColor) {
		width += screen.TextWidth(span.text, 13, uiText)
	}
	x := (float32(screen.Width()) - width) / 2
	const y = 120 // under the dev tools' second row
	plate := golib.Rectangle{
		X: x - 10, Y: y - 6, Width: width + 20, Height: textRow + 12,
	}
	screen.DrawRectangle(plate, panelColor)
	screen.DrawRectangleOutline(plate, 1, dangerColor)
	drawMarkup(screen, words, x, y, 13, panelTextColor)
}
