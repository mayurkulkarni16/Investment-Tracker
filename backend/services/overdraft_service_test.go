package services

import (
	"math"
	"strings"
	"testing"
	"time"

	"investment-tracker/models"
)

func newOverdraft(txns ...models.OverdraftTransaction) *models.Overdraft {
	return &models.Overdraft{
		LenderName: "HDFC", SanctionedLimit: 100000, InterestRate: 12,
		OpenDate: date(2026, time.January, 1), Status: models.LoanActive, Transactions: txns,
	}
}

func odTxn(day time.Time, typ models.OverdraftTxnType, amount float64) models.OverdraftTransaction {
	return models.OverdraftTransaction{TxnID: day.Format("0102") + string(typ), Date: day, Type: typ, Amount: amount}
}

func TestComputeOverdraftTracksPrincipalAndInterest(t *testing.T) {
	od := newOverdraft(
		odTxn(date(2026, time.February, 1), models.OverdraftRepayment, 20000),
		odTxn(date(2026, time.January, 1), models.OverdraftWithdrawal, 50000),
		odTxn(date(2026, time.January, 31), models.OverdraftInterest, 500),
	)
	ComputeOverdraft(od, date(2026, time.February, 10))

	if od.Outstanding != 30000 || od.AvailableLimit != 70000 || od.UtilizationPct != 30 {
		t.Errorf("outstanding/available/utilization = %v/%v/%v, want 30000/70000/30", od.Outstanding, od.AvailableLimit, od.UtilizationPct)
	}
	if od.TotalInterestPaid != 500 || od.TotalWithdrawn != 50000 || od.TotalRepaid != 20000 {
		t.Errorf("totals = %v/%v/%v", od.TotalWithdrawn, od.TotalRepaid, od.TotalInterestPaid)
	}
	if od.Transactions[1].Type != models.OverdraftInterest || od.Transactions[1].BalanceAfter != 50000 {
		t.Errorf("interest payment should not change principal: %+v", od.Transactions[1])
	}
	if od.MonthlyInterest != 300 {
		t.Errorf("monthly interest = %v, want 300", od.MonthlyInterest)
	}
	// 1-10 Feb (10 days) on 30000 at 12%: interest up to 31 Jan was paid.
	want := math.Round(30000*0.12/365*10*100) / 100
	if od.AccruedInterest != want {
		t.Errorf("accrued interest = %v, want %v", od.AccruedInterest, want)
	}
}

func TestOverdraftInterestFollowsDailyBalance(t *testing.T) {
	od := newOverdraft(
		odTxn(date(2026, time.January, 1), models.OverdraftWithdrawal, 36500),
		odTxn(date(2026, time.January, 11), models.OverdraftRepayment, 36500),
	)
	// 10 days on 36500 at 12% = 120, then nothing owed.
	got := OverdraftInterestBetween(od, date(2026, time.January, 1), date(2026, time.March, 1))
	if math.Abs(got-120) > 1e-6 {
		t.Errorf("interest = %v, want 120", got)
	}
}

func TestValidateOverdraft(t *testing.T) {
	jan := func(d int) time.Time { return date(2026, time.January, d) }
	cases := []struct {
		name       string
		od         *models.Overdraft
		checkLimit bool
		wantErr    string
	}{
		{"repayment above balance", newOverdraft(odTxn(jan(2), models.OverdraftWithdrawal, 1000), odTxn(jan(3), models.OverdraftRepayment, 1500)), false, "more than the outstanding"},
		{"over the limit", newOverdraft(odTxn(jan(2), models.OverdraftWithdrawal, 60000), odTxn(jan(3), models.OverdraftWithdrawal, 50000)), true, "sanctioned limit"},
		{"before open date", newOverdraft(odTxn(date(2025, time.December, 31), models.OverdraftWithdrawal, 1000)), false, "before the account was opened"},
		{"limit not enforced on edits", newOverdraft(odTxn(jan(2), models.OverdraftWithdrawal, 150000)), false, ""},
	}
	for _, tc := range cases {
		err := validateOverdraft(tc.od, tc.checkLimit)
		if tc.wantErr == "" && err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
		}
		if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("%s: error = %v, want %q", tc.name, err, tc.wantErr)
		}
	}

	closed := newOverdraft(odTxn(jan(2), models.OverdraftWithdrawal, 1000))
	closed.Status = models.LoanClosed
	if err := validateOverdraft(closed, false); err == nil {
		t.Error("closing an overdraft with a balance should fail")
	}
}

func TestAlternateGoalHoldingSpreadsPendingPayouts(t *testing.T) {
	v := newRental(t, ridojiRequest())
	h := alternateGoalHolding(v, testNow, date(2027, time.January, 10))

	// Payouts of 7000 fall on 5 Feb 2026 .. 5 Jan 2027: 12 in 12 months.
	if h.monthly != 7000 {
		t.Errorf("monthly = %v, want 7000", h.monthly)
	}
	if !h.unweighted || h.value != v.CurrentValue || h.value <= 0 {
		t.Errorf("holding = %+v, want unweighted at current value %v", h, v.CurrentValue)
	}
}
