package main

import (
	"testing"

	"golib"
)

func near(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.01
}

// A kind with no group is one the player can never mark, and a kind in
// two shows up twice on a ring.
func TestEveryBlueprintSitsInExactlyOneGroup(t *testing.T) {
	want := map[BuildingKind]bool{
		BuildingFactory:    true,
		BuildingCharger:    true,
		BuildingSilo:       true,
		BuildingWarehouse:  true,
		BuildingProtector:  true,
		BuildingGuard:      true,
		BuildingWarFactory: true,
		BuildingArtillery:  true,
	}
	seen := map[BuildingKind]int{}
	for _, group := range buildGroups {
		members := groupMembers[group]
		if len(members) == 0 {
			t.Errorf("group %q has no blueprints", group)
		}
		for _, kind := range members {
			seen[kind]++
		}
	}
	for kind := range want {
		if seen[kind] != 1 {
			t.Errorf("%s sits in %d groups, want 1", kind, seen[kind])
		}
	}
	for kind, n := range seen {
		if !want[kind] {
			t.Errorf("%s sits in %d groups, want none", kind, n)
		}
	}
}

func TestEveryBlueprintReadsInItsOwnColor(t *testing.T) {
	for _, group := range buildGroups {
		for _, kind := range groupMembers[group] {
			info := catalogInfo(buildingType(kind))
			if info.Name == "" {
				t.Errorf("%s has no name to label it with", kind)
			}
			if info.Color == (golib.Color{}) {
				t.Errorf("%s has no color to ring it with", kind)
			}
		}
	}
}

func TestRadialSpotStartsAtTheTop(t *testing.T) {
	x, y := radialSpot(3, 0, 100, 200)
	if !near(x, 100) || !near(y, 200-radialRadius) {
		t.Errorf("first of 3 sits at %v, %v, want 100, %v",
			x, y, 200-radialRadius)
	}
	x, y = radialSpot(4, 1, 100, 200)
	if !near(x, 100+radialRadius) || !near(y, 200) {
		t.Errorf("second of 4 sits at %v, %v, want %v, 200",
			x, y, 100+radialRadius)
	}
}

func TestRadialSpotsShareOneRing(t *testing.T) {
	for n := 2; n <= 5; n++ {
		for i := 0; i < n; i++ {
			x, y := radialSpot(n, i, 100, 200)
			d := float32(0)
			dx, dy := x-100, y-200
			d = dx*dx + dy*dy
			if !near(d, radialRadius*radialRadius) {
				t.Errorf("option %d of %d sits %v from the center, want %v",
					i, n, d, radialRadius*radialRadius)
			}
		}
	}
}

func TestRadialHit(t *testing.T) {
	if !radialHit(50, 50, 50, 50) {
		t.Error("an option's center is on it")
	}
	if !radialHit(50, 50, 50, 50+radialItemR+3) {
		t.Error("just inside the margin is on it")
	}
	if radialHit(50, 50, 50, 50+radialItemR+8) {
		t.Error("past the margin is off it")
	}
}

// A group with no members, or a blueprints sitting in two rings, is a
// menu the player can lose a pick in: the layout's shape is the guard.
func TestRadialRingLaysOutEveryOption(t *testing.T) {
	s := newPlayScene(newGame())
	arriveAll(s.state)
	s.openRadial(104, 96)
	groups := radialGroupLayout(s)
	if len(groups) != len(buildGroups) {
		t.Errorf("the first ring has %d options, want %d",
			len(groups), len(buildGroups))
	}
	for i, item := range groups {
		if item.group != buildGroups[i] {
			t.Errorf("option %d is %q, want %q", i, item.group, buildGroups[i])
		}
	}
	s.radialGroup = groupIndustry
	s.radialLevel = 1
	leaves := radialLeafLayout(s)
	if len(leaves) != len(groupMembers[groupIndustry]) {
		t.Errorf("the industry ring has %d options, want %d",
			len(leaves), len(groupMembers[groupIndustry]))
	}
	for i, item := range leaves {
		if item.kind != groupMembers[groupIndustry][i] {
			t.Errorf("option %d is %q, want %q",
				i, item.kind, groupMembers[groupIndustry][i])
		}
	}
}

func TestRadialGoesBackARingAndThenCloses(t *testing.T) {
	s := newPlayScene(newGame())
	s.openRadial(104, 96)
	if !s.radial || s.radialLevel != 0 {
		t.Fatalf("the menu opened on ring %d, want 0", s.radialLevel)
	}
	s.radialGroup = groupMilitary
	s.radialLevel = 1
	s.backRadial()
	if !s.radial || s.radialLevel != 0 {
		t.Errorf("a right click on the second ring left ring %d open=%v",
			s.radialLevel, s.radial)
	}
	s.backRadial()
	if s.radial {
		t.Error("a right click on the first ring left the menu open")
	}
}

// The menu only offers what the colony could really raise: a fresh
// colony's rings are empty, infrastructure alone shows one group, an
// empty store empties the rings, and the military ring holds only the
// guard post once a party drives in.
func TestTheRadialOnlyOffersWhatArrived(t *testing.T) {
	s := newPlayScene(newGame())
	s.openRadial(104, 96)
	if groups := radialGroupLayout(s); len(groups) != 0 {
		t.Fatalf("a fresh colony's menu offers %d groups, want none", len(groups))
	}
	Apply(s.state, DevNextTech{})
	runTicks(s.state, 1)
	groups := radialGroupLayout(s)
	if len(groups) != 1 || groups[0].group != groupLogistics {
		t.Fatalf("infrastructure alone shows %d groups, want logistics only",
			len(groups))
	}
	s.radialGroup, s.radialLevel = groupLogistics, 1
	if leaves := radialLeafLayout(s); len(leaves) != 3 {
		t.Fatalf("infrastructure offers %d logistics options, want 3",
			len(leaves))
	}
	// What the stores can't pay for is not offered either.
	lilac := s.state.Stock.Lilac
	s.state.Stock.Lilac = 0
	if groups := radialGroupLayout(s); len(groups) != 0 {
		t.Fatalf("an empty store still offers %d groups, want none", len(groups))
	}
	s.state.Stock.Lilac = lilac
	// The guard post comes with the scout's mark on the ground, alone.
	s.state.Marks[1] = Mark{ID: 1, X: 100, Y: 100}
	runTicks(s.state, 1)
	if !kindUnlocked(s.state, BuildingGuard) {
		t.Fatal("the guard post never answered the mark")
	}
	groups = radialGroupLayout(s)
	if len(groups) != 2 {
		t.Fatalf("infrastructure and the guard show %d groups, want 2", len(groups))
	}
	s.radialGroup, s.radialLevel = groupMilitary, 1
	leaves := radialLeafLayout(s)
	if len(leaves) != 1 || leaves[0].kind != BuildingGuard {
		t.Fatalf("the military ring offers %v, want the guard post alone", leaves)
	}
}

func TestProtectorBelongsToTheLogisticsRing(t *testing.T) {
	s := newPlayScene(newGame())
	seedStock(s.state)
	arriveAll(s.state)
	s.openRadial(groundNearCore())
	s.radialGroup = groupLogistics
	s.radialLevel = 1

	leaves := radialLeafLayout(s)
	for _, item := range leaves {
		if item.kind == BuildingProtector {
			return
		}
	}
	t.Fatal("the logistics ring does not offer the protector")
}
