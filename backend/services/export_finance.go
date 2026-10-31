package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"investment-tracker/repository"
)

type financeExportRepos struct {
	bank      *repository.BankAccountRepo
	gold      *repository.GoldRepo
	property  *repository.PropertyRepo
	insurance *repository.InsuranceRepo
}

// SetPersonalFinanceRepos adds bank accounts, gold, real estate and insurance to CSV exports.
func (s *ExportService) SetPersonalFinanceRepos(bank *repository.BankAccountRepo, gold *repository.GoldRepo, property *repository.PropertyRepo, insurance *repository.InsuranceRepo) {
	s.finance = &financeExportRepos{bank: bank, gold: gold, property: property, insurance: insurance}
}

func dateOrBlank(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func (s *ExportService) exportBankAccounts(ctx context.Context, userID string, w *csv.Writer) {
	w.Write([]string{"Type", "Name", "Institution", "Account Type", "Last Four", "Balance", "Balance As Of", "Emergency Fund", "Status"})
	if s.finance == nil {
		return
	}
	accounts, _ := s.finance.bank.GetAll(ctx, userID)
	for _, a := range accounts {
		w.Write([]string{
			"Bank Account", a.Name, a.Institution, a.AccountType, a.LastFour, fmt.Sprintf("%.2f", a.Balance),
			dateOrBlank(&a.BalanceAsOf), fmt.Sprintf("%t", a.IsEmergencyFund), a.Status,
		})
	}
}

func (s *ExportService) exportGold(ctx context.Context, userID string, w *csv.Writer) {
	w.Write([]string{"Type", "Name", "Form", "Quantity", "Purity", "Purchase Price", "Purchase Date", "Current Price", "Current Value", "Gain/Loss", "CAGR %"})
	if s.finance == nil {
		return
	}
	items, _ := s.finance.gold.GetAll(ctx, userID)
	now := time.Now()
	for i := range items {
		g := &items[i]
		ComputeGold(g, now)
		w.Write([]string{
			"Gold", g.Name, g.Form, fmt.Sprintf("%.4f", g.Quantity), fmt.Sprintf("%.0f", g.Purity),
			fmt.Sprintf("%.2f", g.PurchasePrice), dateOrBlank(&g.PurchaseDate), fmt.Sprintf("%.2f", g.CurrentPrice),
			fmt.Sprintf("%.2f", g.CurrentValue), fmt.Sprintf("%.2f", g.GainLoss), fmt.Sprintf("%.2f", g.CAGR),
		})
	}
}

func (s *ExportService) exportRealEstate(ctx context.Context, userID string, w *csv.Writer) {
	w.Write([]string{"Type", "Name", "Property Type", "Location", "Purchase Price", "Purchase Date", "Current Value", "Ownership %", "Owned Value", "Monthly Rent", "Net Rental Yield %"})
	if s.finance == nil {
		return
	}
	items, _ := s.finance.property.GetAll(ctx, userID)
	now := time.Now()
	for i := range items {
		p := &items[i]
		ComputeProperty(p, nil, now)
		w.Write([]string{
			"Real Estate", p.Name, p.PropertyType, p.Location, fmt.Sprintf("%.2f", p.PurchasePrice), dateOrBlank(&p.PurchaseDate),
			fmt.Sprintf("%.2f", p.CurrentValue), fmt.Sprintf("%.2f", p.OwnershipPct), fmt.Sprintf("%.2f", p.OwnedValue),
			fmt.Sprintf("%.2f", p.MonthlyRent), fmt.Sprintf("%.2f", p.NetRentalYield),
		})
	}
}

func (s *ExportService) exportInsurance(ctx context.Context, userID string, w *csv.Writer) {
	w.Write([]string{"Type", "Policy Type", "Insurer", "Policy Name", "Policy Number", "Sum Assured", "Premium", "Frequency", "Start Date", "End Date", "Current Value", "Nominee", "Status"})
	if s.finance == nil {
		return
	}
	items, _ := s.finance.insurance.GetAll(ctx, userID)
	for _, p := range items {
		w.Write([]string{
			"Insurance", p.PolicyType, p.Insurer, p.PolicyName, p.PolicyNumber, fmt.Sprintf("%.2f", p.SumAssured),
			fmt.Sprintf("%.2f", p.PremiumAmount), p.PremiumFrequency, dateOrBlank(&p.StartDate), dateOrBlank(p.EndDate),
			fmt.Sprintf("%.2f", p.CurrentValue), p.Nominee, p.Status,
		})
	}
}

func (s *ExportService) exportFinance(ctx context.Context, userID string, w *csv.Writer) {
	if s.finance == nil {
		return
	}
	sections := []struct {
		title string
		fn    func(context.Context, string, *csv.Writer)
	}{
		{"BANK & CASH", s.exportBankAccounts}, {"GOLD", s.exportGold}, {"REAL ESTATE", s.exportRealEstate}, {"INSURANCE", s.exportInsurance},
	}
	for _, sec := range sections {
		w.Write([]string{""})
		w.Write([]string{"--- " + sec.title + " ---"})
		sec.fn(ctx, userID, w)
	}
}
