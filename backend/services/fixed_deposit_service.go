package services

import (
	"context"
	"math"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type FixedDepositService struct {
	repo *repository.FixedDepositRepo
}

func NewFixedDepositService(repo *repository.FixedDepositRepo) *FixedDepositService {
	return &FixedDepositService{repo: repo}
}

func (s *FixedDepositService) Create(ctx context.Context, req models.CreateFixedDepositRequest) (*models.FixedDeposit, error) {
	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	maturityDate, err := parseDate(req.MaturityDate)
	if err != nil {
		return nil, err
	}

	fd := &models.FixedDeposit{
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

func (s *FixedDepositService) GetAll(ctx context.Context) ([]models.FixedDeposit, error) {
	return s.repo.GetAll(ctx)
}

func (s *FixedDepositService) GetByID(ctx context.Context, id string) (*models.FixedDeposit, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, objID)
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
