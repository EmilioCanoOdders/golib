package main

import (
	"testing"

	"golib"
)

func TestParseMarkupPlain(t *testing.T) {
	spans := parseMarkup("no colors here", panelTextColor)
	if len(spans) != 1 || spans[0].text != "no colors here" {
		t.Errorf("plain text parsed into %v, want one span", spans)
	}
	if spans[0].color != panelTextColor {
		t.Errorf("plain text took color %v, want the base one", spans[0].color)
	}
}

func TestParseMarkupColorsSpans(t *testing.T) {
	spans := parseMarkup("an [oil]oil pool[/] holds", panelTextColor)
	want := []markupSpan{
		{"an ", panelTextColor},
		{"oil pool", oilColor},
		{" holds", panelTextColor},
	}
	if len(spans) != len(want) {
		t.Fatalf("parsed %v, want %v", spans, want)
	}
	for i, w := range want {
		if spans[i] != w {
			t.Errorf("span %d is %v, want %v", i, spans[i], w)
		}
	}
}

func TestParseMarkupNests(t *testing.T) {
	spans := parseMarkup("[dim]a [oil]b[/] c[/]", panelTextColor)
	want := []markupSpan{
		{"a ", panelDimColor},
		{"b", oilColor},
		{" c", panelDimColor},
	}
	if len(spans) != len(want) {
		t.Fatalf("parsed %v, want %v", spans, want)
	}
	for i, w := range want {
		if spans[i] != w {
			t.Errorf("span %d is %v, want %v", i, spans[i], w)
		}
	}
}

func TestParseMarkupKeepsUnknownTags(t *testing.T) {
	spans := parseMarkup("[nope]x", panelTextColor)
	if len(spans) != 1 || spans[0].text != "[nope]x" {
		t.Errorf("an unknown tag parsed into %v, want it printed as it is", spans)
	}
}

func TestParseMarkupUnclosed(t *testing.T) {
	spans := parseMarkup("a [oil]b", panelTextColor)
	want := []markupSpan{
		{"a ", panelTextColor},
		{"b", oilColor},
	}
	if len(spans) != len(want) {
		t.Fatalf("parsed %v, want %v", spans, want)
	}
	for i, w := range want {
		if spans[i] != w {
			t.Errorf("span %d is %v, want %v", i, spans[i], w)
		}
	}
}

func TestTooltipLayoutRowsAndCards(t *testing.T) {
	camera := golib.NewCamera(screenWidth, screenHeight)
	panel := tooltipLayout(camera, coreCol, coreRow, map[string]bool{})
	titles := 0
	for _, r := range panel.rows {
		if r.title {
			titles++
		}
	}
	if titles != 1 {
		t.Fatalf("the core tile laid %d card titles, want 1", titles)
	}
	open := map[string]bool{"core@12,12": true}
	panel = tooltipLayout(camera, coreCol, coreRow, open)
	details := 0
	for _, r := range panel.rows {
		if r.title {
			if r.summary != "r = 20 m" {
				t.Errorf("the core card's headline is %q, want \"r = 20 m\"", r.summary)
			}
		} else if !r.header {
			details++
		}
	}
	if details != 5 {
		t.Errorf("the expanded core card shows %d details, want 5", details)
	}
	if !panel.contains(panel.x+1, panel.y+1) || panel.contains(panel.x-1, panel.y) {
		t.Errorf("contains answers wrongly around the panel's edges")
	}
	thing, ok := panel.cardAt(panel.x+10, panel.y+tooltipPad+textRow+titleRow/2)
	if !ok || thing.Type != TypeCore {
		t.Errorf("cardAt the title found %v, %v, want the core thing", thing, ok)
	}
	if _, ok := panel.cardAt(panel.x+10, panel.y+tooltipPad+textRow/2); ok {
		t.Errorf("cardAt the header found a card, want none")
	}
}
