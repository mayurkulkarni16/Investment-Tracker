package services

import (
	"context"
	"time"

	"investment-tracker/repository"
)

type DashboardService struct {
	mfRepo           *repository.MutualFundRepo
	bondRepo         *repository.CorporateBondRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	stockRepo        *repository.StockRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
}

func NewDashboardService(
	mfRepo *repository.MutualFundRepo,
	bondRepo *repository.CorporateBondRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	stockRepo *repository.StockRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
) *DashboardService {
	return &DashboardService{
		mfRepo:           mfRepo,
		bondRepo:         bondRepo,
		fdRepo:           fdRepo,
		pfRepo:           pfRepo,
		stockRepo:        stockRepo,
		homeLoanRepo:     homeLoanRepo,
		personalLoanRepo: personalLoanRepo,
	}
}

type DashboardData struct {
	TotalInvested       float64            `json:"total_invested"`
	CurrentValue        float64            `json:"current_value"`
	TotalGains          float64            `json:"total_gains"`
	OverallReturnPct    float64            `json:"overall_return_percent"`
	ELSSTaxSaving       float64            `json:"elss_tax_saving"`
	AssetAllocation     map[string]float64 `json:"asset_allocation"`
	UpcomingPayouts     []UpcomingPayout   `json:"upcoming_payouts"`
	MFSummary           CategorySummary    `json:"mutual_fund_summary"`
	BondSummary         CategorySummary    `json:"corporate_bond_summary"`
	FDSummary           CategorySummary    `json:"fixed_deposit_summary"`
	PFSummary           CategorySummary    `json:"provident_fund_summary"`
	StockSummary        CategorySummary    `json:"stock_summary"`
	HomeLoanSummary     LoanSummary        `json:"home_loan_summary"`
	PersonalLoanSummary LoanSummary        `json:"personal_loan_summary"`
}

type CategorySummary struct {
	TotalInvested float64 `json:"total_invested"`
	CurrentValue  float64 `json:"current_value"`
	Count         int     `json:"count"`
}

type LoanSummary struct {
	TotalDisbursed    float64 `json:"total_disbursed"`
	TotalOutstanding  float64 `json:"total_outstanding"`
	TotalInterestPaid float64 `json:"total_interest_paid"`
	TotalPrepayments  float64 `json:"total_prepayments"`
	MonthlyEMI        float64 `json:"monthly_emi"`
	Count             int     `json:"count"`
}

type UpcomingPayout struct {
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	Date       string  `json:"date"`
	Amount     float64 `json:"amount"`
	PayoutType string  `json:"payout_type"`
}

func (s *DashboardService) GetDashboard(ctx context.Context) (*DashboardData, error) {
	dashboard := &DashboardData{
		AssetAllocation: make(map[string]float64),
		UpcomingPayouts: []UpcomingPayout{},
	}

	now := time.Now()
	thirtyDaysLater := now.AddDate(0, 0, 30)
	currentFYStart := getCurrentFYStart(now)

	// Mutual Funds
	funds, err := s.mfRepo.GetAll(ctx)
	if err == nil {
		for _, mf := range funds {
			dashboard.MFSummary.TotalInvested += mf.TotalInvested
			dashboard.MFSummary.CurrentValue += mf.CurrentValue
			dashboard.MFSummary.Count++

			if mf.IsELSS {
				for _, txn := range mf.Transactions {
					if !txn.Date.Before(currentFYStart) {
						dashboard.ELSSTaxSaving += txn.Amount
					}
				}
			}
		}
	}

	// Corporate Bonds
	bonds, err := s.bondRepo.GetAll(ctx)
	if err == nil {
		for _, bond := range bonds {
			dashboard.BondSummary.TotalInvested += bond.InvestmentAmount
			dashboard.BondSummary.CurrentValue += bond.RemainingPrincipal
			dashboard.BondSummary.Count++

			for _, p := range bond.InterestPayouts {
				if p.Status == "pending" && !p.ScheduledDate.Before(now) && !p.ScheduledDate.After(thirtyDaysLater) {
					dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
						Type:       "Corporate Bond",
						Name:       bond.BondName,
						Date:       p.ScheduledDate.Format("2006-01-02"),
						Amount:     p.Amount,
						PayoutType: "interest",
					})
				}
			}
			for _, p := range bond.PrincipalRepayments {
				if p.Status == "pending" && !p.ScheduledDate.Before(now) && !p.ScheduledDate.After(thirtyDaysLater) {
					dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
						Type:       "Corporate Bond",
						Name:       bond.BondName,
						Date:       p.ScheduledDate.Format("2006-01-02"),
						Amount:     p.Amount,
						PayoutType: "principal",
					})
				}
			}
		}
	}

	// Fixed Deposits
	fds, err := s.fdRepo.GetAll(ctx)
	if err == nil {
		for _, fd := range fds {
			dashboard.FDSummary.TotalInvested += fd.PrincipalAmount
			if fd.InterestType == "cumulative" {
				dashboard.FDSummary.CurrentValue += fd.MaturityAmount
			} else {
				dashboard.FDSummary.CurrentValue += fd.PrincipalAmount
			}
			dashboard.FDSummary.Count++

			if fd.Status == "active" && !fd.MaturityDate.Before(now) && !fd.MaturityDate.After(thirtyDaysLater) {
				dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
					Type:       "Fixed Deposit",
					Name:       fd.BankName + " - " + fd.FDNumber,
					Date:       fd.MaturityDate.Format("2006-01-02"),
					Amount:     fd.MaturityAmount,
					PayoutType: "maturity",
				})
			}
		}
	}

	// Provident Fund
	pfs, err := s.pfRepo.GetAll(ctx)
	if err == nil {
		for _, pf := range pfs {
			totalContrib := pf.TotalEmployeeContribution + pf.TotalEmployerContribution
			dashboard.PFSummary.TotalInvested += totalContrib
			dashboard.PFSummary.CurrentValue += pf.CurrentBalance
			dashboard.PFSummary.Count++
		}
	}

	// Stocks
	stocks, err := s.stockRepo.GetAll(ctx)
	if err == nil {
		for _, stock := range stocks {
			dashboard.StockSummary.TotalInvested += stock.TotalInvested
			dashboard.StockSummary.CurrentValue += stock.CurrentValue
			dashboard.StockSummary.Count++
		}
	}

	// Home Loans
	loans, err := s.homeLoanRepo.GetAll(ctx)
	if err == nil {
		for _, loan := range loans {
			if loan.Status == "active" {
				dashboard.HomeLoanSummary.TotalDisbursed += loan.DisbursedAmount
				dashboard.HomeLoanSummary.TotalOutstanding += loan.OutstandingPrincipal
				dashboard.HomeLoanSummary.TotalInterestPaid += loan.TotalInterestPaid
				dashboard.HomeLoanSummary.TotalPrepayments += loan.TotalPrepayments
				dashboard.HomeLoanSummary.MonthlyEMI += loan.EMIAmount
				dashboard.HomeLoanSummary.Count++

				// Add upcoming EMI as upcoming payout
				nextEMIDate := loan.EMIStartDate
				if len(loan.EMIsPaid) > 0 {
					lastPaid := loan.EMIsPaid[len(loan.EMIsPaid)-1]
					if lastPaid.PaidDate != nil {
						nextEMIDate = lastPaid.DueDate.AddDate(0, 1, 0)
					}
				}
				if !nextEMIDate.Before(now) && !nextEMIDate.After(thirtyDaysLater) {
					dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
						Type:       "Home Loan",
						Name:       loan.BankName + " - EMI",
						Date:       nextEMIDate.Format("2006-01-02"),
						Amount:     loan.EMIAmount,
						PayoutType: "emi",
					})
				}
			}
		}
	}

	// Personal Loans
	pLoans, err := s.personalLoanRepo.GetAll(ctx)
	if err == nil {
		for _, pl := range pLoans {
			if pl.Status == "active" {
				dashboard.PersonalLoanSummary.TotalDisbursed += pl.DisbursedAmount
				dashboard.PersonalLoanSummary.TotalOutstanding += pl.OutstandingPrincipal
				dashboard.PersonalLoanSummary.TotalInterestPaid += pl.TotalInterestPaid
				dashboard.PersonalLoanSummary.TotalPrepayments += pl.TotalPrepayments
				dashboard.PersonalLoanSummary.MonthlyEMI += pl.EMIAmount
				dashboard.PersonalLoanSummary.Count++

				nextEMIDate := pl.EMIStartDate
				if len(pl.EMIsPaid) > 0 {
					lastPaid := pl.EMIsPaid[len(pl.EMIsPaid)-1]
					if lastPaid.PaidDate != nil {
						nextEMIDate = lastPaid.DueDate.AddDate(0, 1, 0)
					}
				}
				if !nextEMIDate.Before(now) && !nextEMIDate.After(thirtyDaysLater) {
					dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
						Type:       "Personal Loan",
						Name:       pl.LenderName + " - EMI",
						Date:       nextEMIDate.Format("2006-01-02"),
						Amount:     pl.EMIAmount,
						PayoutType: "emi",
					})
				}
			}
		}
	}

	// Totals
	dashboard.TotalInvested = dashboard.MFSummary.TotalInvested +
		dashboard.BondSummary.TotalInvested +
		dashboard.FDSummary.TotalInvested +
		dashboard.PFSummary.TotalInvested +
		dashboard.StockSummary.TotalInvested

	dashboard.CurrentValue = dashboard.MFSummary.CurrentValue +
		dashboard.BondSummary.CurrentValue +
		dashboard.FDSummary.CurrentValue +
		dashboard.PFSummary.CurrentValue +
		dashboard.StockSummary.CurrentValue

	dashboard.TotalGains = dashboard.CurrentValue - dashboard.TotalInvested
	if dashboard.TotalInvested > 0 {
		dashboard.OverallReturnPct = (dashboard.TotalGains / dashboard.TotalInvested) * 100
	}

	// Cap ELSS at 1.5L
	if dashboard.ELSSTaxSaving > 150000 {
		dashboard.ELSSTaxSaving = 150000
	}

	// Asset allocation (actual amounts — frontend computes percentages)
	if dashboard.CurrentValue > 0 {
		dashboard.AssetAllocation["Mutual Funds"] = dashboard.MFSummary.CurrentValue
		dashboard.AssetAllocation["Corporate Bonds"] = dashboard.BondSummary.CurrentValue
		dashboard.AssetAllocation["Fixed Deposits"] = dashboard.FDSummary.CurrentValue
		dashboard.AssetAllocation["Provident Fund"] = dashboard.PFSummary.CurrentValue
		dashboard.AssetAllocation["Stocks"] = dashboard.StockSummary.CurrentValue
	}

	return dashboard, nil
}

func getCurrentFYStart(now time.Time) time.Time {
	year := now.Year()
	if now.Month() < 4 {
		year--
	}
	return time.Date(year, 4, 1, 0, 0, 0, 0, time.Local)
}
