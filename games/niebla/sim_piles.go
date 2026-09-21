package main

import (
	"math"
	"sort"
)

// Demolition and loose items. What is built can be unbuilt: a building
// or a site leaves the state at once, and everything it was made of or
// held falls on its cell as one pile, which the robots haul back to the
// stores. Nothing goes straight back: a refund is a haul.
const (
	demolishRefund = 1.0 // the part of a blueprint's cost that falls to the ground

	pileDust = 0.0001 // under this, a pile's amount counts as nothing

	storeStandoff = 11.0 // u from a store's middle to where a robot unloads
)

func sortedPileIDs(s *State) []int64 {
	ids := make([]int64, 0, len(s.Piles))
	for id := range s.Piles {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// pileAt returns the pile lying on a cell, if any.
func pileAt(s *State, col, row int) (Pile, bool) {
	for _, id := range sortedPileIDs(s) {
		if p := s.Piles[id]; p.Col == col && p.Row == row {
			return p, true
		}
	}
	return Pile{}, false
}

// pilesOnTile returns every pile lying on a tile, in ID order.
func pilesOnTile(s *State, tcol, trow int) []Pile {
	var found []Pile
	for _, id := range sortedPileIDs(s) {
		p := s.Piles[id]
		if ccol, crow := cellTile(p.Col, p.Row); ccol == tcol && crow == trow {
			found = append(found, p)
		}
	}
	return found
}

// dropPile leaves loose items on a cell, joining the pile already there
// when there is one. Nothing to drop leaves no pile.
func (s *State) dropPile(col, row int, oil, lilac float64) {
	if oil < pileDust && lilac < pileDust {
		return
	}
	// A save from before the piles loads with no table for them.
	if s.Piles == nil {
		s.Piles = map[int64]Pile{}
	}
	p, ok := pileAt(s, col, row)
	if !ok {
		p = Pile{ID: s.NextID, Col: col, Row: row}
		s.NextID++
	}
	p.Oil += oil
	p.Lilac += lilac
	s.Piles[p.ID] = p
}

// canDemolish reports whether a building may go. A protector stays
// while it is the only bubble over another building or a site, so the
// fog's law — nothing but a protector outside a bubble — can't be
// broken by taking one away.
func canDemolish(s *State, b Building) bool {
	if b.Kind != BuildingProtector {
		return true
	}
	for _, id := range sortedBuildingIDs(s) {
		other := s.Buildings[id]
		if other.ID == b.ID || other.Kind == BuildingProtector {
			continue
		}
		x, y := cellCenterUnits(other.Col, other.Row)
		if !shelteredWithout(s, x, y, b.ID) {
			return false
		}
	}
	for _, job := range s.Jobs {
		if job.Kind == BuildingProtector {
			continue
		}
		x, y := cellCenterUnits(job.Col, job.Row)
		if !shelteredWithout(s, x, y, b.ID) {
			return false
		}
	}
	return true
}

// spillOverflow moves the lilac the stores no longer have a roof for
// onto a cell: lilac is one stock under many roofs, so what a warehouse
// held is the part that stops fitting when its roof goes. Oil has a
// place, and a demolished tank drops its own (Demolish).
func (s *State) spillOverflow(col, row int) {
	lilac := math.Max(0, s.Stock.Lilac-lilacCap(s))
	s.Stock.Lilac -= lilac
	s.dropPile(col, row, 0, lilac)
}

// freeRoom returns the room the stores have left for a cargo, counting
// what the robots already carry home, so nobody loads what won't fit.
func freeRoom(s *State, cargo ThingType) float64 {
	room := oilCap(s) - oilTotal(s)
	if cargo == TypeLilac {
		room = lilacCap(s) - s.Stock.Lilac
	}
	for _, id := range sortedRobotIDs(s) {
		if r := s.Robots[id]; r.Cargo == cargo {
			room -= r.Carry
		}
	}
	return math.Max(0, room)
}

// pileOffer returns what a robot would take from a pile in one trip:
// one kind, lilac first, as much as its arms and the stores' room
// allow. An amount of zero means the pile has nothing for now.
func pileOffer(s *State, p Pile) (cargo ThingType, amount float64) {
	lilac := math.Min(math.Min(p.Lilac, robotCarryLilac), freeRoom(s, TypeLilac))
	if lilac >= pileDust {
		return TypeLilac, lilac
	}
	oil := math.Min(math.Min(p.Oil, robotCarryOil), freeRoom(s, TypeOil))
	if oil >= pileDust {
		return TypeOil, oil
	}
	return "", 0
}

// nearestPile returns the closest pile that holds something the stores
// have room for; IDs break ties.
func nearestPile(s *State, r Robot) (Pile, bool) {
	var best Pile
	found, bestDist := false, 0.0
	for _, id := range sortedPileIDs(s) {
		p := s.Piles[id]
		if _, amount := pileOffer(s, p); amount <= 0 {
			continue
		}
		x, y := cellCenterUnits(p.Col, p.Row)
		if d := math.Hypot(r.X-x, r.Y-y); !found || d < bestDist {
			best, found, bestDist = p, true, d
		}
	}
	return best, found
}

// takeFromPile fills the robot's arms from the pile it was loading at.
// A pile that left in the meantime, or has nothing the stores can take,
// leaves the robot empty-handed. The last item takes the pile with it.
func (s *State) takeFromPile(r *Robot) {
	p, ok := s.Piles[r.Pile]
	r.Pile = 0
	if !ok {
		return
	}
	cargo, amount := pileOffer(s, p)
	if amount <= 0 {
		return
	}
	if cargo == TypeLilac {
		p.Lilac -= amount
	} else {
		p.Oil -= amount
	}
	r.Carry, r.Cargo = amount, cargo
	if p.Oil < pileDust && p.Lilac < pileDust {
		delete(s.Piles, p.ID)
		return
	}
	s.Piles[p.ID] = p
}

// storeSpot returns where a robot unloads. Lilac is one stock, so the
// nearest warehouse or the core is only where the walking ends; oil goes
// into the tank it is carried to, the nearest with room for it
// (haulTank). Robots spread around a store by ID, as builders do.
func storeSpot(s *State, r Robot) (x, y float64) {
	x, y = parkSpot(r.ID)
	angle := float64(r.ID) * goldenAngle
	if r.Cargo == TypeOil {
		tank := haulTank(s, r)
		if tank == coreTank {
			return x, y
		}
		spot, _ := tankSpot(s, tank)
		return spot.X + math.Cos(angle)*storeStandoff,
			spot.Y + math.Sin(angle)*storeStandoff
	}
	best := math.Hypot(r.X-x, r.Y-y)
	for _, id := range sortedBuildingIDs(s) {
		b := s.Buildings[id]
		if b.Kind != BuildingWarehouse {
			continue
		}
		cx, cy := cellCenterUnits(b.Col, b.Row)
		cx += math.Cos(angle) * storeStandoff
		cy += math.Sin(angle) * storeStandoff
		if d := math.Hypot(r.X-cx, r.Y-cy); d < best {
			best, x, y = d, cx, cy
		}
	}
	return x, y
}
