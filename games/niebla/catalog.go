package main

import (
	"fmt"
	"strings"

	"golib"
)

// ThingInfo is the catalog's entry for a thing type: what it is called, the
// color everything about it is written in, and the lines its card shows.
// The catalog is metadata, never state: it is the database the views read
// to print any entity. A type it has no entry for still gets a name and a
// stable color out of the fallback, so new things print the moment they
// exist and get dressed later.
type ThingInfo struct {
	Name    string
	Color   golib.Color
	Unit    string                      // the SI unit of the type's Amount
	Summary func(amount float64) string // the card's headline; default si()
	Details func(thing Thing) []Detail  // the expanded card's lines
}

// summarize returns the card's headline for a thing's amount.
func (info ThingInfo) summarize(amount float64) string {
	if info.Summary != nil {
		return info.Summary(amount)
	}
	if info.Unit == "" {
		return fmt.Sprintf("%.0f", amount)
	}
	return si(amount, info.Unit)
}

// catalog holds an entry per thing type so far.
var catalog = map[ThingType]ThingInfo{
	TypeOil: {
		Name:  "Oil pool",
		Color: oilColor,
		Unit:  "L",
		Details: func(Thing) []Detail {
			return []Detail{
				{"amount", fmt.Sprintf("[oil]%s[/]", si(oilPerPoolTile, "L"))},
				{"tile", tileAreaLabel},
				{"state", "intact"},
			}
		},
	},
	TypeLilac: {
		Name:  "Lilac vein",
		Color: lilacColor,
		Unit:  "kg",
		Details: func(Thing) []Detail {
			return []Detail{
				{"amount", fmt.Sprintf("[lilac]%s[/]", si(lilacPerVeinTile, "kg"))},
				{"tile", tileAreaLabel},
				{"state", "intact"},
			}
		},
	},
	TypeCore: {
		Name:  "Repelling core",
		Color: coreGlowColor,
		Unit:  "m",
		Summary: func(amount float64) string {
			return "r = " + si(amount, "m")
		},
		Details: func(Thing) []Detail {
			return []Detail{
				{"height", si(coreHeight, "m")},
				{"pole", si(corePoleAcross, "m") + " across"},
				{"bubble", "r = " + si(coreBubbleMeters(), "m")},
				{"upkeep", "none"},
				{"integrity", "[core]indestructible[/]"},
			}
		},
	},
}

// catalogInfo returns the entry for a thing type. A type the catalog has no
// entry for yet takes its name from the type and its color from the type's
// name, which never changes.
func catalogInfo(kind ThingType) ThingInfo {
	if info, ok := catalog[kind]; ok {
		return info
	}
	name := string(kind)
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return ThingInfo{Name: name, Color: stableColor(string(kind))}
}

// stableColor turns a name into a color that stays the same for that name:
// the hue spreads the types around the wheel, washed out and bright enough
// to sit on the game's cold palette.
func stableColor(name string) golib.Color {
	h := 2166136261
	for _, r := range name {
		h ^= int(r)
		h *= 16777619
	}
	if h < 0 {
		h = -h
	}
	return hsv(float32(h%360)/360, 0.45, 0.85)
}

// hsv turns a color from hue (0 to 1), saturation and value into RGB.
func hsv(h, s, v float32) golib.Color {
	sector := h * 6
	i := int(sector) % 6
	f := sector - float32(int(sector))
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	to8 := func(x float32) uint8 { return uint8(x * 255) }
	switch i {
	case 0:
		return golib.Color{R: to8(v), G: to8(t), B: to8(p), A: 255}
	case 1:
		return golib.Color{R: to8(q), G: to8(v), B: to8(p), A: 255}
	case 2:
		return golib.Color{R: to8(p), G: to8(v), B: to8(t), A: 255}
	case 3:
		return golib.Color{R: to8(p), G: to8(q), B: to8(v), A: 255}
	case 4:
		return golib.Color{R: to8(t), G: to8(p), B: to8(v), A: 255}
	default:
		return golib.Color{R: to8(v), G: to8(p), B: to8(q), A: 255}
	}
}

// markupPalette holds the colors text can name with markup, "[name]...[/]":
// one per thing type, from the catalog, plus a few that say how text is
// written rather than what it is about.
var markupPalette = map[string]golib.Color{}

func init() {
	for kind, info := range catalog {
		markupPalette[string(kind)] = info.Color
	}
	markupPalette["dim"] = panelDimColor
	markupPalette["light"] = panelTextColor
	markupPalette["fog"] = fogColor
}
