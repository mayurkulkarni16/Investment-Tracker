package services

import (
	"context"
	"math"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type VehicleRentalService struct {
	repo *repository.VehicleRentalRepo
}

func NewVehicleRentalService(repo *repository.VehicleRentalRepo) *VehicleRentalService {
	return &VehicleRentalService{repo: repo}
}

func (s *VehicleRentalService) Create(ctx context.Context, userID string, req models.CreateVehicleRentalRequest) (*models.VehicleRental, error) {
	v := &models.VehicleRental{UserID: userID, BuybackStatus: models.PayoutPending}
	if err := applyVehicleRentalRequest(v, req); err != nil {
		return nil, err
	}
	v.Payouts = buildRentalSchedule(v, nil)

	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	ComputeVehicleRental(v, time.Now())
	return v, nil
}

func (s *VehicleRentalService) GetAll(ctx context.Context, userID string) ([]models.VehicleRental, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		ComputeVehicleRental(&items[i], now)
	}
	return items, nil
}

func (s *VehicleRentalService) GetByID(ctx context.Context, id string) (*models.VehicleRental, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	ComputeVehicleRental(v, time.Now())
	return v, nil
}

// Update changes the agreement terms and regenerates the payout schedule,
// carrying over receipts already recorded for each month.
func (s *VehicleRentalService) Update(ctx context.Context, id string, req models.CreateVehicleRentalRequest) (*models.VehicleRental, error) {
	v, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyVehicleRentalRequest(v, req); err != nil {
		return nil, err
	}
	v.Payouts = buildRentalSchedule(v, v.Payouts)
	return s.save(ctx, v)
}

func (s *VehicleRentalService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *VehicleRentalService) UpdatePayout(ctx context.Context, id, payoutID string, req models.UpdateRentalPayoutRequest) (*models.VehicleRental, error) {
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
func (s *VehicleRentalService) MarkDueReceived(ctx context.Context, id string) (*models.VehicleRental, error) {
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

func (s *VehicleRentalService) UpdateBuyback(ctx context.Context, id string, req models.UpdateRentalPayoutRequest) (*models.VehicleRental, error) {
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

func (s *VehicleRentalService) load(ctx context.Context, id string) (*models.VehicleRental, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, objID)
}

func (s *VehicleRentalService) save(ctx context.Context, v *models.VehicleRental) (*models.VehicleRental, error) {
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	ComputeVehicleRental(v, time.Now())
	return v, nil
}

func applyVehicleRentalRequest(v *models.VehicleRental, req models.CreateVehicleRentalRequest) error {
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

	v.Provider = strings.TrimSpace(req.Provider)
	v.VehicleModel = strings.TrimSpace(req.VehicleModel)
	v.Units = req.Units
	v.UnitPrice = req.UnitPrice
	v.GSTPercent = req.GSTPercent
	v.OtherCharges = req.OtherCharges
	v.PurchaseDate = purchaseDate
	v.PayoutStartDate = payoutStart
	v.TenureMonths = req.TenureMonths
	v.MonthlyPayoutPerUnit = req.MonthlyPayoutPerUnit
	v.BuybackPercent = req.BuybackPercent
	v.AgreementNumber = strings.TrimSpace(req.AgreementNumber)
	v.Notes = req.Notes

	v.RegistrationNumbers = []string{}
	for _, r := range req.RegistrationNumbers {
		if r = strings.ToUpper(strings.TrimSpace(r)); r != "" {
			v.RegistrationNumbers = append(v.RegistrationNumbers, r)
		}
	}

	vehicleCost := float64(v.Units) * v.UnitPrice
	v.InvestmentAmount = round2(vehicleCost*(1+v.GSTPercent/100) + v.OtherCharges)
	v.BuybackAmount = round2(vehicleCost * v.BuybackPercent / 100)
	if v.BuybackStatus == "" {
		v.BuybackStatus = models.PayoutPending
	}
	return nil
}

// buildRentalSchedule generates one payout per month of the tenure. Receipts in
// previous (by month index) are preserved so editing terms never loses history.
func buildRentalSchedule(v *models.VehicleRental, previous []models.RentalPayout) []models.RentalPayout {
	amount := round2(float64(v.Units) * v.MonthlyPayoutPerUnit)
	payouts := make([]models.RentalPayout, v.TenureMonths)
	for i := range payouts {
		p := models.RentalPayout{
			PayoutID:      uuid.New().String(),
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

// ComputeVehicleRental fills in the derived (non-persisted) fields: payout
// progress, overdue tracking, book value, total return and XIRR.
// Book value declines straight-line from the purchase cost to the buyback value
// between the purchase date and the last scheduled payout.
func ComputeVehicleRental(v *models.VehicleRental, now time.Time) {
	v.MonthlyPayout = round2(float64(v.Units) * v.MonthlyPayoutPerUnit)
	v.AgreementEndDate = v.PayoutStartDate
	if n := len(v.Payouts); n > 0 {
		v.AgreementEndDate = v.Payouts[n-1].ScheduledDate
	}

	v.ExpectedTotalPayout, v.TotalPayoutsReceived = 0, 0
	v.PayoutsReceivedCount, v.OverdueCount, v.OverdueAmount = 0, 0, 0
	v.NextPayoutDate = nil
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	overdueBefore := today.AddDate(0, 0, -models.RentalOverdueGraceDays)

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

	v.CurrentValue = vehicleRentalBookValue(v, now)

	recovered := v.TotalPayoutsReceived + v.BuybackReceived
	v.TotalReturn = round2(recovered + v.CurrentValue - v.InvestmentAmount)
	v.CapitalRecoveredPct = 0
	if v.InvestmentAmount > 0 {
		v.CapitalRecoveredPct = round2(recovered / v.InvestmentAmount * 100)
	}

	allPaid := v.PayoutsReceivedCount == len(v.Payouts)
	buybackDone := v.BuybackAmount <= 0 || v.BuybackStatus == models.PayoutReceived
	if allPaid && buybackDone {
		v.Status = models.VehicleRentalCompleted
	} else {
		v.Status = models.VehicleRentalActive
	}

	v.XIRR = computeVehicleRentalXIRR(v, now)
}

func vehicleRentalBookValue(v *models.VehicleRental, now time.Time) float64 {
	if v.BuybackStatus == models.PayoutReceived {
		return 0
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

// VehicleRentalCashflows returns the actual cash movements (purchase outflow,
// received payouts and buyback) used for XIRR and portfolio-level calculations.
func VehicleRentalCashflows(v *models.VehicleRental) []Cashflow {
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

func computeVehicleRentalXIRR(v *models.VehicleRental, now time.Time) float64 {
	flows := VehicleRentalCashflows(v)
	if v.CurrentValue > 0 {
		flows = append(flows, Cashflow{Date: now, Amount: v.CurrentValue})
	}
	xirr, err := CalculateXIRR(flows)
	if err != nil || math.IsNaN(xirr) || math.IsInf(xirr, 0) {
		return 0
	}
	return round2(xirr * 100)
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
