package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type AlternateInvestmentService struct {
	repo *repository.AlternateInvestmentRepo
}

func NewAlternateInvestmentService(repo *repository.AlternateInvestmentRepo) *AlternateInvestmentService {
	return &AlternateInvestmentService{repo: repo}
}

func (s *AlternateInvestmentService) Create(ctx context.Context, userID string, req models.CreateAlternateInvestmentRequest) (*models.AlternateInvestment, error) {
	v := &models.AlternateInvestment{UserID: userID, BuybackStatus: models.PayoutPending}
	if err := applyAlternateInvestmentRequest(v, req); err != nil {
		return nil, err
	}
	now := time.Now()
	v.Payouts = buildAltPayoutSchedule(v, nil, now)

	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	ComputeAlternateInvestment(v, now)
	return v, nil
}

func (s *AlternateInvestmentService) GetAll(ctx context.Context, userID string) ([]models.AlternateInvestment, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		ComputeAlternateInvestment(&items[i], now)
	}
	return items, nil
}

func (s *AlternateInvestmentService) GetByID(ctx context.Context, id string) (*models.AlternateInvestment, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	ComputeAlternateInvestment(v, time.Now())
	return v, nil
}

// Update changes the agreement terms and regenerates the payout schedule,
// carrying over receipts already recorded for each month.
func (s *AlternateInvestmentService) Update(ctx context.Context, id string, req models.CreateAlternateInvestmentRequest) (*models.AlternateInvestment, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyAlternateInvestmentRequest(v, req); err != nil {
		return nil, err
	}
	v.Payouts = buildAltPayoutSchedule(v, v.Payouts, time.Now())
	return s.save(ctx, v)
}

func (s *AlternateInvestmentService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *AlternateInvestmentService) UpdatePayout(ctx context.Context, id, payoutID string, req models.UpdateAltPayoutRequest) (*models.AlternateInvestment, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range v.Payouts {
		p := &v.Payouts[i]
		if p.PayoutID != payoutID {
			continue
		}
		if req.Status == models.PayoutPending {
			p.Status = models.PayoutPending
			p.ReceivedDate = nil
			p.ReceivedAmount = 0
		} else {
			date, err := receivedDateOrDefault(req.ReceivedDate, p.ScheduledDate)
			if err != nil {
				return nil, err
			}
			p.Status = models.PayoutReceived
			p.ReceivedDate = &date
			p.ReceivedAmount = amountOrDefault(req.Amount, p.Amount)
		}
		return s.save(ctx, v)
	}
	return nil, notFoundf("payout %s not found", payoutID)
}

// MarkDueReceived marks every pending payout scheduled on or before today as
// received in full on its scheduled date.
func (s *AlternateInvestmentService) MarkDueReceived(ctx context.Context, id string) (*models.AlternateInvestment, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range v.Payouts {
		p := &v.Payouts[i]
		if p.Status == models.PayoutPending && !p.ScheduledDate.After(now) {
			d := p.ScheduledDate
			p.Status = models.PayoutReceived
			p.ReceivedDate = &d
			p.ReceivedAmount = p.Amount
		}
	}
	return s.save(ctx, v)
}

func (s *AlternateInvestmentService) UpdateBuyback(ctx context.Context, id string, req models.UpdateAltPayoutRequest) (*models.AlternateInvestment, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Status == models.PayoutPending {
		v.BuybackStatus = models.PayoutPending
		v.BuybackReceivedDate = nil
		v.BuybackReceived = 0
	} else {
		date, err := receivedDateOrDefault(req.ReceivedDate, time.Now())
		if err != nil {
			return nil, err
		}
		v.BuybackStatus = models.PayoutReceived
		v.BuybackReceivedDate = &date
		v.BuybackReceived = amountOrDefault(req.Amount, v.BuybackAmount)
	}
	return s.save(ctx, v)
}

// load fetches an investment and extends an open-ended schedule to today's
// horizon, so payouts generated since the last save can be updated by ID.
func (s *AlternateInvestmentService) load(ctx context.Context, id string) (*models.AlternateInvestment, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	v, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	extendOpenEndedSchedule(v, time.Now())
	return v, nil
}

func (s *AlternateInvestmentService) save(ctx context.Context, v *models.AlternateInvestment) (*models.AlternateInvestment, error) {
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	ComputeAlternateInvestment(v, time.Now())
	return v, nil
}

func applyAlternateInvestmentRequest(v *models.AlternateInvestment, req models.CreateAlternateInvestmentRequest) error {
	purchaseDate, err := parseDate(req.PurchaseDate)
	if err != nil {
		return invalidf("invalid purchase_date: %v", err)
	}
	payoutStart, err := parseDate(req.PayoutStartDate)
	if err != nil {
		return invalidf("invalid payout_start_date: %v", err)
	}
	if payoutStart.Before(purchaseDate) {
		return invalidf("payout_start_date cannot be before purchase_date")
	}

	assetType := req.AssetType
	if assetType == "" {
		assetType = models.AltAssetVehicle
	}
	valuation := req.ValuationMethod
	if valuation == "" {
		valuation = models.AltValuationStraightLine
	}
	if req.TenureMonths == 0 {
		if valuation == models.AltValuationStraightLine {
			return invalidf("straight-line valuation needs a fixed tenure; use 'at cost' or 'manual' for open-ended investments")
		}
		if req.BuybackPercent > 0 {
			return invalidf("a buyback needs a fixed tenure; set the tenure or remove the buyback")
		}
	}

	v.AssetType = assetType
	v.Provider = strings.TrimSpace(req.Provider)
	v.AssetName = strings.TrimSpace(req.AssetName)
	v.Units = req.Units
	v.UnitPrice = req.UnitPrice
	v.GSTPercent = req.GSTPercent
	v.OtherCharges = req.OtherCharges
	v.PurchaseDate = purchaseDate
	v.PayoutStartDate = payoutStart
	v.TenureMonths = req.TenureMonths
	v.MonthlyPayoutPerUnit = req.MonthlyPayoutPerUnit
	v.BuybackPercent = req.BuybackPercent
	v.ValuationMethod = valuation
	v.ManualValue = 0
	if valuation == models.AltValuationManual {
		v.ManualValue = round2(req.ManualValue)
	}
	v.DailyYieldLitres, v.PricePerLitre = 0, 0
	if assetType == models.AltAssetLivestock {
		v.DailyYieldLitres = req.DailyYieldLitres
		v.PricePerLitre = req.PricePerLitre
	}
	v.AgreementNumber = strings.TrimSpace(req.AgreementNumber)
	v.Notes = req.Notes

	v.Identifiers = []string{}
	for _, r := range req.Identifiers {
		if r = strings.ToUpper(strings.TrimSpace(r)); r != "" {
			v.Identifiers = append(v.Identifiers, r)
		}
	}

	assetCost := float64(v.Units) * v.UnitPrice
	v.InvestmentAmount = round2(assetCost*(1+v.GSTPercent/100) + v.OtherCharges)
	v.BuybackAmount = round2(assetCost * v.BuybackPercent / 100)
	if v.BuybackStatus == "" {
		v.BuybackStatus = models.PayoutPending
	}
	return nil
}

// buildAltPayoutSchedule generates one payout per month of the tenure, or up to
// AltOpenEndedLookaheadMonths past now when open-ended. Receipts in previous
// (by month index) are preserved so editing terms never loses history.
func buildAltPayoutSchedule(v *models.AlternateInvestment, previous []models.AltPayout, now time.Time) []models.AltPayout {
	n := v.TenureMonths
	if n <= 0 {
		n = openEndedPayoutCount(v, now)
	}
	amount := round2(float64(v.Units) * v.MonthlyPayoutPerUnit)
	payouts := make([]models.AltPayout, n)
	for i := range payouts {
		p := models.AltPayout{
			PayoutID:      newAltPayoutID(v, i),
			ScheduledDate: addMonthsClamped(v.PayoutStartDate, i),
			Amount:        amount,
			Status:        models.PayoutPending,
		}
		if i < len(previous) {
			p.PayoutID = previous[i].PayoutID
			if previous[i].Status == models.PayoutReceived {
				p.Status = models.PayoutReceived
				p.ReceivedDate = previous[i].ReceivedDate
				p.ReceivedAmount = previous[i].ReceivedAmount
			}
		}
		payouts[i] = p
	}
	return payouts
}

// newAltPayoutID returns a random ID for fixed schedules. Open-ended schedules
// are extended in memory on every read, so their IDs must be deterministic for
// a payout generated on read to be updatable by ID before it is ever saved.
func newAltPayoutID(v *models.AlternateInvestment, index int) string {
	if v.TenureMonths <= 0 {
		return fmt.Sprintf("m%04d", index+1)
	}
	return uuid.New().String()
}

func openEndedPayoutCount(v *models.AlternateInvestment, now time.Time) int {
	horizon := now.AddDate(0, models.AltOpenEndedLookaheadMonths, 0)
	n := 0
	for !addMonthsClamped(v.PayoutStartDate, n).After(horizon) {
		n++
	}
	if n < 1 {
		n = 1
	}
	return n
}

// extendOpenEndedSchedule appends the payouts that have come within the
// look-ahead horizon since the schedule was last generated.
func extendOpenEndedSchedule(v *models.AlternateInvestment, now time.Time) {
	if v.TenureMonths > 0 {
		return
	}
	if len(v.Payouts) < openEndedPayoutCount(v, now) {
		v.Payouts = buildAltPayoutSchedule(v, v.Payouts, now)
	}
}

// ComputeAlternateInvestment fills in the derived (non-persisted) fields: payout
// progress, overdue tracking, book value, total return and XIRR.
func ComputeAlternateInvestment(v *models.AlternateInvestment, now time.Time) {
	if v.AssetType == "" {
		v.AssetType = models.AltAssetVehicle
	}
	if v.ValuationMethod == "" {
		v.ValuationMethod = models.AltValuationStraightLine
	}
	extendOpenEndedSchedule(v, now)

	v.OpenEnded = v.TenureMonths <= 0
	v.MonthlyPayout = round2(float64(v.Units) * v.MonthlyPayoutPerUnit)
	v.AgreementEndDate = v.PayoutStartDate
	if n := len(v.Payouts); n > 0 {
		v.AgreementEndDate = v.Payouts[n-1].ScheduledDate
	}

	v.ExpectedTotalPayout, v.TotalPayoutsReceived = 0, 0
	v.PayoutsReceivedCount, v.OverdueCount, v.OverdueAmount = 0, 0, 0
	v.NextPayoutDate = nil
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	overdueBefore := today.AddDate(0, 0, -models.AltPayoutOverdueGraceDays)

	for i := range v.Payouts {
		p := &v.Payouts[i]
		v.ExpectedTotalPayout += p.Amount
		p.IsOverdue = false
		if p.Status == models.PayoutReceived {
			v.TotalPayoutsReceived += p.ReceivedAmount
			v.PayoutsReceivedCount++
			continue
		}
		if p.ScheduledDate.Before(overdueBefore) {
			p.IsOverdue = true
			v.OverdueCount++
			v.OverdueAmount += p.Amount
		}
		if v.NextPayoutDate == nil && !p.ScheduledDate.Before(today) {
			d := p.ScheduledDate
			v.NextPayoutDate = &d
		}
	}
	v.ExpectedTotalPayout = round2(v.ExpectedTotalPayout)
	v.TotalPayoutsReceived = round2(v.TotalPayoutsReceived)
	v.OverdueAmount = round2(v.OverdueAmount)

	v.CurrentValue = altInvestmentBookValue(v, now)

	recovered := v.TotalPayoutsReceived + v.BuybackReceived
	v.TotalReturn = round2(recovered + v.CurrentValue - v.InvestmentAmount)
	v.CapitalRecoveredPct, v.BreakEvenMonths = 0, 0
	if v.InvestmentAmount > 0 {
		v.CapitalRecoveredPct = round2(recovered / v.InvestmentAmount * 100)
	}
	if v.MonthlyPayout > 0 {
		v.BreakEvenMonths = round2(v.InvestmentAmount / v.MonthlyPayout)
	}
	v.GrossMonthlyRevenue = round2(float64(v.Units) * v.DailyYieldLitres * v.PricePerLitre * 30)

	allPaid := !v.OpenEnded && v.PayoutsReceivedCount == len(v.Payouts)
	buybackDone := v.BuybackAmount <= 0 || v.BuybackStatus == models.PayoutReceived
	if allPaid && buybackDone {
		v.Status = models.AltInvestmentCompleted
	} else {
		v.Status = models.AltInvestmentActive
	}

	v.XIRR = computeAltInvestmentXIRR(v, now)
}

func altInvestmentBookValue(v *models.AlternateInvestment, now time.Time) float64 {
	if v.BuybackStatus == models.PayoutReceived {
		return 0
	}
	switch v.ValuationMethod {
	case models.AltValuationAtCost:
		return v.InvestmentAmount
	case models.AltValuationManual:
		return v.ManualValue
	}
	if !now.After(v.PurchaseDate) {
		return v.InvestmentAmount
	}
	if !now.Before(v.AgreementEndDate) {
		return v.BuybackAmount
	}
	total := v.AgreementEndDate.Sub(v.PurchaseDate).Hours()
	if total <= 0 {
		return v.BuybackAmount
	}
	elapsed := now.Sub(v.PurchaseDate).Hours()
	return round2(v.InvestmentAmount - (v.InvestmentAmount-v.BuybackAmount)*elapsed/total)
}

// pendingAltPayoutsBetween sums the pending payouts scheduled in (from, to],
// continuing an open-ended schedule beyond the payouts generated so far.
// Requires ComputeAlternateInvestment to have set MonthlyPayout.
func pendingAltPayoutsBetween(v *models.AlternateInvestment, from, to time.Time) float64 {
	sum := 0.0
	for _, p := range v.Payouts {
		if p.Status == models.PayoutPending && p.ScheduledDate.After(from) && !p.ScheduledDate.After(to) {
			sum += p.Amount
		}
	}
	if v.TenureMonths <= 0 {
		for i := len(v.Payouts); ; i++ {
			d := addMonthsClamped(v.PayoutStartDate, i)
			if d.After(to) {
				break
			}
			if d.After(from) {
				sum += v.MonthlyPayout
			}
		}
	}
	return sum
}

// AlternateInvestmentCashflows returns the actual cash movements (purchase
// outflow, received payouts and buyback) used for XIRR and portfolio-level
// calculations.
func AlternateInvestmentCashflows(v *models.AlternateInvestment) []Cashflow {
	flows := []Cashflow{{Date: v.PurchaseDate, Amount: -v.InvestmentAmount}}
	for _, p := range v.Payouts {
		if p.Status == models.PayoutReceived && p.ReceivedAmount > 0 {
			d := p.ScheduledDate
			if p.ReceivedDate != nil {
				d = *p.ReceivedDate
			}
			flows = append(flows, Cashflow{Date: d, Amount: p.ReceivedAmount})
		}
	}
	if v.BuybackStatus == models.PayoutReceived && v.BuybackReceived > 0 && v.BuybackReceivedDate != nil {
		flows = append(flows, Cashflow{Date: *v.BuybackReceivedDate, Amount: v.BuybackReceived})
	}
	return flows
}

func computeAltInvestmentXIRR(v *models.AlternateInvestment, now time.Time) float64 {
	flows := AlternateInvestmentCashflows(v)
	if v.CurrentValue > 0 {
		flows = append(flows, Cashflow{Date: now, Amount: v.CurrentValue})
	}
	xirr, err := CalculateXIRR(flows)
	if err != nil || math.IsNaN(xirr) || math.IsInf(xirr, 0) {
		return 0
	}
	return round2(xirr * 100)
}

// altInvestmentName is the display name used across reports: "Provider - Asset".
func altInvestmentName(v *models.AlternateInvestment) string {
	if v.AssetName != "" {
		return v.Provider + " - " + v.AssetName
	}
	return v.Provider
}

// addMonthsClamped adds months to t, clamping the day to the target month's
// last day (e.g. 31 Jan + 1 month = 28/29 Feb rather than 2/3 Mar).
func addMonthsClamped(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	lastDay := first.AddDate(0, 1, -1).Day()
	day := t.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(first.Year(), first.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func receivedDateOrDefault(s string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return fallback, nil
	}
	d, err := parseDate(s)
	if err != nil {
		return time.Time{}, invalidf("invalid received_date: %v", err)
	}
	return d, nil
}

func amountOrDefault(amount, fallback float64) float64 {
	if amount > 0 {
		return round2(amount)
	}
	return fallback
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}
