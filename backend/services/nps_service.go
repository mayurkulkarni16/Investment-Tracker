package services

import (
	"context"
	"fmt"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type NPSService struct {
	repo *repository.NPSRepo
}

func NewNPSService(repo *repository.NPSRepo) *NPSService {
	return &NPSService{repo: repo}
}

func (s *NPSService) Create(ctx context.Context, req models.CreateNPSAccountRequest) (*models.NPSAccount, error) {
	doj, err := parseDate(req.DateOfJoining)
	if err != nil {
		return nil, err
	}

	account := &models.NPSAccount{
		AccountHolderName: req.AccountHolderName,
		PRAN:              req.PRAN,
		AccountType:       req.AccountType,
		FundManager:       req.FundManager,
		DateOfJoining:     doj,
		EquityPct:         req.EquityPct,
		CorporateBondPct:  req.CorporateBondPct,
		GovtSecPct:        req.GovtSecPct,
		AlternatePct:      req.AlternatePct,
		CurrentValue:      req.CurrentValue,
		Contributions:     []models.NPSContribution{},
		Status:            "active",
		Notes:             req.Notes,
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return nil, err
	}
	s.computeDerived(account)
	return account, nil
}

func (s *NPSService) GetAll(ctx context.Context) ([]models.NPSAccount, error) {
	accounts, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		s.computeDerived(&accounts[i])
		accounts[i].XIRR = ComputeNPSXIRR(&accounts[i])
	}
	return accounts, nil
}

func (s *NPSService) GetByID(ctx context.Context, id string) (*models.NPSAccount, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.computeDerived(account)
	account.XIRR = ComputeNPSXIRR(account)
	return account, nil
}

func (s *NPSService) Update(ctx context.Context, id string, req models.UpdateNPSAccountRequest) (*models.NPSAccount, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.AccountHolderName != "" {
		account.AccountHolderName = req.AccountHolderName
	}
	if req.FundManager != "" {
		account.FundManager = req.FundManager
	}
	if req.EquityPct > 0 || req.CorporateBondPct > 0 || req.GovtSecPct > 0 || req.AlternatePct > 0 {
		account.EquityPct = req.EquityPct
		account.CorporateBondPct = req.CorporateBondPct
		account.GovtSecPct = req.GovtSecPct
		account.AlternatePct = req.AlternatePct
	}
	if req.CurrentValue > 0 {
		account.CurrentValue = req.CurrentValue
	}
	if req.Notes != "" {
		account.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, account); err != nil {
		return nil, err
	}
	s.computeDerived(account)
	return account, nil
}

func (s *NPSService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *NPSService) AddContribution(ctx context.Context, id string, req models.AddNPSContributionRequest) (*models.NPSAccount, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	fy := getFY(date)

	entry := models.NPSContribution{
		ContributionID: uuid.New().String(),
		Date:           date,
		Amount:         req.Amount,
		Type:           req.Type,
		FY:             fy,
	}

	account.Contributions = append(account.Contributions, entry)
	if req.Type == "employer" {
		account.TotalEmployerContribution += req.Amount
	} else {
		account.TotalSelfContribution += req.Amount
	}
	account.TotalContribution += req.Amount

	if err := s.repo.Update(ctx, account); err != nil {
		return nil, err
	}
	s.computeDerived(account)
	return account, nil
}

func (s *NPSService) computeDerived(account *models.NPSAccount) {
	now := time.Now()
	fyStart := getCurrentFYStart(now)

	selfThisFY := 0.0
	employerThisFY := 0.0

	for _, c := range account.Contributions {
		if !c.Date.Before(fyStart) {
			if c.Type == "employer" {
				employerThisFY += c.Amount
			} else {
				selfThisFY += c.Amount
			}
		}
	}

	// 80CCD(1): Self contribution (within 80C 1.5L limit) - max 10% of salary, we track actual
	account.Section80CCD1 = selfThisFY
	// 80CCD(1B): Additional deduction up to 50K
	if selfThisFY > 0 {
		account.Section80CCD1B = selfThisFY
		if account.Section80CCD1B > 50000 {
			account.Section80CCD1B = 50000
		}
	}
	// 80CCD(2): Employer contribution (max 14% of basic for govt, 10% for others)
	account.Section80CCD2 = employerThisFY
}

func getFY(date time.Time) string {
	year := date.Year()
	if date.Month() < 4 {
		year--
	}
	return fmt.Sprintf("%d-%02d", year, (year+1)%100)
}
