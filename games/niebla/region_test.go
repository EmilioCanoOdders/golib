package main

import (
	"math"
	"testing"
)

func TestRegionLayoutIsSound(t *testing.T) {
	if len(regionLayout) != regionRows {
		t.Fatalf("the layout has %d rows, want %d", len(regionLayout), regionRows)
	}
	oils, veins, cores := 0, 0, 0
	insideOils, insideVeins := 0, 0
	for row, line := range regionLayout {
		if len(line) != regionCols {
			t.Fatalf("row %d is %d tiles wide, want %d", row, len(line), regionCols)
		}
		for col := 0; col < regionCols; col++ {
			var kind byte
			switch kind = tileAt(col, row); kind {
			case kindOil:
				oils++
			case kindLilac:
				veins++
			case kindCore:
				cores++
				if col != coreCol || row != coreRow {
					t.Errorf("a core tile at %d, %d, want it at %d, %d",
						col, row, coreCol, coreRow)
				}
				continue
			case kindGround:
				continue
			default:
				t.Errorf("tile %d, %d has an unknown kind %q", col, row, kind)
				continue
			}
			if d := tileDistance(col, row); d > fogLineRadius {
				t.Errorf("a feature at %d, %d stands beyond the fog line, %v tiles out",
					col, row, d)
			} else if d <= coreBubbleRadius {
				if kind == kindOil {
					insideOils++
				} else {
					insideVeins++
				}
			}
		}
	}
	if cores != 1 {
		t.Errorf("the layout has %d core tiles, want exactly 1", cores)
	}
	if oils < 6 {
		t.Errorf("the layout has %d oil tiles, want 6 or more", oils)
	}
	if veins < 4 {
		t.Errorf("the layout has %d lilac tiles, want 4 or more", veins)
	}
	// The attrition-free zone must let the colony mine one resource of each
	// type in comfort, from deposits generously long.
	if insideOils < 6 {
		t.Errorf("only %d oil tiles inside the bubble, want 6 or more", insideOils)
	}
	if insideVeins < 6 {
		t.Errorf("only %d lilac tiles inside the bubble, want 6 or more", insideVeins)
	}
	if fogLineRadius <= coreBubbleRadius {
		t.Errorf("the fog line at %v is not beyond the bubble at %v",
			fogLineRadius, coreBubbleRadius)
	}
}

func TestProjectionFitsTheRegionOnScreen(t *testing.T) {
	if x, _ := project(coreCol, coreRow); x != regionOriginX {
		t.Errorf("the core projects at x %v, want the middle %v", x, regionOriginX)
	}
	corners := [][2]float32{
		{0, 0},
		{regionCols - 1, 0},
		{regionCols - 1, regionRows - 1},
		{0, regionRows - 1},
	}
	for _, c := range corners {
		x, y := project(c[0], c[1])
		if x < 0 || x > screenWidth || y < 0 || y > screenHeight {
			t.Errorf("corner %v, %v projects at %v, %v, off the screen",
				c[0], c[1], x, y)
		}
	}
}

func TestCirclesFlattenAsTheProjectionSays(t *testing.T) {
	halfW, halfH := ellipseSemiAxes(1)
	want := float32(tileW) / float32(tileH)
	if got := halfW / halfH; math.Abs(float64(got-want)) > 0.001 {
		t.Errorf("a world circle flattens to %v wide for %v high, want %v",
			got, want, 1/want)
	}
}
