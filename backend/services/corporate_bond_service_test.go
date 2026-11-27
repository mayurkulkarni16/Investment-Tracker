package services

import (
	"testing"
	"time"

	"investment-tracker/models"
)

func TestGenerateInterestScheduleUsesFaceValueAndFixedPayoutDay(t *testing.T) {
	purchaseDate := time.Date(2025, time.November, 6, 0, 0, 0, 0, time.UTC)
	maturityDate := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)
	service := &CorporateBondService{}
	bond := &models.CorporateBond{
		PurchaseDate:       purchaseDate,
		InvestmentAmount:   99157.40,
		FaceValue:          100000,
		CouponRate:         10.30,
		InterestPayoutFreq: models.PayoutMonthly,
		PayoutDay:          10,
		MaturityDate:       maturityDate,
		PrincipalRepayments: service.buildPrincipalSchedule(models.CreateCorporateBondRequest{
			MaturityType:     models.MaturityBullet,
			InvestmentAmount: 99157.40,
			FaceValue:        100000,
		}, maturityDate),
	}
	if len(bond.PrincipalRepayments) != 1 {
		t.Fatalf("got %d principal repayments, want 1", len(bond.PrincipalRepayments))
	}
	if got := bond.PrincipalRepayments[0].Amount; got != 100000 {
		t.Fatalf("maturity repayment = %.2f, want face value 100000", got)
	}

	payouts := service.generateInterestSchedule(bond)
	wantDates := []time.Time{
		time.Date(2025, time.December, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.February, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC),
	}
	if len(payouts) != len(wantDates) {
		t.Fatalf("got %d payouts, want %d", len(payouts), len(wantDates))
	}
	for i, wantDate := range wantDates {
		if !payouts[i].ScheduledDate.Equal(wantDate) {
			t.Errorf("payout %d date = %s, want %s", i, payouts[i].ScheduledDate, wantDate)
		}
		if payouts[i].PrincipalAtTime != 100000 {
			t.Errorf("payout %d principal = %.2f, want face value 100000", i, payouts[i].PrincipalAtTime)
		}
	}
	if payouts[0].Amount != 846.58 {
		t.Errorf("first interest = %.2f, want 846.58 for 30 days at 10.30%% on face value", payouts[0].Amount)
	}
}

func TestBondDateAtMonthOffsetClampsWithoutDateDrift(t *testing.T) {
	anchor := time.Date(2025, time.January, 31, 0, 0, 0, 0, time.UTC)
	wantDates := []time.Time{
		time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.March, 31, 0, 0, 0, 0, time.UTC),
	}
	for i, wantDate := range wantDates {
		got := bondDateAtMonthOffset(anchor, i+1, 31)
		if !got.Equal(wantDate) {
			t.Errorf("month offset %d date = %s, want %s", i+1, got, wantDate)
		}
	}
}

func TestPreserveReceivedBondPayoutsWhenRebuildingSchedule(t *testing.T) {
	receivedDate := time.Date(2026, time.January, 8, 0, 0, 0, 0, time.UTC)
	previous := []models.InterestPayout{{Status: models.PayoutReceived, ReceivedDate: &receivedDate}}
	current := []models.InterestPayout{{Status: models.PayoutPending}}

	preserveReceivedInterestPayouts(previous, current)

	if current[0].Status != models.PayoutReceived {
		t.Fatalf("status = %s, want received", current[0].Status)
	}
	if current[0].ReceivedDate == nil || !current[0].ReceivedDate.Equal(receivedDate) {
		t.Fatalf("received date = %v, want %s", current[0].ReceivedDate, receivedDate)
	}
}
