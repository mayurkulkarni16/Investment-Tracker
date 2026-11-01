package taxengine

import (
	"math"
	"testing"
)

func other(fy string, income float64) Input {
	return Input{FY: fy, Age: 35, Resident: true, OtherIncome: income}
}

func TestSlabTotals(t *testing.T) {
	cases := []struct {
		name   string
		in     Input
		regime Regime
		want   float64
	}{
		{"new 13L", other("2025-26", 1300000), New, 78000},
		{"new 15L", other("2025-26", 1500000), New, 109200},
		{"new 20L", other("2026-27", 2000000), New, 208000},
		{"new 25L", other("2025-26", 2500000), New, 343200},
		{"new 30L", other("2025-26", 3000000), New, 499200},
		{"new 12L rebate", other("2025-26", 1200000), New, 0},
		{"new 12.1L marginal relief", other("2025-26", 1210000), New, 10400},
		{"old 5L rebate", other("2025-26", 500000), Old, 0},
		{"old 8L", other("2025-26", 800000), Old, 75400},
		{"old 10L", other("2025-26", 1000000), Old, 117000},
		{"old 12L", other("2025-26", 1200000), Old, 179400},
		{"old 13L", other("2026-27", 1300000), Old, 210600},
		{"new 60L surcharge", other("2025-26", 6000000), New, 1578720},
		{"new 50.1L surcharge relief", other("2025-26", 5010000), New, 1133600},
		{"salaried 12.75L new", Input{FY: "2025-26", Age: 30, Resident: true, GrossSalary: 1275000, BasicDA: 600000}, New, 0},
		{"senior old 3L", Input{FY: "2025-26", Age: 65, Resident: true, OtherIncome: 300000}, Old, 0},
		{"super senior old 5L", Input{FY: "2025-26", Age: 82, Resident: true, OtherIncome: 500000}, Old, 0},
		{"super senior old 6L", Input{FY: "2025-26", Age: 82, Resident: true, OtherIncome: 600000}, Old, 20800},
		// 2L slab income + 3L STCG: 2L of unused exemption absorbs STCG, 1L @20%, no rebate on special income.
		{"stcg uses basic exemption", Input{FY: "2025-26", Age: 40, Resident: true, OtherIncome: 200000, STCGEquity: 300000}, New, 20800},
		{"ltcg within exemption + basic", Input{FY: "2025-26", Age: 40, Resident: true, LTCGEquity: 300000}, New, 0},
	}
	for _, c := range cases {
		if got := Total(c.in, c.regime); got != c.want {
			t.Errorf("%s: got %.0f want %.0f", c.name, got, c.want)
		}
	}
}

func TestHRAExemption(t *testing.T) {
	if got := HRAExemption(50000, 20000, 25000, true); got != 20000 {
		t.Fatalf("metro HRA got %v", got)
	}
	if got := HRAExemption(50000, 30000, 25000, false); got != 20000 {
		t.Fatalf("non-metro HRA got %v", got) // 40% of basic = 20000
	}
	if got := HRAExemption(50000, 20000, 0, true); got != 0 {
		t.Fatalf("no rent got %v", got)
	}
}

func TestCompareSalaried(t *testing.T) {
	in := Input{FY: "2025-26", Age: 34, Resident: true,
		GrossSalary: 1500000, BasicDA: 600000, HRAExempt: 200000, HRAReceived: 240000,
		Ded80C: 150000, OwnNPS: 50000, Ded80DSelf: 25000}
	c := Compare(in)
	if c.Old.TotalTax != 124800 {
		t.Errorf("old tax got %.0f want 124800", c.Old.TotalTax)
	}
	if c.New.TotalTax != 97500 {
		t.Errorf("new tax got %.0f want 97500", c.New.TotalTax)
	}
	if c.Recommended != New || c.Savings != 27300 {
		t.Errorf("recommended %s savings %.0f", c.Recommended, c.Savings)
	}
	if c.BreakEvenDeductions <= 0 {
		t.Fatalf("expected positive break-even, got %.0f", c.BreakEvenDeductions)
	}
	in.ExtraDeductionOld = c.BreakEvenDeductions
	if Total(in, Old) > c.New.TotalTax {
		t.Errorf("break-even %.0f does not reach new regime tax", c.BreakEvenDeductions)
	}
	if c.Headroom.Sec80C != 0 || c.Headroom.Sec80CCD1B != 0 || c.Headroom.Sec80DSelf != 0 {
		t.Errorf("unexpected headroom %+v", c.Headroom)
	}
}

func TestOldWinsWithLargeDeductions(t *testing.T) {
	in := Input{FY: "2026-27", Age: 40, Resident: true,
		GrossSalary: 1800000, BasicDA: 900000, HRAExempt: 360000, HRAReceived: 360000,
		Ded80C: 150000, OwnNPS: 50000, Ded80DSelf: 25000, Ded80DParents: 50000, ParentsSenior: true,
		SelfOccupiedInterest: 200000}
	c := Compare(in)
	if c.Recommended != Old {
		t.Fatalf("expected old regime, old=%.0f new=%.0f", c.Old.TotalTax, c.New.TotalTax)
	}
	if c.BreakEvenDeductions != 0 {
		t.Errorf("break-even should be 0 when old wins")
	}
	found := false
	for _, l := range c.Old.Deductions {
		if l.Code == "80C" && l.NewSection == "123" {
			found = true
		}
	}
	if !found {
		t.Errorf("FY 2026-27 should map 80C to new section 123")
	}
}

func TestNewRegimeIgnoresOldDeductionsButKeepsEmployerNPS(t *testing.T) {
	in := Input{FY: "2025-26", Age: 30, Resident: true, GrossSalary: 2000000, BasicDA: 1000000,
		EmployerNPS: 200000, Ded80C: 150000, SelfOccupiedInterest: 200000}
	r := Compute(in, New)
	// 80CCD(2) capped at 14% of basic = 1.4L
	if r.TotalDeductions != 140000 {
		t.Fatalf("new regime deductions got %.0f want 140000", r.TotalDeductions)
	}
	if r.HouseProperty != 0 {
		t.Fatalf("self-occupied interest must not reduce income in new regime")
	}
}

func TestNRIHasNoRebate(t *testing.T) {
	in := other("2025-26", 1000000)
	in.Resident = false
	if got := Total(in, New); got == 0 {
		t.Fatalf("NRI should not get 87A rebate")
	}
}

func TestNormalizeFY(t *testing.T) {
	for in, want := range map[string]string{"2025-26": "2025-26", "2025-2026": "2025-26", "2026": "2026-27", "": ""} {
		if got := NormalizeFY(in); got != want {
			t.Errorf("NormalizeFY(%q)=%q want %q", in, got, want)
		}
	}
	if _, ok := RulesFor("2030-31"); ok {
		t.Errorf("unsupported FY should report ok=false")
	}
}

func TestMarginalRate(t *testing.T) {
	// At exactly ₹20L the next rupee falls in the 25% slab (26% with cess).
	c := Compare(other("2025-26", 2000000))
	if math.Abs(c.New.MarginalPc-26) > 0.01 {
		t.Errorf("new marginal at 20L got %.2f want 26", c.New.MarginalPc)
	}
	if math.Abs(c.Old.MarginalPc-31.2) > 0.01 {
		t.Errorf("old marginal at 20L got %.2f want 31.2", c.Old.MarginalPc)
	}
}
