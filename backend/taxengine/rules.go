// Package taxengine computes Indian individual income tax under the old and
// new regimes. It is pure (no I/O) and driven by per-financial-year rule
// tables, so a new budget is a data change in this file.
package taxengine

import (
	"fmt"
	"math"
	"strings"
)

type Regime string

const (
	Old Regime = "old"
	New Regime = "new"
)

type Slab struct {
	UpTo float64 // upper bound of the slab; math.Inf(1) for the last slab
	Rate float64 // percent
}

type SurchargeBand struct {
	Above float64 // total income above which Rate applies
	Rate  float64 // percent
}

// Rules holds every amount that differs between financial years.
type Rules struct {
	FY       string
	Act      string // law under which the year is assessed
	NewAct   bool   // Income-tax Act, 2025 (tax year 2026-27 onwards)
	NewSlabs []Slab
	// Old regime slabs by age: below 60, 60–79 (resident), 80+ (resident).
	OldSlabs            []Slab
	OldSlabsSenior      []Slab
	OldSlabsSuperSenior []Slab

	StdDeductionNew, StdDeductionOld   float64
	FamilyPensionNew, FamilyPensionOld float64 // cap on 1/3 family pension deduction

	RebateNewLimit, RebateNewMax float64
	RebateOldLimit, RebateOldMax float64

	STCGEquityRate      float64 // 111A
	LTCGEquityRate      float64 // 112A
	LTCGEquityExemption float64 // 112A threshold per person per year
	LTCGOtherRate       float64 // 112 (listed bonds, gold, property etc.)

	Lim80C, Lim80CCD1B                    float64
	Lim80DSelf, Lim80DSelfSenior          float64
	Lim80DParents, Lim80DParentsSenior    float64
	Lim80TTA, Lim80TTB                    float64
	Lim24bSelfOccupied, LimHPLossSetOff   float64
	Lim80U, Lim80USevere, Lim80GGPerMonth float64
	EmployerNPSPctNew, EmployerNPSPctOld  float64 // % of basic+DA allowed under 80CCD(2)
	EmployeeNPSPctSalaried                float64 // 80CCD(1) cap as % of basic+DA
	SelfEmployedNPSPct                    float64 // 80CCD(1) cap as % of gross total income

	SurchargeOld, SurchargeNew []SurchargeBand
	SurchargeSpecialCap        float64 // max surcharge on 111A/112A/112 tax
	CessRate                   float64
	MinorClubbingExemption     float64 // per minor child, on clubbed income
}

var inf = math.Inf(1)

func baseRules2025() Rules {
	return Rules{
		NewSlabs:            []Slab{{400000, 0}, {800000, 5}, {1200000, 10}, {1600000, 15}, {2000000, 20}, {2400000, 25}, {inf, 30}},
		OldSlabs:            []Slab{{250000, 0}, {500000, 5}, {1000000, 20}, {inf, 30}},
		OldSlabsSenior:      []Slab{{300000, 0}, {500000, 5}, {1000000, 20}, {inf, 30}},
		OldSlabsSuperSenior: []Slab{{500000, 0}, {1000000, 20}, {inf, 30}},

		StdDeductionNew: 75000, StdDeductionOld: 50000,
		FamilyPensionNew: 25000, FamilyPensionOld: 15000,

		RebateNewLimit: 1200000, RebateNewMax: 60000,
		RebateOldLimit: 500000, RebateOldMax: 12500,

		STCGEquityRate: 20, LTCGEquityRate: 12.5, LTCGEquityExemption: 125000, LTCGOtherRate: 12.5,

		Lim80C: 150000, Lim80CCD1B: 50000,
		Lim80DSelf: 25000, Lim80DSelfSenior: 50000,
		Lim80DParents: 25000, Lim80DParentsSenior: 50000,
		Lim80TTA: 10000, Lim80TTB: 50000,
		Lim24bSelfOccupied: 200000, LimHPLossSetOff: 200000,
		Lim80U: 75000, Lim80USevere: 125000, Lim80GGPerMonth: 5000,
		EmployerNPSPctNew: 14, EmployerNPSPctOld: 10,
		EmployeeNPSPctSalaried: 10, SelfEmployedNPSPct: 20,

		SurchargeOld:           []SurchargeBand{{5000000, 10}, {10000000, 15}, {20000000, 25}, {50000000, 37}},
		SurchargeNew:           []SurchargeBand{{5000000, 10}, {10000000, 15}, {20000000, 25}},
		SurchargeSpecialCap:    15,
		CessRate:               4,
		MinorClubbingExemption: 1500,
	}
}

var rulesByFY = func() map[string]*Rules {
	fy2526 := baseRules2025()
	fy2526.FY = "2025-26"
	fy2526.Act = "Income-tax Act, 1961"

	// Budget 2026 kept slabs and limits unchanged; the year is assessed under
	// the Income-tax Act, 2025 (sections renumbered, see sectionMap).
	fy2627 := baseRules2025()
	fy2627.FY = "2026-27"
	fy2627.Act = "Income-tax Act, 2025"
	fy2627.NewAct = true

	return map[string]*Rules{fy2526.FY: &fy2526, fy2627.FY: &fy2627}
}()

// SupportedFYs lists the years with rule tables, oldest first.
func SupportedFYs() []string { return []string{"2025-26", "2026-27"} }

// NormalizeFY accepts "2025-26", "2025-2026" or "2025" and returns "2025-26".
func NormalizeFY(fy string) string {
	fy = strings.TrimSpace(fy)
	var start int
	if _, err := fmt.Sscanf(fy, "%d", &start); err != nil || start < 2000 {
		return ""
	}
	return fmt.Sprintf("%d-%02d", start, (start+1)%100)
}

// RulesFor returns the rules for fy. Unknown years fall back to the closest
// supported year and ok is false.
func RulesFor(fy string) (r *Rules, ok bool) {
	n := NormalizeFY(fy)
	if r, found := rulesByFY[n]; found {
		return r, true
	}
	fys := SupportedFYs()
	if n != "" && n < fys[0] {
		return rulesByFY[fys[0]], false
	}
	return rulesByFY[fys[len(fys)-1]], false
}

// sectionMap maps Income-tax Act, 1961 sections to their Income-tax Act, 2025
// equivalents. Only verified mappings are listed; others show the old number.
var sectionMap = map[string]string{
	"16(ia)":    "19",
	"16(iii)":   "19",
	"24(b)":     "22",
	"80C":       "123",
	"80CCC":     "123",
	"80CCD(1)":  "124",
	"80CCD(1B)": "124",
	"80CCD(2)":  "124",
	"80D":       "126",
	"80E":       "129",
	"80G":       "133",
	"80TTA":     "153",
	"87A":       "156",
	"115BAC":    "202",
	"10(13A)":   "Schedule",
	"10(5)":     "Schedule",
}

// NewSection returns the Income-tax Act, 2025 section for an old section code
// when the year is assessed under the new Act, else "".
func (r *Rules) NewSection(code string) string {
	if !r.NewAct {
		return ""
	}
	return sectionMap[code]
}
