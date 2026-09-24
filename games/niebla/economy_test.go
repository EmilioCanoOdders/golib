package main

import (
	"encoding/csv"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"
)

const economySampleTicks = 60 * 60 // one minute of game time

type economyPlan struct {
	name       string
	workerGoal int
	expand     bool
	guardAt    int64
}

var economyPlans = []economyPlan{
	{name: "safe-harvest", workerGoal: startingRobots},
	{name: "growth", workerGoal: 6},
	{name: "outpost", workerGoal: 6, expand: true, guardAt: 10 * economySampleTicks},
}

type economyPlanner struct {
	plan        economyPlan
	factoryCol  int
	factoryRow  int
	guardCol    int
	guardRow    int
	farOil      Deposit
	hasFarOil   bool
	farLilac    Deposit
	hasFarLilac bool
}

// TestWriteEconomyReport runs real, deterministic colonies over three
// opening plans and seeds. Set NIEBLA_ECONOMY_REPORT to a CSV path to write
// a report; ordinary tests leave no diagnostics behind.
func TestWriteEconomyReport(t *testing.T) {
	path := os.Getenv("NIEBLA_ECONOMY_REPORT")
	if path == "" {
		t.Skip("set NIEBLA_ECONOMY_REPORT to write the economy report")
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err := w.Write([]string{
		"plan", "seed", "minute", "oil", "protector_oil", "lilac", "oil_mined",
		"lilac_mined", "workers", "troopers", "buildings", "protectors",
		"pumps", "pipes", "guards", "swells", "visits", "enemies",
	}); err != nil {
		t.Fatalf("writing the report header: %v", err)
	}
	for _, plan := range economyPlans {
		for _, seed := range []int64{0, 1, 2} {
			writeEconomyRun(t, w, plan, seed, 60)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func writeEconomyRun(
	t *testing.T,
	w *csv.Writer,
	plan economyPlan,
	seed int64,
	minutes int,
) {
	t.Helper()
	s := newGameOn(seed)
	arriveAll(s) // the planner builds the whole ladder from the start
	p := newEconomyPlanner(plan)
	for minute := 0; minute <= minutes; minute++ {
		if err := w.Write(economyRow(s, plan.name, seed, minute)); err != nil {
			t.Fatalf("writing %s, seed %d, minute %d: %v", plan.name, seed, minute, err)
		}
		if minute == minutes {
			break
		}
		for tick := 0; tick < economySampleTicks; tick++ {
			if s.Ticks%60 == 0 {
				p.decide(s)
			}
			Apply(s, Tick{})
		}
	}
}

func newEconomyPlanner(plan economyPlan) economyPlanner {
	factoryCol, factoryRow := groundNearCore()
	p := economyPlanner{
		plan:       plan,
		factoryCol: factoryCol,
		factoryRow: factoryRow,
		guardCol:   factoryCol + 2,
		guardRow:   factoryRow,
	}
	for _, deposit := range depositsByDistance() {
		if depositDistance(deposit) <= coreBubbleRadius {
			continue
		}
		if deposit.Kind == kindOil && !p.hasFarOil {
			p.farOil, p.hasFarOil = deposit, true
		}
		if deposit.Kind == kindLilac && !p.hasFarLilac {
			p.farLilac, p.hasFarLilac = deposit, true
		}
	}
	return p
}

func (p economyPlanner) decide(s *State) {
	p.assignPosts(s)
	if p.plan.workerGoal > startingRobots {
		p.growWorkers(s)
	}
	if p.plan.expand {
		p.expandOil(s)
	}
	if p.plan.guardAt > 0 && s.Ticks >= p.plan.guardAt {
		p.mark(s, BuildingGuard, p.guardCol, p.guardRow)
	}
}

func (p economyPlanner) assignPosts(s *State) {
	for _, deposit := range depositsByDistance() {
		tileCol, tileRow := cellTile(deposit.HeartCol, deposit.HeartRow)
		far := depositDistance(deposit) > coreBubbleRadius
		if !p.plan.expand && far {
			continue
		}
		if p.plan.expand && far && !depositSafe(s, deposit) {
			continue
		}
		if len(postRobots(s, tileCol, tileRow)) == 0 {
			Apply(s, SendRobot{Col: tileCol, Row: tileRow})
		}
	}
}

func (p economyPlanner) growWorkers(s *State) {
	if !p.mark(s, BuildingFactory, p.factoryCol, p.factoryRow) {
		return
	}
	if workerCount(s) >= p.plan.workerGoal {
		return
	}
	if factory, raised := buildingAt(s, p.factoryCol, p.factoryRow); raised {
		Apply(s, QueueRobot{Building: factory.ID})
	}
}

func (p economyPlanner) expandOil(s *State) {
	if !p.hasFarOil {
		return
	}
	col, row, found := protectorSite(p.farOil)
	if !found || !p.mark(s, BuildingProtector, col, row) {
		return
	}
	pumpCol, pumpRow := pumpCell(p.farOil)
	if !p.mark(s, BuildingPump, pumpCol, pumpRow) {
		return
	}
	protector, raised := buildingAt(s, col, row)
	if !raised {
		return
	}
	pump, raised := buildingAt(s, pumpCol, pumpRow)
	if !raised {
		return
	}
	if canJoin(s, pump.ID, protector.ID) {
		Apply(s, LayPipe{From: pump.ID, To: protector.ID})
		return
	}
	if canJoin(s, protector.ID, coreTank) {
		Apply(s, LayPipe{From: protector.ID, To: coreTank})
	}
}

// mark asks for a building until it is raised. Its true answer means the
// site or building now stands, so later parts of a plan may depend on it.
func (p economyPlanner) mark(s *State, kind BuildingKind, col, row int) bool {
	if building, raised := buildingAt(s, col, row); raised {
		return building.Kind == kind
	}
	for _, job := range s.Jobs {
		if job.Col == col && job.Row == row {
			return job.Kind == kind
		}
	}
	Apply(s, MarkBuilding{Kind: kind, Col: col, Row: row})
	return false
}

// protectorSite finds flat ground close enough to cover a far deposit's
// pump. The generator guarantees the heart is flat; its body may not leave
// a free cell for the protector itself.
func protectorSite(d Deposit) (col, row int, found bool) {
	for radius := 1; radius <= 16; radius++ {
		for rowOffset := -radius; rowOffset <= radius; rowOffset++ {
			for colOffset := -radius; colOffset <= radius; colOffset++ {
				if colOffset != -radius && colOffset != radius &&
					rowOffset != -radius && rowOffset != radius {
					continue
				}
				col, row = d.HeartCol+colOffset, d.HeartRow+rowOffset
				if col < 0 || row < 0 || col >= regionCellCols || row >= regionCellRows ||
					tileAt(cellTile(col, row)) != kindGround || !land.flatCell(col, row) {
					continue
				}
				x, y := cellCenterUnits(col, row)
				px, py := cellCenterUnits(d.HeartCol, d.HeartRow)
				if math.Hypot(x-px, y-py) <= protectorBubbleTiles*unitsPerTile-buildingCell {
					return col, row, true
				}
			}
		}
	}
	return 0, 0, false
}

func depositsByDistance() []Deposit {
	deposits := append([]Deposit(nil), land.deposits...)
	sort.Slice(deposits, func(i, j int) bool {
		return depositDistance(deposits[i]) < depositDistance(deposits[j])
	})
	return deposits
}

func depositDistance(d Deposit) float32 {
	col, row := cellTile(d.HeartCol, d.HeartRow)
	return tileDistance(col, row)
}

func depositSafe(s *State, d Deposit) bool {
	x, y := cellCenterUnits(d.HeartCol, d.HeartRow)
	return inSafeZone(s, x, y)
}

func economyRow(s *State, plan string, seed int64, minute int) []string {
	oilMined, lilacMined := mined(s)
	workers, troopers := robotCounts(s)
	buildings, protectors, pumps, guards := buildingCounts(s)
	return []string{
		plan,
		strconv.FormatInt(seed, 10),
		strconv.Itoa(minute),
		quantity(oilTotal(s)),
		quantity(protectorOilTotal(s)),
		quantity(s.Stock.Lilac),
		quantity(oilMined),
		quantity(lilacMined),
		strconv.Itoa(workers),
		strconv.Itoa(troopers),
		strconv.Itoa(buildings),
		strconv.Itoa(protectors),
		strconv.Itoa(pumps),
		strconv.Itoa(len(s.Pipes)),
		strconv.Itoa(guards),
		strconv.FormatInt(s.Fog.Swells, 10),
		strconv.FormatInt(s.Raids.Visits, 10),
		strconv.Itoa(len(s.Enemies)),
	}
}

func mined(s *State) (oil, lilac float64) {
	for _, deposit := range land.deposits {
		mined := deposit.Full - s.Drain[depositKey(deposit)]
		if deposit.Kind == kindOil {
			oil += mined
		} else {
			lilac += mined
		}
	}
	return oil, lilac
}

func robotCounts(s *State) (workers, troopers int) {
	for _, robot := range s.Robots {
		if robot.Kind == RobotCombat {
			troopers++
		} else {
			workers++
		}
	}
	return workers, troopers
}

func workerCount(s *State) int {
	workers, _ := robotCounts(s)
	return workers
}

func buildingCounts(s *State) (all, protectors, pumps, guards int) {
	for _, building := range s.Buildings {
		all++
		switch building.Kind {
		case BuildingProtector:
			protectors++
		case BuildingPump:
			pumps++
		case BuildingGuard:
			guards++
		}
	}
	return all, protectors, pumps, guards
}

func quantity(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}
