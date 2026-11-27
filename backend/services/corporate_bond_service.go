package services

import (
	"context"
	"math"
	"sort"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type CorporateBondService struct {
	repo *repository.CorporateBondRepo
}

func NewCorporateBondService(repo *repository.CorporateBondRepo) *CorporateBondService {
	return &CorporateBondService{repo: repo}
}

func (s *CorporateBondService) Create(ctx context.Context, userID string, req models.CreateCorporateBondRequest) (*models.CorporateBond, error) {
	purchaseDate, err := parseDate(req.PurchaseDate)
	if err != nil {
		return nil, invalidf("invalid purchase_date: %v", err)
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, invalidf("invalid maturity_date: %v", err)
	}

	faceValue := req.FaceValue
	if faceValue <= 0 {
		faceValue = req.InvestmentAmount
	}
	payoutDay := req.PayoutDay
	if payoutDay < 1 {
		payoutDay = maturityDate.Day()
	}

	bond := &models.CorporateBond{
		UserID:             userID,
		BondName:           req.BondName,
		Issuer:             req.Issuer,
		PurchaseDate:       purchaseDate,
		InvestmentAmount:   req.InvestmentAmount,
		FaceValue:          faceValue,
		CouponRate:         req.CouponRate,
		YieldToMaturity:    req.YieldToMaturity,
		InterestPayoutFreq: req.InterestPayoutFreq,
		PayoutDay:          payoutDay,
		MaturityDate:       maturityDate,
		MaturityType:       req.MaturityType,
		RemainingPrincipal: faceValue,
		Status:             models.BondActive,
		Notes:              req.Notes,
	}

	// Build principal repayment schedule
	bond.PrincipalRepayments = s.buildPrincipalSchedule(req, maturityDate)

	// Generate interest payout schedule
	bond.InterestPayouts = s.generateInterestSchedule(bond)

	if err := s.repo.Create(ctx, bond); err != nil {
		return nil, err
	}
	return bond, nil
}

func (s *CorporateBondService) buildPrincipalSchedule(req models.CreateCorporateBondRequest, maturityDate time.Time) []models.PrincipalRepayment {
	var repayments []models.PrincipalRepayment

	if req.MaturityType == models.MaturityBullet {
		repayments = append(repayments, models.PrincipalRepayment{
			RepaymentID:   uuid.New().String(),
			ScheduledDate: maturityDate,
			Amount:        bondFaceValueFromRequest(req),
			Status:        models.PayoutPending,
		})
	} else {
		for _, r := range req.PrincipalRepayments {
			d, _ := parseDate(r.ScheduledDate)
			repayments = append(repayments, models.PrincipalRepayment{
				RepaymentID:   uuid.New().String(),
				ScheduledDate: d,
				Amount:        r.Amount,
				Status:        models.PayoutPending,
			})
		}
		sort.Slice(repayments, func(i, j int) bool {
			return repayments[i].ScheduledDate.Before(repayments[j].ScheduledDate)
		})
	}

	return repayments
}

// generateInterestSchedule creates the full interest payout schedule.
// For staggered bonds, interest is recalculated based on remaining principal
// after each principal repayment.
func (s *CorporateBondService) generateInterestSchedule(bond *models.CorporateBond) []models.InterestPayout {
	var payouts []models.InterestPayout

	months := frequencyToMonths(bond.InterestPayoutFreq)
	if months == 0 || bond.CouponRate <= 0 {
		return payouts
	}

	// Build a timeline of principal changes
	type principalEvent struct {
		date   time.Time
		amount float64 // principal reduction
	}
	var events []principalEvent
	for _, r := range bond.PrincipalRepayments {
		events = append(events, principalEvent{date: r.ScheduledDate, amount: r.Amount})
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].date.Before(events[j].date)
	})

	currentPrincipal := bondFaceValue(bond)
	payoutDay := bond.PayoutDay
	if payoutDay < 1 || payoutDay > 31 {
		payoutDay = bond.MaturityDate.Day()
	}
	eventIdx := 0

	for period := months; ; period += months {
		payoutDate := bondDateAtMonthOffset(bond.PurchaseDate, period, payoutDay)
		if payoutDate.After(bond.MaturityDate) {
			break
		}
		prevDate := bondDateAtMonthOffset(bond.PurchaseDate, period-months, payoutDay)
		// Check if any principal repayments happened before this payout date
		for eventIdx < len(events) && events[eventIdx].date.Before(payoutDate) {
			currentPrincipal -= events[eventIdx].amount
			eventIdx++
		}

		if currentPrincipal <= 0 {
			break
		}

		// Calculate interest based on actual day count (ACT/365)
		days := int(payoutDate.Sub(prevDate).Hours() / 24)
		interestAmount := s.calculateInterestByDays(currentPrincipal, bond.CouponRate, days)

		payouts = append(payouts, models.InterestPayout{
			PayoutID:        uuid.New().String(),
			ScheduledDate:   payoutDate,
			PrincipalAtTime: currentPrincipal,
			Amount:          interestAmount,
			Status:          models.PayoutPending,
		})

		for eventIdx < len(events) && events[eventIdx].date.Equal(payoutDate) {
			currentPrincipal -= events[eventIdx].amount
			eventIdx++
		}
	}

	return payouts
}

func (s *CorporateBondService) calculateInterestByDays(principal, annualRate float64, days int) float64 {
	interest := principal * (annualRate / 100) * float64(days) / 365.0
	return math.Round(interest*100) / 100
}

func (s *CorporateBondService) GetAll(ctx context.Context, userID string) ([]models.CorporateBond, error) {
	bonds, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range bonds {
		s.recalculateBond(&bonds[i])
		bonds[i].XIRR = ComputeCorporateBondXIRR(&bonds[i])
	}
	return bonds, nil
}

func (s *CorporateBondService) GetByID(ctx context.Context, id string) (*models.CorporateBond, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	bond, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.recalculateBond(bond)
	bond.XIRR = ComputeCorporateBondXIRR(bond)
	return bond, nil
}

// recalculateBond recomputes TotalInterestEarned, TotalPrincipalReturned,
// RemainingPrincipal, and Status from the payout schedules.
// Interest payouts whose scheduled date has passed are considered earned.
// Principal repayments whose scheduled date has passed are considered returned.
func (s *CorporateBondService) recalculateBond(bond *models.CorporateBond) {
	now := time.Now()

	bond.TotalInterestEarned = 0
	for i := range bond.InterestPayouts {
		if bond.InterestPayouts[i].Status == models.PayoutReceived ||
			!bond.InterestPayouts[i].ScheduledDate.After(now) {
			bond.TotalInterestEarned += bond.InterestPayouts[i].Amount
			// Auto-mark past payouts as received
			if bond.InterestPayouts[i].Status == models.PayoutPending &&
				!bond.InterestPayouts[i].ScheduledDate.After(now) {
				bond.InterestPayouts[i].Status = models.PayoutReceived
				d := bond.InterestPayouts[i].ScheduledDate
				bond.InterestPayouts[i].ReceivedDate = &d
			}
		}
	}

	bond.TotalPrincipalReturned = 0
	bond.RemainingPrincipal = bondFaceValue(bond)
	for i := range bond.PrincipalRepayments {
		if bond.PrincipalRepayments[i].Status == models.PayoutReceived ||
			!bond.PrincipalRepayments[i].ScheduledDate.After(now) {
			bond.TotalPrincipalReturned += bond.PrincipalRepayments[i].Amount
			bond.RemainingPrincipal -= bond.PrincipalRepayments[i].Amount
			// Auto-mark past repayments as received
			if bond.PrincipalRepayments[i].Status == models.PayoutPending &&
				!bond.PrincipalRepayments[i].ScheduledDate.After(now) {
				bond.PrincipalRepayments[i].Status = models.PayoutReceived
				d := bond.PrincipalRepayments[i].ScheduledDate
				bond.PrincipalRepayments[i].ReceivedDate = &d
			}
		}
	}

	if bond.RemainingPrincipal < 0 {
		bond.RemainingPrincipal = 0
	}

	// Update status
	if bond.RemainingPrincipal <= 0 {
		bond.Status = models.BondMatured
	} else if bond.TotalPrincipalReturned > 0 {
		bond.Status = models.BondPartiallyMatured
	} else {
		bond.Status = models.BondActive
	}

	// Recalculate future interest for staggered bonds based on actual remaining principal
	if bond.MaturityType == models.MaturityStaggered {
		for i := range bond.InterestPayouts {
			if bond.InterestPayouts[i].Status == models.PayoutPending {
				// Find the previous payout date for day count
				prevDate := bond.PurchaseDate
				for j := i - 1; j >= 0; j-- {
					prevDate = bond.InterestPayouts[j].ScheduledDate
					break
				}
				days := int(bond.InterestPayouts[i].ScheduledDate.Sub(prevDate).Hours() / 24)
				bond.InterestPayouts[i].PrincipalAtTime = bond.RemainingPrincipal
				bond.InterestPayouts[i].Amount = s.calculateInterestByDays(
					bond.RemainingPrincipal, bond.CouponRate, days,
				)
			}
		}
	}
}

func (s *CorporateBondService) MarkPayoutReceived(ctx context.Context, bondID, payoutID string, receivedDate time.Time) (*models.CorporateBond, error) {
	objID, err := parseObjectID(bondID)
	if err != nil {
		return nil, err
	}

	bond, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	// Check interest payouts
	for i := range bond.InterestPayouts {
		if bond.InterestPayouts[i].PayoutID == payoutID {
			bond.InterestPayouts[i].Status = models.PayoutReceived
			bond.InterestPayouts[i].ReceivedDate = &receivedDate
			bond.TotalInterestEarned += bond.InterestPayouts[i].Amount
			return bond, s.repo.Update(ctx, bond)
		}
	}

	// Check principal repayments
	for i := range bond.PrincipalRepayments {
		if bond.PrincipalRepayments[i].RepaymentID == payoutID {
			bond.PrincipalRepayments[i].Status = models.PayoutReceived
			bond.PrincipalRepayments[i].ReceivedDate = &receivedDate
			bond.TotalPrincipalReturned += bond.PrincipalRepayments[i].Amount
			bond.RemainingPrincipal -= bond.PrincipalRepayments[i].Amount

			// Recalculate future interest payouts based on new remaining principal
			s.recalculateFutureInterest(bond, receivedDate)

			// Update bond status
			if bond.RemainingPrincipal <= 0 {
				bond.Status = models.BondMatured
			} else if bond.TotalPrincipalReturned > 0 {
				bond.Status = models.BondPartiallyMatured
			}

			return bond, s.repo.Update(ctx, bond)
		}
	}

	return nil, notFoundf("payout with ID %s not found", payoutID)
}

// recalculateFutureInterest updates all pending interest payouts after a principal repayment
func (s *CorporateBondService) recalculateFutureInterest(bond *models.CorporateBond, afterDate time.Time) {
	for i := range bond.InterestPayouts {
		if bond.InterestPayouts[i].Status == models.PayoutPending &&
			bond.InterestPayouts[i].ScheduledDate.After(afterDate) {
			// Find the previous payout date for day count
			prevDate := bond.PurchaseDate
			for j := i - 1; j >= 0; j-- {
				prevDate = bond.InterestPayouts[j].ScheduledDate
				break
			}
			days := int(bond.InterestPayouts[i].ScheduledDate.Sub(prevDate).Hours() / 24)
			bond.InterestPayouts[i].PrincipalAtTime = bond.RemainingPrincipal
			bond.InterestPayouts[i].Amount = s.calculateInterestByDays(
				bond.RemainingPrincipal, bond.CouponRate, days,
			)
		}
	}
}

func (s *CorporateBondService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *CorporateBondService) Update(ctx context.Context, id string, req models.CreateCorporateBondRequest) (*models.CorporateBond, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	bond, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	previousInterestPayouts := append([]models.InterestPayout(nil), bond.InterestPayouts...)
	previousPrincipalRepayments := append([]models.PrincipalRepayment(nil), bond.PrincipalRepayments...)

	purchaseDate, err := parseDate(req.PurchaseDate)
	if err != nil {
		return nil, invalidf("invalid purchase_date: %v", err)
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, invalidf("invalid maturity_date: %v", err)
	}

	bond.BondName = req.BondName
	bond.Issuer = req.Issuer
	faceValue := req.FaceValue
	if faceValue <= 0 {
		faceValue = req.InvestmentAmount
	}
	payoutDay := req.PayoutDay
	if payoutDay < 1 {
		payoutDay = maturityDate.Day()
	}

	bond.PurchaseDate = purchaseDate
	bond.InvestmentAmount = req.InvestmentAmount
	bond.FaceValue = faceValue
	bond.CouponRate = req.CouponRate
	bond.YieldToMaturity = req.YieldToMaturity
	bond.InterestPayoutFreq = req.InterestPayoutFreq
	bond.PayoutDay = payoutDay
	bond.MaturityDate = maturityDate
	bond.MaturityType = req.MaturityType
	bond.Notes = req.Notes

	// Rebuild schedules while retaining receipts already recorded by the user.
	bond.RemainingPrincipal = faceValue
	bond.TotalInterestEarned = 0
	bond.TotalPrincipalReturned = 0
	bond.PrincipalRepayments = s.buildPrincipalSchedule(req, maturityDate)
	bond.InterestPayouts = s.generateInterestSchedule(bond)
	preserveReceivedInterestPayouts(previousInterestPayouts, bond.InterestPayouts)
	preserveReceivedPrincipalRepayments(previousPrincipalRepayments, bond.PrincipalRepayments)
	s.recalculateBond(bond)
	bond.XIRR = ComputeCorporateBondXIRR(bond)

	if err := s.repo.Update(ctx, bond); err != nil {
		return nil, err
	}
	return bond, nil
}

func preserveReceivedInterestPayouts(previous, current []models.InterestPayout) {
	for i := 0; i < len(previous) && i < len(current); i++ {
		if previous[i].Status == models.PayoutReceived {
			current[i].Status = models.PayoutReceived
			current[i].ReceivedDate = previous[i].ReceivedDate
		}
	}
}

func preserveReceivedPrincipalRepayments(previous, current []models.PrincipalRepayment) {
	for i := 0; i < len(previous) && i < len(current); i++ {
		if previous[i].Status == models.PayoutReceived {
			current[i].Status = models.PayoutReceived
			current[i].ReceivedDate = previous[i].ReceivedDate
		}
	}
}

func bondFaceValueFromRequest(req models.CreateCorporateBondRequest) float64 {
	if req.FaceValue > 0 {
		return req.FaceValue
	}
	return req.InvestmentAmount
}

func bondFaceValue(bond *models.CorporateBond) float64 {
	if bond.FaceValue > 0 {
		return bond.FaceValue
	}
	return bond.InvestmentAmount
}

func bondDateAtMonthOffset(anchor time.Time, monthOffset, day int) time.Time {
	first := time.Date(anchor.Year(), anchor.Month()+time.Month(monthOffset), 1, 0, 0, 0, 0, anchor.Location())
	lastDay := first.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, anchor.Location())
}

func frequencyToMonths(freq models.PayoutFrequency) int {
	switch freq {
	case models.PayoutMonthly:
		return 1
	case models.PayoutQuarterly:
		return 3
	case models.PayoutBiannually:
		return 6
	case models.PayoutAnnually:
		return 12
	default:
		return 0
	}
}
