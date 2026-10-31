package services

import (
	"math"
	"testing"
	"time"

	"investment-tracker/models"
)

func TestNormalizeFixedDepositMaturesAndCountsCash(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	fd := &models.FixedDeposit{
		PrincipalAmount: 100000, InterestRate: 8, InterestType: models.InterestCumulative,
		StartDate: start, MaturityDate: start.AddDate(1, 0, 0), Status: models.FDActive,
	}
	fd.MaturityAmount = math.Round(100000*math.Pow(1.02, 4)*100) / 100

	// Before maturity: still active, valued at accrued (not maturity) amount.
	mid := start.AddDate(0, 6, 0)
	NormalizeFixedDeposit(fd, mid)
	if fd.Status != models.FDActive {
		t.Fatalf("status before maturity = %s", fd.Status)
	}
	v := FixedDepositCurrentValue(fd, mid)
	if v <= 100000 || v >= fd.MaturityAmount {
		t.Errorf("accrued value at 6 months = %.0f, want between principal and %.0f", v, fd.MaturityAmount)
	}
	if FixedDepositCashReturned(fd) != 0 {
		t.Error("active cumulative FD should not have returned cash")
	}

	// After maturity: matured, zero current value, maturity proceeds returned.
	after := start.AddDate(1, 1, 0)
	NormalizeFixedDeposit(fd, after)
	if fd.Status != models.FDMatured {
		t.Fatalf("status after maturity = %s", fd.Status)
	}
	if FixedDepositCurrentValue(fd, after) != 0 || FixedDepositCashReturned(fd) != fd.MaturityAmount {
		t.Errorf("matured FD value=%.0f returned=%.0f", FixedDepositCurrentValue(fd, after), FixedDepositCashReturned(fd))
	}

	// Auto-renewed FDs stay active.
	renewed := *fd
	renewed.Status, renewed.IsAutoRenewed = models.FDActive, true
	NormalizeFixedDeposit(&renewed, after)
	if renewed.Status != models.FDActive {
		t.Error("auto-renewed FD should stay active")
	}
}

func TestPayoutFDXIRRIncludesInterest(t *testing.T) {
	now := time.Now()
	start := now.AddDate(-1, 0, 0)
	freq := models.PayoutQuarterly
	fd := &models.FixedDeposit{
		PrincipalAmount: 100000, InterestRate: 8, InterestType: models.InterestNonCumulative,
		PayoutFrequency: &freq, StartDate: start, MaturityDate: start.AddDate(3, 0, 0), Status: models.FDActive,
	}
	for q := 1; q <= 12; q++ {
		fd.InterestPayouts = append(fd.InterestPayouts, models.InterestPayout{
			ScheduledDate: start.AddDate(0, 3*q, 0), Amount: 2000, Status: models.PayoutPending,
		})
	}
	NormalizeFixedDeposit(fd, now)
	if x := ComputeFixedDepositXIRR(fd); x < 7 || x > 9 {
		t.Errorf("payout FD XIRR = %.2f%%, want ~8%% (interest payouts must count)", x)
	}
	// Taxable interest for the past year = the 4 payouts received.
	if got := FixedDepositInterestBetween(fd, start, now.AddDate(0, 0, 1)); got != 8000 {
		t.Errorf("interest in year = %.0f, want 8000", got)
	}
}

func TestCumulativeFDInterestBetweenSplitsAcrossYears(t *testing.T) {
	start := time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC)
	fd := &models.FixedDeposit{
		PrincipalAmount: 100000, InterestRate: 8, InterestType: models.InterestCumulative,
		StartDate: start, MaturityDate: start.AddDate(2, 0, 0),
	}
	fd.MaturityAmount = 100000 * math.Pow(1.02, 8)
	fy1 := FixedDepositInterestBetween(fd, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC))
	fy2 := FixedDepositInterestBetween(fd, time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	fy3 := FixedDepositInterestBetween(fd, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC))
	if fy1 <= 0 || fy2 <= fy1 || fy3 <= 0 {
		t.Errorf("FY interest = %.0f / %.0f / %.0f", fy1, fy2, fy3)
	}
	if total := fy1 + fy2 + fy3; math.Abs(total-(fd.MaturityAmount-100000)) > 1 {
		t.Errorf("FY interest sums to %.2f, want total interest %.2f", total, fd.MaturityAmount-100000)
	}
}

func TestSummarizeBenchmark(t *testing.T) {
	// Daily closes for 5 years + a bit, growing 10%/yr.
	end := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	var pts []BenchmarkPoint
	for d := end.AddDate(-5, 0, -10); !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		years := end.Sub(d).Hours() / 8766
		pts = append(pts, BenchmarkPoint{Date: d.Format("2006-01-02"), Value: 20000 / math.Pow(1.10, years)})
	}
	b := summarizeBenchmark(pts, 12, end)
	if math.Abs(b.Change1Y-10) > 0.5 || math.Abs(b.CAGR3Y-10) > 0.3 || math.Abs(b.CAGR5Y-10) > 0.3 {
		t.Errorf("1Y=%.2f 3Y CAGR=%.2f 5Y CAGR=%.2f, want ~10%%", b.Change1Y, b.CAGR3Y, b.CAGR5Y)
	}
	if math.Abs(b.Change5Y-61.05) > 1 {
		t.Errorf("5Y absolute = %.2f, want ~61%%", b.Change5Y)
	}
	if len(b.Available) != 3 || b.AsOf != "2026-10-16" {
		t.Errorf("available=%v asOf=%s", b.Available, b.AsOf)
	}
	if first := b.Points[0].Date; first < "2025-10-16" || first > "2025-10-20" {
		t.Errorf("1y chart starts %s", first)
	}

	// Only ~2 years of history: 3Y/5Y must be marked unavailable, not 0%.
	short := summarizeBenchmark(pts[len(pts)-520:], 60, end)
	if len(short.Available) != 1 || short.Available[0] != "1y" || short.Change3Y != 0 {
		t.Errorf("short history available=%v 3y=%.2f", short.Available, short.Change3Y)
	}
}
