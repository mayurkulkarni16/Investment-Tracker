package services

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("invalid purchase_date: %w", err)
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, fmt.Errorf("invalid maturity_date: %w", err)
	}

	bond := &models.CorporateBond{
		UserID:             userID,
		BondName:           req.BondName,
		Issuer:             req.Issuer,
		PurchaseDate:       purchaseDate,
		InvestmentAmount:   req.InvestmentAmount,
		CouponRate:         req.CouponRate,
		InterestPayoutFreq: req.InterestPayoutFreq,
		MaturityDate:       maturityDate,
		MaturityType:       req.MaturityType,
		RemainingPrincipal: req.InvestmentAmount,
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
			Amount:        req.InvestmentAmount,
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
	if months == 0 {
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

	currentPrincipal := bond.InvestmentAmount

	// Payouts happen on the maturity date's day-of-month, not the purchase day.
	// First payout is never in the same month as purchase — always starts from the next month.
	maturityDay := bond.MaturityDate.Day()
	payoutDate := time.Date(bond.PurchaseDate.Year(), bond.PurchaseDate.Month()+1, maturityDay, 0, 0, 0, 0, bond.PurchaseDate.Location())
	eventIdx := 0

	// Track previous date for actual day count calculation
	prevDate := bond.PurchaseDate

	for !payoutDate.After(bond.MaturityDate) {
		// Check if any principal repayments happened before this payout date
		for eventIdx < len(events) && !events[eventIdx].date.After(payoutDate) {
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

		prevDate = payoutDate
		payoutDate = payoutDate.AddDate(0, months, 0)
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
	bond.RemainingPrincipal = bond.InvestmentAmount
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

	return nil, fmt.Errorf("payout with ID %s not found", payoutID)
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

	purchaseDate, err := parseDate(req.PurchaseDate)
	if err != nil {
		return nil, fmt.Errorf("invalid purchase_date: %w", err)
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, fmt.Errorf("invalid maturity_date: %w", err)
	}

	bond.BondName = req.BondName
	bond.Issuer = req.Issuer
	bond.PurchaseDate = purchaseDate
	bond.InvestmentAmount = req.InvestmentAmount
	bond.CouponRate = req.CouponRate
	bond.InterestPayoutFreq = req.InterestPayoutFreq
	bond.MaturityDate = maturityDate
	bond.MaturityType = req.MaturityType
	bond.Notes = req.Notes

	// Always rebuild schedules from scratch on update
	bond.RemainingPrincipal = req.InvestmentAmount
	bond.TotalInterestEarned = 0
	bond.TotalPrincipalReturned = 0
	bond.PrincipalRepayments = s.buildPrincipalSchedule(req, maturityDate)
	bond.InterestPayouts = s.generateInterestSchedule(bond)

	// recalculateBond will auto-mark past payouts as received on next GetAll

	if err := s.repo.Update(ctx, bond); err != nil {
		return nil, err
	}
	return bond, nil
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
