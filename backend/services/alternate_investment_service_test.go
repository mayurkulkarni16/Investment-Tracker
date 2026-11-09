package services

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"investment-tracker/models"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// testNow is the "today" used when building schedules in tests.
var testNow = date(2026, time.January, 10)

func ridojiRequest() models.CreateAlternateInvestmentRequest {
	return models.CreateAlternateInvestmentRequest{
		Provider:             "Ridoji",
		AssetName:         "EV Scooty",
		Units:                2,
		UnitPrice:            63000,
		GSTPercent:           5,
		PurchaseDate:         "2026-01-10",
		PayoutStartDate:      "2026-02-10",
		TenureMonths:         48,
		MonthlyPayoutPerUnit: 3500,
		BuybackPercent:       10,
		Identifiers:  []string{" dl3s ab 1234 ", ""},
	}
}

func newRental(t *testing.T, req models.CreateAlternateInvestmentRequest) *models.AlternateInvestment {
	t.Helper()
	v := &models.AlternateInvestment{BuybackStatus: models.PayoutPending}
	if err := applyAlternateInvestmentRequest(v, req); err != nil {
		t.Fatalf("apply: %v", err)
	}
	v.Payouts = buildAltPayoutSchedule(v, nil, testNow)
	return v
}

func TestApplyAlternateInvestmentRequestComputesAmounts(t *testing.T) {
	v := newRental(t, ridojiRequest())

	if v.InvestmentAmount != 132300 {
		t.Errorf("investment = %v, want 132300 (2 x 63000 + 5%% GST)", v.InvestmentAmount)
	}
	if v.BuybackAmount != 12600 {
		t.Errorf("buyback = %v, want 12600 (10%% of 126000)", v.BuybackAmount)
	}
	if len(v.Identifiers) != 1 || v.Identifiers[0] != "DL3S AB 1234" {
		t.Errorf("identifiers = %q", v.Identifiers)
	}
	if len(v.Payouts) != 48 || v.Payouts[0].Amount != 7000 {
		t.Fatalf("schedule = %d payouts of %v, want 48 of 7000", len(v.Payouts), v.Payouts[0].Amount)
	}
	if !v.Payouts[47].ScheduledDate.Equal(date(2030, time.January, 10)) {
		t.Errorf("last payout = %v, want 2030-01-10", v.Payouts[47].ScheduledDate)
	}
}

func TestApplyAlternateInvestmentRequestRejectsPayoutBeforePurchase(t *testing.T) {
	req := ridojiRequest()
	req.PayoutStartDate = "2025-12-01"
	if err := applyAlternateInvestmentRequest(&models.AlternateInvestment{}, req); err == nil {
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

func TestComputeAlternateInvestmentOverdueAndNext(t *testing.T) {
	v := newRental(t, ridojiRequest())
	// Payouts on 10 Feb, 10 Mar, 10 Apr. Receive Feb only; evaluate on 12 Apr.
	recv := date(2026, time.February, 11)
	v.Payouts[0].Status = models.PayoutReceived
	v.Payouts[0].ReceivedDate = &recv
	v.Payouts[0].ReceivedAmount = 7000

	ComputeAlternateInvestment(v, date(2026, time.April, 12))

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
	if v.Status != models.AltInvestmentActive {
		t.Errorf("status = %v", v.Status)
	}
}

func TestAlternateInvestmentBookValueIsStraightLine(t *testing.T) {
	v := newRental(t, ridojiRequest())
	ComputeAlternateInvestment(v, v.PurchaseDate)
	if v.CurrentValue != v.InvestmentAmount {
		t.Errorf("value at purchase = %v, want %v", v.CurrentValue, v.InvestmentAmount)
	}

	mid := v.PurchaseDate.Add(v.AgreementEndDate.Sub(v.PurchaseDate) / 2)
	ComputeAlternateInvestment(v, mid)
	want := (v.InvestmentAmount + v.BuybackAmount) / 2
	if math.Abs(v.CurrentValue-want) > 1 {
		t.Errorf("value at midpoint = %v, want ~%v", v.CurrentValue, want)
	}

	ComputeAlternateInvestment(v, v.AgreementEndDate.AddDate(0, 1, 0))
	if v.CurrentValue != v.BuybackAmount {
		t.Errorf("value after end = %v, want buyback %v", v.CurrentValue, v.BuybackAmount)
	}
}

func TestComputeAlternateInvestmentCompletedAndXIRR(t *testing.T) {
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

	ComputeAlternateInvestment(v, date(2030, time.June, 1))

	if v.Status != models.AltInvestmentCompleted {
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

func TestBuildAltPayoutSchedulePreservesReceiptsOnEdit(t *testing.T) {
	v := newRental(t, ridojiRequest())
	recv := date(2026, time.February, 12)
	v.Payouts[0].Status = models.PayoutReceived
	v.Payouts[0].ReceivedDate = &recv
	v.Payouts[0].ReceivedAmount = 6500
	firstID := v.Payouts[0].PayoutID

	req := ridojiRequest()
	req.Units = 5
	req.TenureMonths = 24
	if err := applyAlternateInvestmentRequest(v, req); err != nil {
		t.Fatal(err)
	}
	v.Payouts = buildAltPayoutSchedule(v, v.Payouts, testNow)

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

func girCowRequest() models.CreateAlternateInvestmentRequest {
	return models.CreateAlternateInvestmentRequest{
		AssetType:            models.AltAssetLivestock,
		Provider:             "Amrutha Farms",
		AssetName:            "Gir",
		Units:                1,
		UnitPrice:            200000,
		PurchaseDate:         "2026-11-09",
		PayoutStartDate:      "2026-12-09",
		TenureMonths:         0,
		MonthlyPayoutPerUnit: 20000,
		ValuationMethod:      models.AltValuationAtCost,
		DailyYieldLitres:     15,
		PricePerLitre:        100,
		Identifiers:          []string{" tag-101 "},
	}
}

func newOpenEnded(t *testing.T, req models.CreateAlternateInvestmentRequest, now time.Time) *models.AlternateInvestment {
	t.Helper()
	v := &models.AlternateInvestment{BuybackStatus: models.PayoutPending}
	if err := applyAlternateInvestmentRequest(v, req); err != nil {
		t.Fatalf("apply: %v", err)
	}
	v.Payouts = buildAltPayoutSchedule(v, nil, now)
	return v
}

func TestOpenEndedLivestockScheduleAndMetrics(t *testing.T) {
	now := date(2026, time.November, 9)
	v := newOpenEnded(t, girCowRequest(), now)

	// 9 Dec 2026 .. 9 Nov 2027: everything up to 12 months past today.
	if len(v.Payouts) != 12 {
		t.Fatalf("payouts = %d, want 12", len(v.Payouts))
	}
	if v.Payouts[0].PayoutID != "m0001" || v.Payouts[11].PayoutID != "m0012" {
		t.Errorf("open-ended payout IDs must be deterministic, got %q .. %q", v.Payouts[0].PayoutID, v.Payouts[11].PayoutID)
	}

	ComputeAlternateInvestment(v, now)
	if !v.OpenEnded || v.Status != models.AltInvestmentActive {
		t.Errorf("open_ended = %v, status = %v", v.OpenEnded, v.Status)
	}
	if v.CurrentValue != 200000 {
		t.Errorf("at-cost value = %v, want 200000", v.CurrentValue)
	}
	if v.BreakEvenMonths != 10 {
		t.Errorf("break-even = %v months, want 10", v.BreakEvenMonths)
	}
	if v.GrossMonthlyRevenue != 45000 {
		t.Errorf("gross milk revenue = %v, want 45000 (15 L x 100 x 30)", v.GrossMonthlyRevenue)
	}
	if len(v.Identifiers) != 1 || v.Identifiers[0] != "TAG-101" {
		t.Errorf("identifiers = %q", v.Identifiers)
	}
}

func TestOpenEndedScheduleExtendsAndKeepsReceipts(t *testing.T) {
	v := newOpenEnded(t, girCowRequest(), date(2026, time.November, 9))
	recv := date(2026, time.December, 10)
	v.Payouts[0].Status = models.PayoutReceived
	v.Payouts[0].ReceivedDate = &recv
	v.Payouts[0].ReceivedAmount = 20000

	ComputeAlternateInvestment(v, date(2027, time.June, 9))

	// Now runs to 9 Jun 2028: Dec 2026 .. Jun 2028 = 19 payouts.
	if len(v.Payouts) != 19 {
		t.Fatalf("payouts = %d, want 19", len(v.Payouts))
	}
	if v.Payouts[0].Status != models.PayoutReceived || v.Payouts[0].PayoutID != "m0001" {
		t.Errorf("first payout not preserved: %+v", v.Payouts[0])
	}
	if v.Payouts[18].PayoutID != "m0019" {
		t.Errorf("extended payout ID = %q, want m0019", v.Payouts[18].PayoutID)
	}
	if v.Status != models.AltInvestmentActive {
		t.Errorf("open-ended investment must never complete, got %v", v.Status)
	}
}

func TestPendingAltPayoutsBetweenContinuesOpenEnded(t *testing.T) {
	now := date(2026, time.November, 9)
	v := newOpenEnded(t, girCowRequest(), now)
	ComputeAlternateInvestment(v, now)

	// (9 Nov 2026, 9 Nov 2028] holds 24 monthly payouts, 12 beyond the generated schedule.
	if got := pendingAltPayoutsBetween(v, now, now.AddDate(2, 0, 0)); got != 24*20000 {
		t.Errorf("pending payouts over 2 years = %v, want %v", got, 24*20000)
	}
	if r := promisedAltReturn(v, now); r < 200 || r > 300 {
		t.Errorf("promised return = %v%%, want ~240%% (20k/month on 2L held at cost)", r)
	}
}

func TestApplyAlternateInvestmentRequestOpenEndedValidation(t *testing.T) {
	req := girCowRequest()
	req.ValuationMethod = models.AltValuationStraightLine
	if err := applyAlternateInvestmentRequest(&models.AlternateInvestment{}, req); err == nil {
		t.Error("expected error for straight-line valuation without a tenure")
	}

	req = girCowRequest()
	req.BuybackPercent = 10
	if err := applyAlternateInvestmentRequest(&models.AlternateInvestment{}, req); err == nil {
		t.Error("expected error for a buyback without a tenure")
	}

	v := &models.AlternateInvestment{}
	if err := applyAlternateInvestmentRequest(v, ridojiRequest()); err != nil {
		t.Fatal(err)
	}
	if v.AssetType != models.AltAssetVehicle || v.ValuationMethod != models.AltValuationStraightLine {
		t.Errorf("defaults = %q / %q, want vehicle / straight_line", v.AssetType, v.ValuationMethod)
	}
	if v.DailyYieldLitres != 0 {
		t.Errorf("milk yield must only be kept for livestock")
	}
}

func TestManualValuationUsesEnteredValue(t *testing.T) {
	req := girCowRequest()
	req.ValuationMethod = models.AltValuationManual
	req.ManualValue = 150000
	now := date(2027, time.March, 1)
	v := newOpenEnded(t, req, now)
	ComputeAlternateInvestment(v, now)
	if v.CurrentValue != 150000 {
		t.Errorf("manual value = %v, want 150000", v.CurrentValue)
	}
}

func TestComputeDefaultsLegacyRecord(t *testing.T) {
	v := newRental(t, ridojiRequest())
	v.AssetType, v.ValuationMethod = "", ""
	ComputeAlternateInvestment(v, date(2026, time.April, 12))
	if v.AssetType != models.AltAssetVehicle || v.ValuationMethod != models.AltValuationStraightLine || v.OpenEnded {
		t.Errorf("legacy defaults = %q / %q / open-ended %v", v.AssetType, v.ValuationMethod, v.OpenEnded)
	}
}

func TestAlternateInvestmentUnmarshalAcceptsLegacyKeys(t *testing.T) {
	var v models.AlternateInvestment
	raw := `{"provider":"Ridoji","vehicle_model":"EV Scooty","registration_numbers":["DL3S AB 1234"],"units":2}`
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatal(err)
	}
	if v.Provider != "Ridoji" || v.Units != 2 || v.AssetName != "EV Scooty" || len(v.Identifiers) != 1 {
		t.Errorf("legacy backup decoded as %+v", v)
	}

	var cur models.AlternateInvestment
	if err := json.Unmarshal([]byte(`{"asset_name":"Gir","identifiers":["TAG-1"]}`), &cur); err != nil {
		t.Fatal(err)
	}
	if cur.AssetName != "Gir" || len(cur.Identifiers) != 1 {
		t.Errorf("current keys decoded as %+v", cur)
	}
}