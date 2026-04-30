package services

import (
	"context"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type ProvidentFundService struct {
	repo *repository.ProvidentFundRepo
}

func NewProvidentFundService(repo *repository.ProvidentFundRepo) *ProvidentFundService {
	return &ProvidentFundService{repo: repo}
}

func (s *ProvidentFundService) Create(ctx context.Context, req models.CreateProvidentFundRequest) (*models.ProvidentFund, error) {
	pf := &models.ProvidentFund{
		AccountType:          req.AccountType,
		AccountNumber:        req.AccountNumber,
		EmployerName:         req.EmployerName,
		InterestRate:         req.InterestRate,
		FinancialYearEntries: []models.FinancialYearEntry{},
		Notes:                req.Notes,
	}

	if err := s.repo.Create(ctx, pf); err != nil {
		return nil, err
	}
	return pf, nil
}

func (s *ProvidentFundService) GetAll(ctx context.Context) ([]models.ProvidentFund, error) {
	pfs, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range pfs {
		pfs[i].XIRR = ComputeProvidentFundXIRR(&pfs[i])
	}
	return pfs, nil
}

func (s *ProvidentFundService) GetByID(ctx context.Context, id string) (*models.ProvidentFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	pf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	pf.XIRR = ComputeProvidentFundXIRR(pf)
	return pf, nil
}

func (s *ProvidentFundService) AddContribution(ctx context.Context, id string, req models.AddMonthlyContributionRequest) (*models.ProvidentFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	pf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	contribution := models.MonthlyContribution{
		Month:                req.Month,
		EmployeeContribution: req.EmployeeContribution,
		EmployerContribution: req.EmployerContribution,
		Total:                req.EmployeeContribution + req.EmployerContribution,
	}

	// Find or create financial year entry
	fyFound := false
	for i := range pf.FinancialYearEntries {
		if pf.FinancialYearEntries[i].FinancialYear == req.FinancialYear {
			pf.FinancialYearEntries[i].MonthlyContributions = append(
				pf.FinancialYearEntries[i].MonthlyContributions, contribution,
			)
			fyFound = true
			break
		}
	}

	if !fyFound {
		fyEntry := models.FinancialYearEntry{
			FinancialYear:        req.FinancialYear,
			MonthlyContributions: []models.MonthlyContribution{contribution},
			OpeningBalance:       pf.CurrentBalance,
		}
		pf.FinancialYearEntries = append(pf.FinancialYearEntries, fyEntry)
	}

	pf.TotalEmployeeContribution += req.EmployeeContribution
	pf.TotalEmployerContribution += req.EmployerContribution
	pf.CurrentBalance += contribution.Total

	if err := s.repo.Update(ctx, pf); err != nil {
		return nil, err
	}
	return pf, nil
}

func (s *ProvidentFundService) Update(ctx context.Context, id string, req models.CreateProvidentFundRequest) (*models.ProvidentFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	pf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	pf.AccountType = req.AccountType
	pf.AccountNumber = req.AccountNumber
	pf.EmployerName = req.EmployerName
	pf.InterestRate = req.InterestRate
	pf.Notes = req.Notes

	if err := s.repo.Update(ctx, pf); err != nil {
		return nil, err
	}
	return pf, nil
}

func (s *ProvidentFundService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *ProvidentFundService) ImportFromPDF(ctx context.Context, id string, req models.ImportPFRequest) (*models.ImportPFResult, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	pf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	result := &models.ImportPFResult{}

	for _, year := range req.Years {
		// Find or create financial year entry
		fyIdx := -1
		for i := range pf.FinancialYearEntries {
			if pf.FinancialYearEntries[i].FinancialYear == year.FinancialYear {
				fyIdx = i
				break
			}
		}

		if fyIdx == -1 {
			fyEntry := models.FinancialYearEntry{
				FinancialYear:        year.FinancialYear,
				MonthlyContributions: []models.MonthlyContribution{},
				OpeningBalance:       year.OpeningBalance,
				InterestEarned:       year.InterestEarned,
				ClosingBalance:       year.ClosingBalance,
			}
			pf.FinancialYearEntries = append(pf.FinancialYearEntries, fyEntry)
			fyIdx = len(pf.FinancialYearEntries) - 1
			result.YearsAdded++
		} else {
			// Update balances if provided
			if year.OpeningBalance > 0 {
				pf.FinancialYearEntries[fyIdx].OpeningBalance = year.OpeningBalance
			}
			if year.InterestEarned > 0 {
				pf.FinancialYearEntries[fyIdx].InterestEarned = year.InterestEarned
			}
			if year.ClosingBalance > 0 {
				pf.FinancialYearEntries[fyIdx].ClosingBalance = year.ClosingBalance
			}
		}

		for _, contrib := range year.Contributions {
			// Check for duplicate month
			isDuplicate := false
			for _, existing := range pf.FinancialYearEntries[fyIdx].MonthlyContributions {
				if existing.Month == contrib.Month {
					isDuplicate = true
					break
				}
			}
			if isDuplicate {
				continue
			}

			mc := models.MonthlyContribution{
				Month:                contrib.Month,
				EmployeeContribution: contrib.EmployeeContribution,
				EmployerContribution: contrib.EmployerContribution,
				Total:                contrib.EmployeeContribution + contrib.EmployerContribution,
			}
			pf.FinancialYearEntries[fyIdx].MonthlyContributions = append(
				pf.FinancialYearEntries[fyIdx].MonthlyContributions, mc,
			)

			pf.TotalEmployeeContribution += contrib.EmployeeContribution
			pf.TotalEmployerContribution += contrib.EmployerContribution
			pf.CurrentBalance += mc.Total
			result.ContributionsAdded++
		}
	}

	// Update closing balance from latest year if available
	if len(req.Years) > 0 {
		lastYear := req.Years[len(req.Years)-1]
		if lastYear.ClosingBalance > 0 {
			pf.CurrentBalance = lastYear.ClosingBalance
		}
		pf.TotalInterestEarned = 0
		for _, fy := range pf.FinancialYearEntries {
			pf.TotalInterestEarned += fy.InterestEarned
		}
	}

	if err := s.repo.Update(ctx, pf); err != nil {
		return nil, err
	}

	return result, nil
}
