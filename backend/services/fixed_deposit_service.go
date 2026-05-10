package services

import (
	"context"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type FixedDepositService struct {
	repo *repository.FixedDepositRepo
}

func NewFixedDepositService(repo *repository.FixedDepositRepo) *FixedDepositService {
	return &FixedDepositService{repo: repo}
}

func (s *FixedDepositService) Create(ctx context.Context, userID string, req models.CreateFixedDepositRequest) (*models.FixedDeposit, error) {
	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, err
	}

	fd := &models.FixedDeposit{
		UserID:          userID,
		BankName:        req.BankName,
		FDNumber:        req.FDNumber,
		PrincipalAmount: req.PrincipalAmount,
		InterestRate:    req.InterestRate,
		StartDate:       startDate,
		MaturityDate:    maturityDate,
		TenureMonths:    req.TenureMonths,
		InterestType:    req.InterestType,
		PayoutFrequency: req.PayoutFrequency,
		IsAutoRenewed:   req.IsAutoRenewed,
		Status:          models.FDActive,
		Notes:           req.Notes,
	}

	s.calculateMaturity(fd)
	s.generateInterestSchedule(fd)

	if err := s.repo.Create(ctx, fd); err != nil {
		return nil, err
	}
	return fd, nil
}

func (s *FixedDepositService) calculateMaturity(fd *models.FixedDeposit) {
	rate := fd.InterestRate / 100
	years := float64(fd.TenureMonths) / 12.0

	if fd.InterestType == models.InterestCumulative {
		// Quarterly compounding (standard for Indian FDs)
		n := 4.0
		fd.MaturityAmount = fd.PrincipalAmount * math.Pow(1+rate/n, n*years)
		fd.MaturityAmount = math.Round(fd.MaturityAmount*100) / 100
		fd.InterestEarned = fd.MaturityAmount - fd.PrincipalAmount
	} else {
		// Simple interest for non-cumulative
		fd.InterestEarned = math.Round(fd.PrincipalAmount*rate*years*100) / 100
		fd.MaturityAmount = fd.PrincipalAmount // principal returned at maturity
	}
}

func (s *FixedDepositService) generateInterestSchedule(fd *models.FixedDeposit) {
	fd.InterestPayouts = nil

	if fd.InterestType != models.InterestNonCumulative || fd.PayoutFrequency == nil {
		return
	}

	freq := *fd.PayoutFrequency
	if freq == models.PayoutAtMaturity {
		// Single payout at maturity
		fd.InterestPayouts = []models.InterestPayout{{
			PayoutID:        uuid.New().String(),
			ScheduledDate:   fd.MaturityDate,
			PrincipalAtTime: fd.PrincipalAmount,
			Amount:          fd.InterestEarned,
			Status:          models.PayoutPending,
		}}
		s.autoMarkPastPayouts(fd)
		return
	}

	months := frequencyToMonths(freq)
	if months == 0 {
		return
	}

	var payouts []models.InterestPayout
	payoutDate := fd.StartDate.AddDate(0, months, 0)
	prevDate := fd.StartDate

	for !payoutDate.After(fd.MaturityDate) {
		days := int(payoutDate.Sub(prevDate).Hours() / 24)
		interest := fd.PrincipalAmount * (fd.InterestRate / 100) * float64(days) / 365.0
		interest = math.Round(interest*100) / 100

		payouts = append(payouts, models.InterestPayout{
			PayoutID:        uuid.New().String(),
			ScheduledDate:   payoutDate,
			PrincipalAtTime: fd.PrincipalAmount,
			Amount:          interest,
			Status:          models.PayoutPending,
		})

		prevDate = payoutDate
		payoutDate = payoutDate.AddDate(0, months, 0)
	}

	// Add final partial period payout if last payout didn't land on maturity
	if len(payouts) > 0 && payouts[len(payouts)-1].ScheduledDate.Before(fd.MaturityDate) {
		lastDate := payouts[len(payouts)-1].ScheduledDate
		days := int(fd.MaturityDate.Sub(lastDate).Hours() / 24)
		if days > 0 {
			interest := fd.PrincipalAmount * (fd.InterestRate / 100) * float64(days) / 365.0
			interest = math.Round(interest*100) / 100
			payouts = append(payouts, models.InterestPayout{
				PayoutID:        uuid.New().String(),
				ScheduledDate:   fd.MaturityDate,
				PrincipalAtTime: fd.PrincipalAmount,
				Amount:          interest,
				Status:          models.PayoutPending,
			})
		}
	}

	fd.InterestPayouts = payouts
	s.autoMarkPastPayouts(fd)
}

func (s *FixedDepositService) autoMarkPastPayouts(fd *models.FixedDeposit) {
	now := time.Now()
	for i := range fd.InterestPayouts {
		if fd.InterestPayouts[i].Status == models.PayoutPending &&
			!fd.InterestPayouts[i].ScheduledDate.After(now) {
			fd.InterestPayouts[i].Status = models.PayoutReceived
			d := fd.InterestPayouts[i].ScheduledDate
			fd.InterestPayouts[i].ReceivedDate = &d
		}
	}
}

func (s *FixedDepositService) GetAll(ctx context.Context, userID string) ([]models.FixedDeposit, error) {
	fds, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range fds {
		s.autoMarkPastPayouts(&fds[i])
		fds[i].XIRR = ComputeFixedDepositXIRR(&fds[i])
	}
	return fds, nil
}

func (s *FixedDepositService) GetByID(ctx context.Context, id string) (*models.FixedDeposit, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	fd, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	fd.XIRR = ComputeFixedDepositXIRR(fd)
	s.autoMarkPastPayouts(fd)
	return fd, nil
}

func (s *FixedDepositService) Update(ctx context.Context, id string, req models.CreateFixedDepositRequest) (*models.FixedDeposit, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	fd, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, err
	}

	fd.BankName = req.BankName
	fd.FDNumber = req.FDNumber
	fd.PrincipalAmount = req.PrincipalAmount
	fd.InterestRate = req.InterestRate
	fd.StartDate = startDate
	fd.MaturityDate = maturityDate
	fd.TenureMonths = req.TenureMonths
	fd.InterestType = req.InterestType
	fd.PayoutFrequency = req.PayoutFrequency
	fd.IsAutoRenewed = req.IsAutoRenewed
	fd.Notes = req.Notes

	s.calculateMaturity(fd)
	s.generateInterestSchedule(fd)

	if err := s.repo.Update(ctx, fd); err != nil {
		return nil, err
	}
	return fd, nil
}

func (s *FixedDepositService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}
