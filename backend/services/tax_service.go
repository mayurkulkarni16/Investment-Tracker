package services

import (
	"context"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type TaxService struct {
	mfRepo           *repository.MutualFundRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	npsRepo          *repository.NPSRepo
	bondRepo         *repository.CorporateBondRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	stockRepo        *repository.StockRepo
}

func NewTaxService(
	mfRepo *repository.MutualFundRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	npsRepo *repository.NPSRepo,
	bondRepo *repository.CorporateBondRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	stockRepo *repository.StockRepo,
) *TaxService {
	return &TaxService{
		mfRepo: mfRepo, fdRepo: fdRepo, pfRepo: pfRepo, npsRepo: npsRepo,
		bondRepo: bondRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		stockRepo: stockRepo,
	}
}

func (s *TaxService) GetTaxSummary(ctx context.Context, fy string) (*models.TaxSummary, error) {
	now := time.Now()
	fyStart := getCurrentFYStart(now)

	summary := &models.TaxSummary{
		FY: fy,
		Section80C: models.Section80CBreakdown{
			Limit: 150000,
		},
		Section24b: models.Section24bBreakdown{
			Limit: 200000,
		},
	}

	// 80C: EPF Employee contribution
	pfs, _ := s.pfRepo.GetAll(ctx)
	for _, pf := range pfs {
		for _, fyEntry := range pf.FinancialYearEntries {
			for _, mc := range fyEntry.MonthlyContributions {
				// Parse month string "Apr-2025" or use FY match
				summary.Section80C.EPFContribution += mc.EmployeeContribution
			}
		}
	}

	// 80C: ELSS investments
	mfs, _ := s.mfRepo.GetAll(ctx)
	for _, mf := range mfs {
		if mf.IsELSS {
			for _, txn := range mf.Transactions {
				if !txn.Date.Before(fyStart) && txn.Type == "purchase" {
					summary.Section80C.ELSSInvestment += txn.Amount
				}
			}
		}
	}

	// 80C: Home loan principal
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx)
	for _, loan := range homeLoans {
		for _, emi := range loan.EMIsPaid {
			if emi.PaidDate != nil && !emi.PaidDate.Before(fyStart) {
				summary.Section80C.HomeLoanPrincipal += emi.PrincipalPortion
			}
		}
	}

	summary.Section80C.Total = summary.Section80C.EPFContribution +
		summary.Section80C.ELSSInvestment +
		summary.Section80C.HomeLoanPrincipal +
		summary.Section80C.LifeInsurance
	summary.Section80C.Deduction = math.Min(summary.Section80C.Total, summary.Section80C.Limit)

	// Section 24(b): Home loan interest
	for _, loan := range homeLoans {
		for _, emi := range loan.EMIsPaid {
			if emi.PaidDate != nil && !emi.PaidDate.Before(fyStart) {
				summary.Section24b.HomeLoanInterest += emi.InterestPortion
			}
		}
		for _, pe := range loan.PreEMIsPaid {
			if pe.PaidDate != nil && !pe.PaidDate.Before(fyStart) {
				summary.Section24b.PreEMIInterest += pe.InterestAmount
			}
		}
	}
	summary.Section24b.Total = summary.Section24b.HomeLoanInterest + summary.Section24b.PreEMIInterest
	summary.Section24b.Deduction = math.Min(summary.Section24b.Total, summary.Section24b.Limit)

	// Section 80CCD: NPS
	npsAccounts, _ := s.npsRepo.GetAll(ctx)
	for _, nps := range npsAccounts {
		for _, c := range nps.Contributions {
			if !c.Date.Before(fyStart) {
				if c.Type == "employer" {
					summary.Section80CCD.NPS80CCD2 += c.Amount
				} else {
					summary.Section80CCD.NPS80CCD1 += c.Amount
				}
			}
		}
	}
	if summary.Section80CCD.NPS80CCD1 > 50000 {
		summary.Section80CCD.NPS80CCD1B = 50000
	} else {
		summary.Section80CCD.NPS80CCD1B = summary.Section80CCD.NPS80CCD1
	}

	// Interest income from FDs and Bonds
	fds, _ := s.fdRepo.GetAll(ctx)
	for _, fd := range fds {
		if fd.InterestType == "monthly" || fd.InterestType == "quarterly" {
			// Estimate interest for FY
			monthsInFY := 12.0
			if fd.Status == "active" {
				summary.InterestIncome += fd.PrincipalAmount * fd.InterestRate / 100 / 12 * monthsInFY
			}
		}
	}

	bonds, _ := s.bondRepo.GetAll(ctx)
	for _, bond := range bonds {
		for _, p := range bond.InterestPayouts {
			if p.Status == "received" && !p.ScheduledDate.Before(fyStart) {
				summary.InterestIncome += p.Amount
			}
		}
	}

	// Total deductions
	summary.TotalDeductions = summary.Section80C.Deduction +
		summary.Section80CCD.NPS80CCD1B +
		summary.Section80CCD.NPS80CCD2 +
		summary.Section24b.Deduction

	return summary, nil
}

// GetCapitalGains calculates capital gains from MF redemptions and stock sales
func (s *TaxService) GetCapitalGains(ctx context.Context, fy string) (*models.CapitalGainsSummary, error) {
	now := time.Now()
	fyStart := getCurrentFYStart(now)

	cg := &models.CapitalGainsSummary{
		Entries:       []models.CapitalGainEntry{},
		LTCGExemption: 125000, // ₹1.25L for equity
	}

	// MF redemptions
	mfs, _ := s.mfRepo.GetAll(ctx)
	for _, mf := range mfs {
		for _, txn := range mf.Transactions {
			if txn.Type == "redeem" || txn.Type == "redemption" && !txn.Date.Before(fyStart) {
				// Simplified: estimate buy price from average
				avgBuyPrice := 0.0
				if mf.TotalUnits > 0 {
					avgBuyPrice = mf.TotalInvested / mf.TotalUnits
				}
				buyAmount := math.Abs(txn.Units) * avgBuyPrice
				sellAmount := txn.Amount
				gain := sellAmount - buyAmount

				// Holding period: simplified, using fund type
				isLongTerm := false
				taxRate := 20.0 // STCG for equity MF
				if mf.FundType == "equity" || mf.FundType == "hybrid" {
					// Equity MF: > 1 year = LTCG
					isLongTerm = true // simplified
					taxRate = 12.5    // LTCG equity
				} else {
					// Debt MF: taxed at slab rate (simplified as 30%)
					taxRate = 30.0
				}

				entry := models.CapitalGainEntry{
					InvestmentType: "mutual_fund",
					InvestmentName: mf.FundName,
					SellDate:       txn.Date.Format("2006-01-02"),
					SellAmount:     sellAmount,
					BuyAmount:      buyAmount,
					Gain:           gain,
					IsLongTerm:     isLongTerm,
					TaxRate:        taxRate,
				}

				if gain > 0 {
					entry.TaxLiability = math.Round(gain*taxRate/100*100) / 100
				}

				cg.Entries = append(cg.Entries, entry)

				if isLongTerm {
					cg.LTCG += gain
				} else {
					cg.STCG += gain
				}
			}
		}
	}

	// Stock sales
	stocks, _ := s.stockRepo.GetAll(ctx)
	for _, stock := range stocks {
		for _, txn := range stock.Transactions {
			if txn.Type == "sell" && !txn.Date.Before(fyStart) {
				gain := (txn.PricePerShare - stock.AvgBuyPrice) * float64(txn.Quantity)
				isLongTerm := true // simplified for stocks > 1 year
				taxRate := 12.5    // LTCG equity

				entry := models.CapitalGainEntry{
					InvestmentType: "stock",
					InvestmentName: stock.Symbol,
					SellDate:       txn.Date.Format("2006-01-02"),
					SellAmount:     txn.PricePerShare * float64(txn.Quantity),
					BuyAmount:      stock.AvgBuyPrice * float64(txn.Quantity),
					Gain:           gain,
					IsLongTerm:     isLongTerm,
					TaxRate:        taxRate,
				}

				if gain > 0 {
					entry.TaxLiability = math.Round(gain*taxRate/100*100) / 100
				}

				cg.Entries = append(cg.Entries, entry)

				if isLongTerm {
					cg.LTCG += gain
				} else {
					cg.STCG += gain
				}
			}
		}
	}

	// Apply LTCG exemption
	ltcgTaxable := cg.LTCG - cg.LTCGExemption
	if ltcgTaxable < 0 {
		ltcgTaxable = 0
	}
	cg.LTCGTax = math.Round(ltcgTaxable*12.5/100*100) / 100

	if cg.STCG > 0 {
		cg.STCGTax = math.Round(cg.STCG*20/100*100) / 100
	}

	cg.TotalTax = cg.LTCGTax + cg.STCGTax

	return cg, nil
}
