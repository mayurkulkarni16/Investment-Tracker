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
	npsRepo          *repository.NPSRepo
	creditCardRepo   *repository.CreditCardRepo
}

func NewDashboardService(
	mfRepo *repository.MutualFundRepo,
	bondRepo *repository.CorporateBondRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	stockRepo *repository.StockRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	npsRepo *repository.NPSRepo,
	creditCardRepo *repository.CreditCardRepo,
) *DashboardService {
	return &DashboardService{
		mfRepo:           mfRepo,
		bondRepo:         bondRepo,
		fdRepo:           fdRepo,
		pfRepo:           pfRepo,
		stockRepo:        stockRepo,
		homeLoanRepo:     homeLoanRepo,
		personalLoanRepo: personalLoanRepo,
		npsRepo:          npsRepo,
		creditCardRepo:   creditCardRepo,
	}
}

type DashboardData struct {
	TotalInvested       float64            `json:"total_invested"`
	CurrentValue        float64            `json:"current_value"`
	TotalGains          float64            `json:"total_gains"`
	OverallReturnPct    float64            `json:"overall_return_percent"`
	PortfolioXIRR       float64            `json:"portfolio_xirr"`
	ELSSTaxSaving       float64            `json:"elss_tax_saving"`
	AssetAllocation     map[string]float64 `json:"asset_allocation"`
	UpcomingPayouts     []UpcomingPayout   `json:"upcoming_payouts"`
	MFSummary           CategorySummary    `json:"mutual_fund_summary"`
	BondSummary         CategorySummary    `json:"corporate_bond_summary"`
	FDSummary           CategorySummary    `json:"fixed_deposit_summary"`
	PFSummary           CategorySummary    `json:"provident_fund_summary"`
	StockSummary        CategorySummary    `json:"stock_summary"`
	NPSSummary          CategorySummary    `json:"nps_summary"`
	HomeLoanSummary     LoanSummary        `json:"home_loan_summary"`
	PersonalLoanSummary LoanSummary        `json:"personal_loan_summary"`
	CreditCardSummary   CreditCardSummary  `json:"credit_card_summary"`
	RiskMetrics         RiskMetrics        `json:"risk_metrics"`
	DataSources         map[string]string  `json:"data_sources,omitempty"`
}

type CategorySummary struct {
	TotalInvested float64 `json:"total_invested"`
	CurrentValue  float64 `json:"current_value"`
	XIRR          float64 `json:"xirr"`
	Count         int     `json:"count"`
}

type RiskMetrics struct {
	EquityDebtRatio   float64 `json:"equity_debt_ratio"`
	RiskScore         int     `json:"risk_score"`
	Diversification   float64 `json:"diversification"`
	LoanToAssetRatio  float64 `json:"loan_to_asset_ratio"`
	ConcentrationRisk string  `json:"concentration_risk"`
}

type LoanSummary struct {
	TotalDisbursed    float64 `json:"total_disbursed"`
	TotalOutstanding  float64 `json:"total_outstanding"`
	TotalInterestPaid float64 `json:"total_interest_paid"`
	TotalPrepayments  float64 `json:"total_prepayments"`
	MonthlyEMI        float64 `json:"monthly_emi"`
	Count             int     `json:"count"`
}

type CreditCardSummary struct {
	TotalOutstanding float64 `json:"total_outstanding"`
	TotalLimit       float64 `json:"total_limit"`
	AvgUtilization   float64 `json:"avg_utilization"`
	Count            int     `json:"count"`
}

type UpcomingPayout struct {
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	Date       string  `json:"date"`
	Amount     float64 `json:"amount"`
	PayoutType string  `json:"payout_type"`
}

func (s *DashboardService) GetDashboard(ctx context.Context, userID string) (*DashboardData, error) {
	dashboard := &DashboardData{
		AssetAllocation: make(map[string]float64),
		UpcomingPayouts: []UpcomingPayout{},
	}

	now := time.Now()
	thirtyDaysLater := now.AddDate(0, 0, 30)
	currentFYStart := getCurrentFYStart(now)

	// Mutual Funds
	funds, err := s.mfRepo.GetAll(ctx, userID)
	var allCashflows []Cashflow
	var equityValue, debtValue float64
	if err == nil {
		for _, mf := range funds {
			dashboard.MFSummary.TotalInvested += mf.TotalInvested
			dashboard.MFSummary.CurrentValue += mf.CurrentValue
			dashboard.MFSummary.Count++

			// Classify equity vs debt
			switch mf.FundType {
			case "Equity", "ELSS", "Index", "Small Cap", "Mid Cap", "Large Cap", "Multi Cap", "Flexi Cap", "Sectoral", "Thematic":
				equityValue += mf.CurrentValue
			case "Debt", "Liquid", "Gilt", "Corporate Bond", "Dynamic Bond", "Overnight", "Money Market":
				debtValue += mf.CurrentValue
			case "Hybrid":
				equityValue += mf.CurrentValue * 0.65
				debtValue += mf.CurrentValue * 0.35
			}

			// Collect cashflows for portfolio XIRR
			for _, txn := range mf.Transactions {
				switch txn.Type {
				case "purchase", "sip", "switch_in":
					allCashflows = append(allCashflows, Cashflow{Date: txn.Date, Amount: -txn.Amount})
				case "redemption", "switch_out":
					allCashflows = append(allCashflows, Cashflow{Date: txn.Date, Amount: txn.Amount})
				}

				if mf.IsELSS && !txn.Date.Before(currentFYStart) {
					dashboard.ELSSTaxSaving += txn.Amount
				}
			}
		}
	}

	// Corporate Bonds
	bonds, err := s.bondRepo.GetAll(ctx, userID)
	if err == nil {
		for _, bond := range bonds {
			dashboard.BondSummary.TotalInvested += bond.InvestmentAmount
			dashboard.BondSummary.CurrentValue += bond.RemainingPrincipal
			dashboard.BondSummary.Count++
			debtValue += bond.RemainingPrincipal

			// Collect cashflows for portfolio XIRR
			allCashflows = append(allCashflows, Cashflow{Date: bond.PurchaseDate, Amount: -bond.InvestmentAmount})
			for _, p := range bond.InterestPayouts {
				if p.Status == "received" && p.ReceivedDate != nil {
					allCashflows = append(allCashflows, Cashflow{Date: *p.ReceivedDate, Amount: p.Amount})
				}
			}
			for _, p := range bond.PrincipalRepayments {
				if p.Status == "received" && p.ReceivedDate != nil {
					allCashflows = append(allCashflows, Cashflow{Date: *p.ReceivedDate, Amount: p.Amount})
				}
			}

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
	fds, err := s.fdRepo.GetAll(ctx, userID)
	if err == nil {
		for _, fd := range fds {
			dashboard.FDSummary.TotalInvested += fd.PrincipalAmount
			if fd.InterestType == "cumulative" {
				dashboard.FDSummary.CurrentValue += fd.MaturityAmount
			} else {
				dashboard.FDSummary.CurrentValue += fd.PrincipalAmount
			}
			dashboard.FDSummary.Count++
			debtValue += fd.PrincipalAmount

			// Cashflows for portfolio XIRR
			allCashflows = append(allCashflows, Cashflow{Date: fd.StartDate, Amount: -fd.PrincipalAmount})

			if fd.Status == "active" && !fd.MaturityDate.Before(now) && !fd.MaturityDate.After(thirtyDaysLater) {
				dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
					Type:       "Fixed Deposit",
					Name:       fd.BankName + " - " + fd.FDNumber,
					Date:       fd.MaturityDate.Format("2006-01-02"),
					Amount:     fd.MaturityAmount,
					PayoutType: "maturity",
				})
			}

			// Non-cumulative FD interest payouts
			for _, p := range fd.InterestPayouts {
				if p.Status == "pending" && !p.ScheduledDate.Before(now) && !p.ScheduledDate.After(thirtyDaysLater) {
					dashboard.UpcomingPayouts = append(dashboard.UpcomingPayouts, UpcomingPayout{
						Type:       "Fixed Deposit",
						Name:       fd.BankName + " - " + fd.FDNumber,
						Date:       p.ScheduledDate.Format("2006-01-02"),
						Amount:     p.Amount,
						PayoutType: "interest",
					})
				}
			}
		}
	}

	// Provident Fund
	pfs, err := s.pfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, pf := range pfs {
			totalContrib := pf.TotalEmployeeContribution + pf.TotalEmployerContribution
			dashboard.PFSummary.TotalInvested += totalContrib
			dashboard.PFSummary.CurrentValue += pf.CurrentBalance
			dashboard.PFSummary.Count++
			debtValue += pf.CurrentBalance

			// Cashflows for portfolio XIRR
			for _, fy := range pf.FinancialYearEntries {
				for _, mc := range fy.MonthlyContributions {
					date := parsePFMonth(mc.Month, fy.FinancialYear)
					if !date.IsZero() {
						total := mc.EmployeeContribution + mc.EmployerContribution
						if total > 0 {
							allCashflows = append(allCashflows, Cashflow{Date: date, Amount: -total})
						}
					}
				}
			}
		}
	}

	// Stocks
	stocks, err := s.stockRepo.GetAll(ctx, userID)
	if err == nil {
		for _, stock := range stocks {
			dashboard.StockSummary.TotalInvested += stock.TotalInvested
			dashboard.StockSummary.CurrentValue += stock.CurrentValue
			dashboard.StockSummary.Count++
			equityValue += stock.CurrentValue

			// Cashflows for portfolio XIRR
			for _, txn := range stock.Transactions {
				switch txn.Type {
				case "buy":
					allCashflows = append(allCashflows, Cashflow{Date: txn.Date, Amount: -txn.Amount})
				case "sell":
					allCashflows = append(allCashflows, Cashflow{Date: txn.Date, Amount: txn.Amount})
				}
			}
		}
	}

	// NPS
	npsAccounts, err := s.npsRepo.GetAll(ctx, userID)
	if err == nil {
		for _, nps := range npsAccounts {
			dashboard.NPSSummary.TotalInvested += nps.TotalContribution
			dashboard.NPSSummary.CurrentValue += nps.CurrentValue
			dashboard.NPSSummary.Count++

			// Classify equity vs debt based on NPS allocation
			npsEquity := nps.CurrentValue * nps.EquityPct / 100
			npsDebt := nps.CurrentValue - npsEquity
			equityValue += npsEquity
			debtValue += npsDebt

			// Cashflows for portfolio XIRR
			for _, c := range nps.Contributions {
				allCashflows = append(allCashflows, Cashflow{Date: c.Date, Amount: -c.Amount})
			}
		}
	}

	// Home Loans
	loans, err := s.homeLoanRepo.GetAll(ctx, userID)
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
	pLoans, err := s.personalLoanRepo.GetAll(ctx, userID)
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

	// Credit Cards (liabilities, not assets)
	creditCards, err := s.creditCardRepo.GetAll(ctx, userID)
	if err == nil {
		for _, cc := range creditCards {
			dashboard.CreditCardSummary.TotalOutstanding += cc.CurrentOutstanding
			dashboard.CreditCardSummary.TotalLimit += cc.CreditLimit
			dashboard.CreditCardSummary.Count++
		}
		if dashboard.CreditCardSummary.TotalLimit > 0 {
			dashboard.CreditCardSummary.AvgUtilization = dashboard.CreditCardSummary.TotalOutstanding / dashboard.CreditCardSummary.TotalLimit * 100
		}
	}

	// Totals
	dashboard.TotalInvested = dashboard.MFSummary.TotalInvested +
		dashboard.BondSummary.TotalInvested +
		dashboard.FDSummary.TotalInvested +
		dashboard.PFSummary.TotalInvested +
		dashboard.StockSummary.TotalInvested +
		dashboard.NPSSummary.TotalInvested

	dashboard.CurrentValue = dashboard.MFSummary.CurrentValue +
		dashboard.BondSummary.CurrentValue +
		dashboard.FDSummary.CurrentValue +
		dashboard.PFSummary.CurrentValue +
		dashboard.StockSummary.CurrentValue +
		dashboard.NPSSummary.CurrentValue

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
		if dashboard.NPSSummary.CurrentValue > 0 {
			dashboard.AssetAllocation["NPS"] = dashboard.NPSSummary.CurrentValue
		}
	}

	// Portfolio XIRR
	if len(allCashflows) > 0 && dashboard.CurrentValue > 0 {
		// Add terminal value for remaining holdings
		terminalValue := dashboard.MFSummary.CurrentValue + dashboard.StockSummary.CurrentValue +
			dashboard.BondSummary.CurrentValue + dashboard.FDSummary.CurrentValue +
			dashboard.PFSummary.CurrentValue + dashboard.NPSSummary.CurrentValue
		allCashflows = append(allCashflows, Cashflow{Date: now, Amount: terminalValue})
		if xirr, err := CalculateXIRR(allCashflows); err == nil {
			dashboard.PortfolioXIRR = xirr * 100
		}
	}

	// Risk Metrics
	totalAssets := dashboard.CurrentValue
	totalLiabilities := dashboard.HomeLoanSummary.TotalOutstanding + dashboard.PersonalLoanSummary.TotalOutstanding + dashboard.CreditCardSummary.TotalOutstanding

	// Equity/Debt Ratio
	if debtValue > 0 {
		dashboard.RiskMetrics.EquityDebtRatio = equityValue / debtValue
	}

	// Loan-to-Asset Ratio
	if totalAssets > 0 {
		dashboard.RiskMetrics.LoanToAssetRatio = totalLiabilities / totalAssets
	}

	// Diversification (Herfindahl Index: lower = more diversified)
	if totalAssets > 0 {
		categories := []float64{
			dashboard.MFSummary.CurrentValue,
			dashboard.BondSummary.CurrentValue,
			dashboard.FDSummary.CurrentValue,
			dashboard.PFSummary.CurrentValue,
			dashboard.StockSummary.CurrentValue,
			dashboard.NPSSummary.CurrentValue,
		}
		hhi := 0.0
		activeCategories := 0
		for _, v := range categories {
			if v > 0 {
				share := v / totalAssets
				hhi += share * share
				activeCategories++
			}
		}
		// Normalize: 1.0 = fully concentrated, 0.0 = perfectly diversified
		// Convert to 0-100 diversification score (higher = better)
		if activeCategories > 1 {
			minHHI := 1.0 / float64(activeCategories)
			dashboard.RiskMetrics.Diversification = (1.0 - hhi) / (1.0 - minHHI) * 100
		}
	}

	// Concentration Risk
	if totalAssets > 0 {
		maxPct := 0.0
		maxCat := ""
		catMap := map[string]float64{
			"Mutual Funds": dashboard.MFSummary.CurrentValue,
			"Stocks":       dashboard.StockSummary.CurrentValue,
			"Bonds":        dashboard.BondSummary.CurrentValue,
			"FDs":          dashboard.FDSummary.CurrentValue,
			"PF":           dashboard.PFSummary.CurrentValue,
			"NPS":          dashboard.NPSSummary.CurrentValue,
		}
		for name, val := range catMap {
			pct := val / totalAssets * 100
			if pct > maxPct {
				maxPct = pct
				maxCat = name
			}
		}
		if maxPct > 70 {
			dashboard.RiskMetrics.ConcentrationRisk = maxCat
		}
	}

	// Risk Score (1-10: 1=very conservative, 10=very aggressive)
	score := 5 // Base
	eqPct := 0.0
	if totalAssets > 0 {
		eqPct = equityValue / totalAssets * 100
	}
	if eqPct > 80 {
		score += 3
	} else if eqPct > 60 {
		score += 2
	} else if eqPct > 40 {
		score += 1
	} else if eqPct < 20 {
		score -= 2
	}
	if dashboard.RiskMetrics.LoanToAssetRatio > 0.5 {
		score += 1
	}
	if dashboard.RiskMetrics.Diversification < 30 {
		score += 1
	}
	if score < 1 {
		score = 1
	} else if score > 10 {
		score = 10
	}
	dashboard.RiskMetrics.RiskScore = score

	dashboard.DataSources = map[string]string{
		"mutual_funds": "mfapi.in",
		"stocks":       "Yahoo Finance",
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

type RebalanceSuggestion struct {
	Category      string  `json:"category"`
	CurrentPct    float64 `json:"current_pct"`
	TargetPct     float64 `json:"target_pct"`
	DiffPct       float64 `json:"diff_pct"`
	CurrentValue  float64 `json:"current_value"`
	TargetValue   float64 `json:"target_value"`
	AdjustmentAmt float64 `json:"adjustment_amount"`
	Action        string  `json:"action"` // "buy_more" or "reduce"
}

type RebalanceResponse struct {
	TotalValue  float64               `json:"total_value"`
	Suggestions []RebalanceSuggestion `json:"suggestions"`
}

func (s *DashboardService) GetRebalanceSuggestions(ctx context.Context, userID string, targets map[string]float64) (*RebalanceResponse, error) {
	dashboard, err := s.GetDashboard(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalValue := dashboard.CurrentValue
	if totalValue <= 0 {
		return &RebalanceResponse{TotalValue: 0}, nil
	}

	var suggestions []RebalanceSuggestion
	for category, targetPct := range targets {
		currentVal := dashboard.AssetAllocation[category]
		currentPct := (currentVal / totalValue) * 100
		targetVal := totalValue * targetPct / 100
		diff := targetPct - currentPct
		action := "on_target"
		if diff > 1 {
			action = "buy_more"
		} else if diff < -1 {
			action = "reduce"
		}
		suggestions = append(suggestions, RebalanceSuggestion{
			Category:      category,
			CurrentPct:    currentPct,
			TargetPct:     targetPct,
			DiffPct:       diff,
			CurrentValue:  currentVal,
			TargetValue:   targetVal,
			AdjustmentAmt: targetVal - currentVal,
			Action:        action,
		})
	}

	return &RebalanceResponse{TotalValue: totalValue, Suggestions: suggestions}, nil
}
