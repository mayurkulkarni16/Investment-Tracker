package services

import (
	"context"
	"math"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type PersonalLoanService struct {
	repo *repository.PersonalLoanRepo
}

func NewPersonalLoanService(repo *repository.PersonalLoanRepo) *PersonalLoanService {
	return &PersonalLoanService{repo: repo}
}

func (s *PersonalLoanService) Create(ctx context.Context, userID string, req models.CreatePersonalLoanRequest) (*models.PersonalLoan, error) {
	emiStart, err := parseDate(req.EMIStartDate)
	if err != nil {
		return nil, err
	}
	disbDate, err := parseDate(req.DisbursementDate)
	if err != nil {
		return nil, err
	}

	emi := calculateEMI(req.DisbursedAmount, req.InterestRate, req.TenureMonths)
	endDate := emiStart.AddDate(0, req.TenureMonths, 0)

	loan := &models.PersonalLoan{
		UserID:                userID,
		LenderName:            req.LenderName,
		LoanAccountNumber:     req.LoanAccountNumber,
		LoanPurpose:           req.LoanPurpose,
		PrincipalAmount:       req.PrincipalAmount,
		DisbursedAmount:       req.DisbursedAmount,
		InterestRate:          req.InterestRate,
		RateType:              req.RateType,
		TenureMonths:          req.TenureMonths,
		EMIAmount:             emi,
		EMIStartDate:          emiStart,
		DisbursementDate:      disbDate,
		ProcessingFee:         req.ProcessingFee,
		ForeclosureCharges:    req.ForeclosureCharges,
		RateChangeHistory:     []models.RateChangeEntry{},
		EMIsPaid:              []models.EMIEntry{},
		Prepayments:           []models.Prepayment{},
		OutstandingPrincipal:  req.DisbursedAmount,
		RemainingTenureMonths: req.TenureMonths,
		LoanEndDate:           endDate,
		Status:                models.LoanActive,
		Notes:                 req.Notes,
	}

	if err := s.repo.Create(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *PersonalLoanService) GetAll(ctx context.Context, userID string) ([]models.PersonalLoan, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *PersonalLoanService) GetByID(ctx context.Context, id string) (*models.PersonalLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, objID)
}

func (s *PersonalLoanService) Update(ctx context.Context, id string, req models.UpdatePersonalLoanRequest) (*models.PersonalLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.LenderName != "" {
		loan.LenderName = req.LenderName
	}
	if req.LoanAccountNumber != "" {
		loan.LoanAccountNumber = req.LoanAccountNumber
	}
	if req.LoanPurpose != "" {
		loan.LoanPurpose = req.LoanPurpose
	}
	if req.Notes != "" {
		loan.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *PersonalLoanService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *PersonalLoanService) RecordEMIPayment(ctx context.Context, id string, req models.AddEMIPaymentRequest) (*models.PersonalLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	paidDate, err := parseDate(req.PaidDate)
	if err != nil {
		return nil, err
	}

	monthlyRate := loan.InterestRate / 100 / 12
	interestPortion := math.Round(loan.OutstandingPrincipal*monthlyRate*100) / 100
	principalPortion := math.Round((loan.EMIAmount-interestPortion)*100) / 100
	outstandingAfter := math.Round((loan.OutstandingPrincipal-principalPortion)*100) / 100
	if outstandingAfter < 0 {
		outstandingAfter = 0
	}

	entry := models.EMIEntry{
		EMIID:            uuid.New().String(),
		Month:            req.Month,
		DueDate:          paidDate,
		EMIAmount:        loan.EMIAmount,
		PrincipalPortion: principalPortion,
		InterestPortion:  interestPortion,
		OutstandingAfter: outstandingAfter,
		Paid:             true,
		PaidDate:         &paidDate,
	}

	loan.EMIsPaid = append(loan.EMIsPaid, entry)
	loan.OutstandingPrincipal = outstandingAfter
	loan.TotalPrincipalPaid += principalPortion
	loan.TotalInterestPaid += interestPortion
	loan.TotalAmountPaid += loan.EMIAmount

	if loan.OutstandingPrincipal > 0 && loan.EMIAmount > 0 {
		loan.RemainingTenureMonths = calculateRemainingTenure(loan.OutstandingPrincipal, loan.InterestRate, loan.EMIAmount)
	} else {
		loan.RemainingTenureMonths = 0
		loan.Status = models.LoanClosed
	}

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *PersonalLoanService) AddPrepayment(ctx context.Context, id string, req models.AddPrepaymentRequest) (*models.PersonalLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	prepayDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	prepay := models.Prepayment{
		PrepaymentID: uuid.New().String(),
		Date:         prepayDate,
		Amount:       req.Amount,
		Type:         req.Type,
		Notes:        req.Notes,
	}

	loan.OutstandingPrincipal -= req.Amount
	if loan.OutstandingPrincipal < 0 {
		loan.OutstandingPrincipal = 0
	}
	loan.TotalPrepayments += req.Amount
	loan.TotalAmountPaid += req.Amount
	loan.TotalPrincipalPaid += req.Amount

	if req.Type == models.PrepaymentForeclosure || loan.OutstandingPrincipal == 0 {
		loan.Status = models.LoanForeclosed
		loan.RemainingTenureMonths = 0
	} else {
		loan.RemainingTenureMonths = calculateRemainingTenure(loan.OutstandingPrincipal, loan.InterestRate, loan.EMIAmount)
		prepay.NewTenure = loan.RemainingTenureMonths
	}

	loan.Prepayments = append(loan.Prepayments, prepay)

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *PersonalLoanService) ChangeRate(ctx context.Context, id string, req models.ChangeRateRequest) (*models.PersonalLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	effectiveDate, err := parseDate(req.EffectiveDate)
	if err != nil {
		return nil, err
	}

	oldRate := loan.InterestRate
	newEMI := calculateEMI(loan.OutstandingPrincipal, req.NewRate, loan.RemainingTenureMonths)

	entry := models.RateChangeEntry{
		EffectiveDate: effectiveDate,
		OldRate:       oldRate,
		NewRate:       req.NewRate,
		NewEMI:        newEMI,
	}

	loan.RateChangeHistory = append(loan.RateChangeHistory, entry)
	loan.InterestRate = req.NewRate
	loan.EMIAmount = newEMI

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *PersonalLoanService) GetAmortizationSchedule(ctx context.Context, id string) ([]models.AmortizationEntry, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	return generateAmortization(loan.OutstandingPrincipal, loan.InterestRate, loan.EMIAmount, loan.RemainingTenureMonths), nil
}
