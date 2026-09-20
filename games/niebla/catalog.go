package main

import (
	"fmt"
	"strings"

	"golib"
)

// ThingInfo is the catalog's entry for a thing type: what it is called, the
// color everything about it is written in, and the lines its card shows.
// The catalog is metadata, never state: it is the database the views read
// to print any entity. The card lines read the state, so they can say
// what remains in a deposit or what a robot is doing. A type it has no
// entry for still gets a name and a stable color out of the fallback, so
// new things print the moment they exist and get dressed later.
type ThingInfo struct {
	Name    string
	Color   golib.Color
	Unit    string                               // the SI unit of the type's Amount
	Summary func(amount float64) string          // the card's headline; default si()
	Details func(s *State, thing Thing) []Detail // the expanded card's lines
}

// summarize returns the card's headline for a thing.
func (info ThingInfo) summarize(thing Thing) string {
	if thing.Caption != "" {
		return thing.Caption
	}
	if info.Summary != nil {
		return info.Summary(thing.Amount)
	}
	if info.Unit == "" {
		return fmt.Sprintf("%.0f", thing.Amount)
	}
	return si(thing.Amount, info.Unit)
}

// catalog holds an entry per thing type so far.
var catalog = map[ThingType]ThingInfo{
	TypeOil: {
		Name:  "Oil pool",
		Color: oilColor,
		Unit:  "L",
		Details: func(s *State, thing Thing) []Detail {
			return []Detail{
				{"amount", fmt.Sprintf("[oil]%s[/]", si(thing.Amount, "L"))},
				{"tile", tileAreaLabel},
				{"state", depositState(thing.Amount, oilPerPoolTile)},
			}
		},
	},
	TypeLilac: {
		Name:  "Lilac vein",
		Color: lilacColor,
		Unit:  "kg",
		Details: func(s *State, thing Thing) []Detail {
			return []Detail{
				{"amount", fmt.Sprintf("[lilac]%s[/]", si(thing.Amount, "kg"))},
				{"tile", tileAreaLabel},
				{"state", depositState(thing.Amount, lilacPerVeinTile)},
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
		Details: func(s *State, thing Thing) []Detail {
			return []Detail{
				{"height", si(coreHeight, "m")},
				{"pole", si(corePoleAcross, "m") + " across"},
				{"bubble", "r = " + si(coreBubbleMeters(), "m")},
				{"upkeep", "none"},
				{"integrity", "[core]indestructible[/]"},
				{"oil store", fmt.Sprintf("[oil]%s[/]", si(s.Stock.Oil, "L"))},
				{"lilac store", fmt.Sprintf("[lilac]%s[/]", si(s.Stock.Lilac, "kg"))},
				{"robots", fmt.Sprintf("%d", len(s.Robots))},
			}
		},
	},
	TypeRobot: {
		Name:  "Robot",
		Color: robotColor,
		Details: func(s *State, thing Thing) []Detail {
			r, ok := s.Robots[thing.Ref]
			if !ok {
				return nil
			}
			details := []Detail{
				{"task", robotCaption(s, r)},
			}
			if r.Carry > 0 {
				unit, kind := "kg", r.Cargo
				if kind == TypeOil {
					unit = "L"
				}
				details = append(details, Detail{
					"carrying", fmt.Sprintf("[%s]%s[/]", kind, si(r.Carry, unit)),
				})
			}
			if r.hasPost() {
				details = append(details, Detail{
					"post", fmt.Sprintf("tile %d, %d", r.PostCol, r.PostRow),
				})
			}
			return details
		},
	},
}

// depositState says how a deposit tile reads after the robots worked it.
func depositState(amount, full float64) string {
	switch {
	case amount <= 0:
		return "dry"
	case amount < full:
		return "worked"
	}
	return "intact"
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
