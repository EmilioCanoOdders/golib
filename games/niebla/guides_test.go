package main

import (
	"encoding/json"
	"os"
	"testing"

	"golib"
)

func TestEdgeGuidePointsTowardOffscreenTargets(t *testing.T) {
	width, height := float32(screenWidth), float32(screenHeight)
	center := golib.Vector2{X: width / 2, Y: height / 2}
	for _, target := range []golib.Vector2{
		{X: -200, Y: center.Y},
		{X: width + 200, Y: center.Y},
		{X: center.X, Y: -200},
		{X: center.X, Y: height + 200},
		{X: -200, Y: -200},
		{X: width + 200, Y: height + 200},
	} {
		position, direction, ok := edgeGuideAt(target, width, height)
		if !ok {
			t.Fatalf("no guide for offscreen target %v", target)
		}
		if position.X < guideEdgeInset ||
			position.X > width-guideEdgeInset ||
			position.Y < guideTopInset ||
			position.Y > height-guideBottomInset {
			t.Errorf("guide at %v is outside the safe screen area", position)
		}
		if direction.Dot(target.Sub(position)) <= 0 {
			t.Errorf("guide at %v points away from %v", position, target)
		}
	}
}

func TestEdgeGuideIsHiddenWhenTargetIsOnscreen(t *testing.T) {
	position, direction, ok := edgeGuideAt(
		golib.Vector2{X: 640, Y: 360}, screenWidth, screenHeight,
	)
	if ok || position != (golib.Vector2{}) || direction != (golib.Vector2{}) {
		t.Fatalf("onscreen target got guide %v pointing %v", position, direction)
	}
}

func TestSeparateEdgeGuidesAtScreenCorner(t *testing.T) {
	guides := []edgeGuide{
		{
			position:  golib.Vector2{X: guideEdgeInset, Y: guideTopInset},
			direction: golib.Vector2{X: -1, Y: -1}.Normalize(),
		},
		{
			position:  golib.Vector2{X: guideEdgeInset, Y: guideTopInset},
			direction: golib.Vector2{X: -1, Y: -1}.Normalize(),
		},
	}
	separateEdgeGuides(guides, screenWidth, screenHeight)
	if gap := guides[0].position.Distance(guides[1].position); gap <
		2*guideRadius+4 {
		t.Errorf("corner guides still overlap: %v and %v",
			guides[0].position, guides[1].position)
	}
}

func TestOffscreenGuidesFollowReportsAndPendingSchematics(t *testing.T) {
	s := newPlayScene(newGame())
	s.camera.Bounds = golib.Rectangle{}
	s.camera.Zoom = 4
	s.camera.Target = golib.Vector2{X: 640, Y: 372}
	s.camera.Snap()
	s.state.Ticks = 1
	s.state.Deliveries = 1
	s.state.Tech = map[string]bool{techInfraID: false}
	cx, cy := tileCenterUnits(coreCol, coreRow)
	s.state.Reports = []Report{{
		Tick: s.state.Ticks, Kind: ReportRazed, X: cx, Y: cy,
	}}
	s.camera.Target.X += 500
	s.camera.Snap()

	guides := edgeGuides(s, screenWidth, screenHeight)
	if len(guides) != 2 {
		t.Fatalf("got %d guides, want report and schematics", len(guides))
	}
	if guides[0].mark != "!" || guides[1].mark != "S" {
		t.Fatalf("guides are %q and %q, want report and schematics",
			guides[0].mark, guides[1].mark)
	}
	if guides[0].position.Distance(guides[1].position) < 2*guideRadius {
		t.Errorf("overlapping guides at %v and %v",
			guides[0].position, guides[1].position)
	}

	reportTick := s.state.Reports[0].Tick
	s.state.Ticks = reportTick + reportShowTicks - 1
	guides = edgeGuides(s, screenWidth, screenHeight)
	if len(guides) != 2 {
		t.Fatalf("ordinary report expired before 15 seconds: %+v", guides)
	}

	s.state.Ticks = reportTick + reportShowTicks
	guides = edgeGuides(s, screenWidth, screenHeight)
	if len(guides) != 1 || guides[0].mark != "S" {
		t.Errorf("expired report still has a guide: %+v", guides)
	}
}

func TestOffscreenCityGuideExpiresAfterOneMinute(t *testing.T) {
	s := newPlayScene(newGame())
	s.camera.Bounds = golib.Rectangle{}
	s.camera.Zoom = 4
	s.camera.Target = golib.Vector2{X: 640, Y: 372}
	s.camera.Snap()
	cx, cy := tileCenterUnits(coreCol, coreRow)
	cityID := s.state.foundCity(cx, cy, 0)
	city := s.state.Cities[cityID]
	s.camera.Target.X += 500
	s.camera.Snap()

	s.state.Ticks = city.AnnounceUntil - 1
	guides := edgeGuides(s, screenWidth, screenHeight)
	if len(guides) != 1 || guides[0].mark != "!" {
		t.Fatalf("the city guide disappeared before its minute: %+v", guides)
	}

	s.state.Ticks = city.AnnounceUntil
	guides = edgeGuides(s, screenWidth, screenHeight)
	if len(guides) != 0 {
		t.Errorf("the city guide remained after its minute: %+v", guides)
	}
}

func TestWriteGuideShotState(t *testing.T) {
	path := os.Getenv("NIEBLA_GUIDE_SHOT_STATE")
	if path == "" {
		t.Skip("set NIEBLA_GUIDE_SHOT_STATE to write a guide shot state")
	}
	s := newGame()
	s.Deliveries = 1
	s.Tech = map[string]bool{techInfraID: false}
	x, y := tileCenterUnits(coreCol, coreRow)
	s.Reports = []Report{{
		Tick: s.Ticks, Kind: ReportSettled,
		X: x + 1000, Y: y - 1000,
	}}
	data, err := json.MarshalIndent(map[string]any{"state": s}, "", "  ")
	if err != nil {
		t.Fatalf("the state doesn't marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
