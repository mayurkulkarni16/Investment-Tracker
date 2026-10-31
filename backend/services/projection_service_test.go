package services

import (
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"

	"investment-tracker/models"
)

func TestPercentilesMatchSort(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, n := range []int{10, 101, 2000} {
		xs := make([]float64, n)
		for i := range xs {
			xs[i] = math.Floor(rng.NormFloat64() * 100) // duplicates on purpose
		}
		orig := append([]float64(nil), xs...)
		p10, p50, p90 := percentiles(xs, make([]float64, n))

		sorted := append([]float64(nil), xs...)
		sort.Float64s(sorted)
		if p10 != sorted[n*10/100] || p50 != sorted[n/2] || p90 != sorted[n*90/100] {
			t.Errorf("n=%d got %v %v %v, want %v %v %v", n, p10, p50, p90, sorted[n*10/100], sorted[n/2], sorted[n*90/100])
		}
		for i := range xs {
			if xs[i] != orig[i] {
				t.Fatalf("percentiles modified its input")
			}
		}
	}
}

func TestRunProjectionMedianGrowsAtAssumedRate(t *testing.T) {
	now := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	specs := []holdingSpec{{name: "eq", current: 100000, rate: 12, vol: 18, beta: 0.9}}
	inv, agg := runProjection(specs, 120, 4000, 7, now)

	want := 100000 * math.Pow(1.12, 10)
	got := inv[0].Projections[120].P50
	if math.Abs(got/want-1) > 0.05 {
		t.Errorf("10y median = %.0f, want ~%.0f (12%%/yr)", got, want)
	}
	if agg[120].P50 != got {
		t.Errorf("single-holding aggregate median %.0f != holding median %.0f", agg[120].P50, got)
	}
	last := inv[0].Projections[120]
	if !(last.P10 < last.P50 && last.P50 < last.P90) {
		t.Errorf("percentiles not ordered: %+v", last)
	}
	if inv[0].Projections[0].P50 != 100000 || inv[0].Basis != models.ProjectionSimulated {
		t.Errorf("start point = %+v basis=%s", inv[0].Projections[0], inv[0].Basis)
	}
}

func TestRunProjectionPortfolioRangeReflectsDiversification(t *testing.T) {
	now := time.Now()
	specs := []holdingSpec{
		{name: "a", current: 500000, rate: 12, vol: 18, beta: 0.9},
		{name: "b", current: 500000, rate: 7, vol: 3, beta: 0.2},
		{name: "c", current: 500000, rate: 12, vol: 25, beta: 0.8},
	}
	inv, agg := runProjection(specs, 120, 2000, 3, now)
	var sumP10, sumP90 float64
	for _, h := range inv {
		sumP10 += h.Projections[120].P10
		sumP90 += h.Projections[120].P90
	}
	if !(agg[120].P10 > sumP10 && agg[120].P90 < sumP90) {
		t.Errorf("portfolio P10..P90 %.0f..%.0f should be inside sum of holdings %.0f..%.0f",
			agg[120].P10, agg[120].P90, sumP10, sumP90)
	}
}

func TestRunProjectionIsDeterministicAndAddsScheduled(t *testing.T) {
	now := time.Now()
	sched := compoundPath(100000, 8.25, 5000, 24)
	specs := []holdingSpec{
		{name: "mf", current: 200000, rate: 12, vol: 18, beta: 0.9, monthly: 10000, contribMonths: 12},
		{name: "pf", current: 100000, rate: 8.25, scheduled: sched},
	}
	_, a1 := runProjection(specs, 24, 1000, 42, now)
	inv, a2 := runProjection(specs, 24, 1000, 42, now)
	for m := range a1 {
		if a1[m] != a2[m] {
			t.Fatalf("same seed gave different results at month %d", m)
		}
	}
	pf := inv[1]
	if pf.Basis != models.ProjectionScheduled || pf.Projections[24].P10 != pf.Projections[24].P90 ||
		pf.Projections[24].P50 != math.Round(sched[24]) {
		t.Errorf("scheduled holding not flat/exact: %+v", pf.Projections[24])
	}
	// Contributions stop after 12 months: 12 x 10k added to the MF.
	if inv[0].Projections[24].P50 < 200000+120000 {
		t.Errorf("mf median %.0f should include 1.2L of contributions", inv[0].Projections[24].P50)
	}
}

func TestAmortizationPathPaysOffLoan(t *testing.T) {
	// 10L at 9% over 120 months -> EMI ~12,668
	p := amortizationPath(1000000, 9, 12668, 130)
	if p[0] != 1000000 || p[60] >= p[0] || p[120] > 50 || p[130] != 0 {
		t.Errorf("amortization path = start %.0f, 5y %.0f, 10y %.0f, end %.0f", p[0], p[60], p[120], p[130])
	}
	flat := amortizationPath(500000, 9, 0, 12)
	if flat[12] != 500000 {
		t.Errorf("loan without EMI should stay flat, got %.0f", flat[12])
	}
}

func TestFDValueAt(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fd := &models.FixedDeposit{
		PrincipalAmount: 100000, InterestRate: 8, InterestType: models.InterestCumulative,
		StartDate: start, MaturityDate: start.AddDate(2, 0, 0),
	}
	fd.MaturityAmount = math.Round(100000*math.Pow(1.02, 8)*100) / 100
	now := start.AddDate(1, 0, 0)

	if v := fdValueAt(fd, now, now); math.Abs(v-100000*math.Pow(1.02, 4)) > 50 {
		t.Errorf("accrued after 1y = %.0f, want ~108243", v)
	}
	if v := fdValueAt(fd, now, start.AddDate(3, 0, 0)); v != fd.MaturityAmount {
		t.Errorf("after maturity = %.0f, want maturity amount %.0f", v, fd.MaturityAmount)
	}

	freq := models.PayoutQuarterly
	payout := &models.FixedDeposit{
		PrincipalAmount: 100000, InterestType: models.InterestNonCumulative, PayoutFrequency: &freq,
		InterestPayouts: []models.InterestPayout{
			{ScheduledDate: now.AddDate(0, -1, 0), Amount: 2000, Status: models.PayoutReceived},
			{ScheduledDate: now.AddDate(0, 2, 0), Amount: 2000, Status: models.PayoutPending},
			{ScheduledDate: now.AddDate(0, 5, 0), Amount: 2000, Status: models.PayoutPending},
		},
	}
	if v := fdValueAt(payout, now, now.AddDate(0, 3, 0)); v != 102000 {
		t.Errorf("payout FD after 3 months = %.0f, want 102000 (principal + 1 future payout)", v)
	}
}

func TestBlendedReturnShrinksTowardsPrior(t *testing.T) {
	prior := MutualFundProfile(models.FundTypeSmallCap) // 14%
	if got := BlendedReturn(44, 0.5, prior); got < 14 || got > 16 {
		t.Errorf("6-month 44%% streak -> %.1f%%, want close to the 14%% prior", got)
	}
	if got := BlendedReturn(20, 10, prior); got != 17 {
		t.Errorf("10-year 20%% -> %.1f%%, want 17%% (50/50 cap)", got)
	}
	if got := BlendedReturn(-30, 2, prior); got >= 14 || got < 5 {
		t.Errorf("2-year loss -> %.1f%%, want below prior but not extreme", got)
	}
	if MutualFundProfile(models.FundTypeGilt).EquityShare != 0 || MutualFundProfile(models.FundTypeOvernight).Volatility > 1 {
		t.Error("debt sub-types must not be classified as equity")
	}
}

func TestMonthlyEquivalent(t *testing.T) {
	if v := monthlyEquivalent(1000, "weekly"); math.Abs(v-4333.33) > 0.01 {
		t.Errorf("weekly 1000 -> %.2f", v)
	}
	if v := monthlyEquivalent(3000, "quarterly"); v != 1000 {
		t.Errorf("quarterly 3000 -> %.2f", v)
	}
	if v := monthlyEquivalent(5000, "monthly"); v != 5000 {
		t.Errorf("monthly 5000 -> %.2f", v)
	}
}
