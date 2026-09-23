package main

import (
	"math"

	"golib"
)

const (
	guideEdgeInset   = 28
	guideTopInset    = 78
	guideBottomInset = 52
	guideRadius      = 12
)

type edgeGuide struct {
	position  golib.Vector2
	direction golib.Vector2
	color     golib.Color
	mark      string
}

func edgeGuideAt(
	target golib.Vector2,
	width, height float32,
) (position, direction golib.Vector2, ok bool) {
	if target.X >= 0 && target.X <= width &&
		target.Y >= 0 && target.Y <= height {
		return golib.Vector2{}, golib.Vector2{}, false
	}

	center := golib.Vector2{X: width / 2, Y: height / 2}
	direction = target.Sub(center).Normalize()
	if direction.Length() == 0 {
		return golib.Vector2{}, golib.Vector2{}, false
	}

	limit := float32(math.Inf(1))
	if direction.X > 0 {
		limit = min(limit, (width-guideEdgeInset-center.X)/direction.X)
	}
	if direction.X < 0 {
		limit = min(limit, (guideEdgeInset-center.X)/direction.X)
	}
	if direction.Y > 0 {
		limit = min(limit,
			(height-guideBottomInset-center.Y)/direction.Y)
	}
	if direction.Y < 0 {
		limit = min(limit,
			(guideTopInset-center.Y)/direction.Y)
	}
	position = center.Add(direction.Scale(limit))
	return position, direction, true
}

func edgeGuides(s *playScene, width, height float32) []edgeGuide {
	guides := make([]edgeGuide, 0, 2)
	if report, ok := currentReport(s.state); ok {
		x, y := project(float32(report.X), float32(report.Y))
		at := s.camera.ToScreen(golib.Vector2{X: x, Y: y})
		if position, direction, visible := edgeGuideAt(at, width, height); visible {
			guides = append(guides, edgeGuide{
				position: position, direction: direction,
				color: dangerColor, mark: "!",
			})
		}
	}
	if id := techPending(s.state); id != "" {
		x, y := techBadgeAt(s)
		if position, direction, visible := edgeGuideAt(
			golib.Vector2{X: x, Y: y}, width, height,
		); visible {
			guides = append(guides, edgeGuide{
				position: position, direction: direction,
				color: techInk(id), mark: "S",
			})
		}
	}
	separateEdgeGuides(guides, width, height)
	return guides
}

func separateEdgeGuides(guides []edgeGuide, width, height float32) {
	for i := 1; i < len(guides); i++ {
		for j := 0; j < i; j++ {
			if guides[i].position.Distance(guides[j].position) >=
				2*guideRadius+4 {
				continue
			}
			tangent := guides[i].direction.Rotate(90)
			offset := float32(guideRadius + 3)
			guides[j].position = clampGuidePosition(
				guides[j].position.Sub(tangent.Scale(offset)),
				width, height,
			)
			guides[i].position = clampGuidePosition(
				guides[i].position.Add(tangent.Scale(offset)),
				width, height,
			)
			if guides[i].position.Distance(guides[j].position) <
				2*guideRadius+4 {
				extra := float32(guideRadius + 3)
				guides[j].position = clampGuidePosition(
					guides[j].position.Sub(tangent.Scale(extra)),
					width, height,
				)
				guides[i].position = clampGuidePosition(
					guides[i].position.Add(tangent.Scale(extra)),
					width, height,
				)
			}
		}
	}
}

func clampGuidePosition(
	position golib.Vector2,
	width, height float32,
) golib.Vector2 {
	position.X = golib.Clamp(position.X, guideEdgeInset, width-guideEdgeInset)
	position.Y = golib.Clamp(
		position.Y, guideTopInset, height-guideBottomInset,
	)
	return position
}

func drawEdgeGuides(s *playScene, screen *golib.Screen) {
	for _, guide := range edgeGuides(s, screen.Width(), screen.Height()) {
		drawEdgeGuide(screen, guide)
	}
}

func drawEdgeGuide(screen *golib.Screen, guide edgeGuide) {
	x, y := guide.position.X, guide.position.Y
	direction := guide.direction
	back := guide.position.Add(direction.Scale(8))
	side := direction.Rotate(90).Scale(6)
	tip := guide.position.Add(direction.Scale(18))
	screen.DrawTriangle(
		tip.X, tip.Y,
		back.X+side.X, back.Y+side.Y,
		back.X-side.X, back.Y-side.Y,
		guide.color,
	)
	screen.DrawCircle(x, y, guideRadius, panelColor)
	screen.DrawCircleOutline(x, y, guideRadius, 2, guide.color)
	screen.DrawText(guide.mark, x, y-7, 14, panelTextColor,
		golib.TextOptions{Align: golib.AlignCenter})
}
