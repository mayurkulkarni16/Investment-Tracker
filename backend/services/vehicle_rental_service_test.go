package services

import (
	"math"
	"testing"
	"time"

	"investment-tracker/models"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func ridojiRequest() models.CreateVehicleRentalRequest {
	return models.CreateVehicleRentalRequest{
		Provider:             "Ridoji",
		VehicleModel:         "EV Scooty",
		Units:                2,
		UnitPrice:            63000,
		GSTPercent:           5,
		PurchaseDate:         "2026-01-10",
		PayoutStartDate:      "2026-02-10",
		TenureMonths:         48,
		MonthlyPayoutPerUnit: 3500,
		BuybackPercent:       10,
		RegistrationNumbers:  []string{" dl3s ab 1234 ", ""},
	}
}

func newRental(t *testing.T, req models.CreateVehicleRentalRequest) *models.VehicleRental {
	t.Helper()
	v := &models.VehicleRental{BuybackStatus: models.PayoutPending}
	if err := applyVehicleRentalRequest(v, req); err != nil {
		t.Fatalf("apply: %v", err)
	}
	v.Payouts = buildRentalSchedule(v, nil)
	return v
}

func TestApplyVehicleRentalRequestComputesAmounts(t *testing.T) {
	v := newRental(t, ridojiRequest())

	if v.InvestmentAmount != 132300 {
		t.Errorf("investment = %v, want 132300 (2 x 63000 + 5%% GST)", v.InvestmentAmount)
	}
	if v.BuybackAmount != 12600 {
		t.Errorf("buyback = %v, want 12600 (10%% of 126000)", v.BuybackAmount)
	}
	if len(v.RegistrationNumbers) != 1 || v.RegistrationNumbers[0] != "DL3S AB 1234" {
		t.Errorf("registration numbers = %q", v.RegistrationNumbers)
	}
	if len(v.Payouts) != 48 || v.Payouts[0].Amount != 7000 {
		t.Fatalf("schedule = %d payouts of %v, want 48 of 7000", len(v.Payouts), v.Payouts[0].Amount)
	}
	if !v.Payouts[47].ScheduledDate.Equal(date(2030, time.January, 10)) {
		t.Errorf("last payout = %v, want 2030-01-10", v.Payouts[47].ScheduledDate)
	}
}

func TestApplyVehicleRentalRequestRejectsPayoutBeforePurchase(t *testing.T) {
	req := ridojiRequest()
	req.PayoutStartDate = "2025-12-01"
	if err := applyVehicleRentalRequest(&models.VehicleRental{}, req); err == nil {
		t.Fatal("expected error when payouts start before purchase")
	}
}

func TestAddMonthsClampedHandlesMonthEnd(t *testing.T) {
	start := date(2026, time.January, 31)
	cases := map[int]time.Time{
		0:  date(2026, time.January, 31),
		1:  date(2026, time.February, 28),
		2:  date(2026, time.March, 31),
		3:  date(2026, time.April, 30),
		25: date(2028, time.February, 29),
	}
	for n, want := range cases {
		if got := addMonthsClamped(start, n); !got.Equal(want) {
			t.Errorf("addMonthsClamped(+%d) = %v, want %v", n, got.Format("2006-01-02"), want.Format("2006-01-02"))
		}
	}
}

func TestComputeVehicleRentalOverdueAndNext(t *testing.T) {
	v := newRental(t, ridojiRequest())
	// Payouts on 10 Feb, 10 Mar, 10 Apr. Receive Feb only; evaluate on 12 Apr.
	recv := date(2026, time.February, 11)
	v.Payouts[0].Status = models.PayoutReceived
	v.Payouts[0].ReceivedDate = &recv
	v.Payouts[0].ReceivedAmount = 7000

	ComputeVehicleRental(v, date(2026, time.April, 12))

	if v.PayoutsReceivedCount != 1 || v.TotalPayoutsReceived != 7000 {
		t.Errorf("received = %d / %v", v.PayoutsReceivedCount, v.TotalPayoutsReceived)
	}
	// March is >5 days late -> overdue; April (2 days late) is still within grace.
	if v.OverdueCount != 1 || v.OverdueAmount != 7000 {
		t.Errorf("overdue = %d / %v, want 1 / 7000", v.OverdueCount, v.OverdueAmount)
	}
	if !v.Payouts[1].IsOverdue || v.Payouts[2].IsOverdue {
		t.Errorf("overdue flags = %v %v, want true false", v.Payouts[1].IsOverdue, v.Payouts[2].IsOverdue)
	}
	if v.NextPayoutDate == nil || !v.NextPayoutDate.Equal(date(2026, time.May, 10)) {
		t.Errorf("next payout = %v, want 2026-05-10", v.NextPayoutDate)
	}
	if v.Status != models.VehicleRentalActive {
		t.Errorf("status = %v", v.Status)
	}
}

func TestVehicleRentalBookValueIsStraightLine(t *testing.T) {
	v := newRental(t, ridojiRequest())
	ComputeVehicleRental(v, v.PurchaseDate)
	if v.CurrentValue != v.InvestmentAmount {
		t.Errorf("value at purchase = %v, want %v", v.CurrentValue, v.InvestmentAmount)
	}

	mid := v.PurchaseDate.Add(v.AgreementEndDate.Sub(v.PurchaseDate) / 2)
	ComputeVehicleRental(v, mid)
	want := (v.InvestmentAmount + v.BuybackAmount) / 2
	if math.Abs(v.CurrentValue-want) > 1 {
		t.Errorf("value at midpoint = %v, want ~%v", v.CurrentValue, want)
	}

	ComputeVehicleRental(v, v.AgreementEndDate.AddDate(0, 1, 0))
	if v.CurrentValue != v.BuybackAmount {
		t.Errorf("value after end = %v, want buyback %v", v.CurrentValue, v.BuybackAmount)
	}
}

func TestComputeVehicleRentalCompletedAndXIRR(t *testing.T) {
	v := newRental(t, ridojiRequest())
	for i := range v.Payouts {
		d := v.Payouts[i].ScheduledDate
		v.Payouts[i].Status = models.PayoutReceived
		v.Payouts[i].ReceivedDate = &d
		v.Payouts[i].ReceivedAmount = v.Payouts[i].Amount
	}
	bb := v.Payouts[47].ScheduledDate
	v.BuybackStatus = models.PayoutReceived
	v.BuybackReceivedDate = &bb
	v.BuybackReceived = v.BuybackAmount

	ComputeVehicleRental(v, date(2030, time.June, 1))

	if v.Status != models.VehicleRentalCompleted {
		t.Errorf("status = %v, want completed", v.Status)
	}
	if v.CurrentValue != 0 {
		t.Errorf("value after buyback = %v, want 0", v.CurrentValue)
	}
	// 48 x 7000 + 12600 - 132300 = 216300
	if v.TotalReturn != 216300 {
		t.Errorf("total return = %v, want 216300", v.TotalReturn)
	}
	// Advertised 2-unit plan: ~4.8%/month, i.e. roughly 75-80% a year.
	if v.XIRR < 70 || v.XIRR > 85 {
		t.Errorf("xirr = %v%%, want ~75-80%%", v.XIRR)
	}
}

func TestBuildRentalSchedulePreservesReceiptsOnEdit(t *testing.T) {
	v := newRental(t, ridojiRequest())
	recv := date(2026, time.February, 12)
	v.Payouts[0].Status = models.PayoutReceived
	v.Payouts[0].ReceivedDate = &recv
	v.Payouts[0].ReceivedAmount = 6500
	firstID := v.Payouts[0].PayoutID

	req := ridojiRequest()
	req.Units = 5
	req.TenureMonths = 24
	if err := applyVehicleRentalRequest(v, req); err != nil {
		t.Fatal(err)
	}
	v.Payouts = buildRentalSchedule(v, v.Payouts)

	if len(v.Payouts) != 24 {
		t.Fatalf("payouts = %d, want 24", len(v.Payouts))
	}
	p := v.Payouts[0]
	if p.PayoutID != firstID || p.Status != models.PayoutReceived || p.ReceivedAmount != 6500 {
		t.Errorf("first payout not preserved: %+v", p)
	}
	if v.Payouts[1].Amount != 17500 || v.Payouts[1].Status != models.PayoutPending {
		t.Errorf("second payout = %+v, want pending 17500", v.Payouts[1])
	}
}
