package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"investment-tracker/repository"
)

type CashflowService struct {
	mfRepo           *repository.MutualFundRepo
	bondRepo         *repository.CorporateBondRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	stockRepo        *repository.StockRepo
	npsRepo          *repository.NPSRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	creditCardRepo   *repository.CreditCardRepo
}

type MonthlyCashflow struct {
	Month   string  `json:"month"`
	Inflow  float64 `json:"inflow"`
	Outflow float64 `json:"outflow"`
	Net     float64 `json:"net"`
}

type CashflowDetail struct {
	Date      string  `json:"date"`
	Category  string  `json:"category"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Direction string  `json:"direction"`
}

func NewCashflowService(mfRepo *repository.MutualFundRepo, bondRepo *repository.CorporateBondRepo, fdRepo *repository.FixedDepositRepo, pfRepo *repository.ProvidentFundRepo, stockRepo *repository.StockRepo, npsRepo *repository.NPSRepo, homeLoanRepo *repository.HomeLoanRepo, personalLoanRepo *repository.PersonalLoanRepo, creditCardRepo *repository.CreditCardRepo) *CashflowService {
	return &CashflowService{mfRepo: mfRepo, bondRepo: bondRepo, fdRepo: fdRepo, pfRepo: pfRepo, stockRepo: stockRepo, npsRepo: npsRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo, creditCardRepo: creditCardRepo}
}

func (s *CashflowService) GetMonthlyCashflows(ctx context.Context, userID string, months int) ([]MonthlyCashflow, error) {
	if months <= 0 {
		months = 12
	}
	if months > 120 {
		months = 120
	}

	// Determine the cutoff date
	now := time.Now()
	cutoff := now.AddDate(0, -months, 0)

	// Map: "2025-01" -> {inflow, outflow}
	type flowPair struct {
		inflow  float64
		outflow float64
	}
	flows := make(map[string]*flowPair)

	monthKey := func(t time.Time) string {
		return t.Format("2006-01")
	}

	getOrCreate := func(key string) *flowPair {
		if f, ok := flows[key]; ok {
			return f
		}
		f := &flowPair{}
		flows[key] = f
		return f
	}

	// 1. Mutual Fund transactions
	funds, err := s.mfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, fund := range funds {
			for _, tx := range fund.Transactions {
				if tx.Date.Before(cutoff) {
					continue
				}
				key := monthKey(tx.Date)
				fp := getOrCreate(key)
				switch tx.Type {
				case "purchase", "sip", "switch_in":
					fp.outflow += tx.Amount
				case "redemption", "switch_out", "dividend":
					fp.inflow += tx.Amount
				}
			}
		}
	}

	// 2. Stock transactions
	stocks, err := s.stockRepo.GetAll(ctx, userID)
	if err == nil {
		for _, stock := range stocks {
			for _, tx := range stock.Transactions {
				if tx.Date.Before(cutoff) {
					continue
				}
				key := monthKey(tx.Date)
				fp := getOrCreate(key)
				total := tx.PricePerShare * float64(tx.Quantity)
				switch tx.Type {
				case "buy":
					fp.outflow += total
				case "sell":
					fp.inflow += total
				}
			}
		}
	}

	// 3. Corporate Bond interest payouts
	bonds, err := s.bondRepo.GetAll(ctx, userID)
	if err == nil {
		for _, bond := range bonds {
			if bond.PurchaseDate.After(cutoff) || bond.PurchaseDate.Equal(cutoff) {
				key := monthKey(bond.PurchaseDate)
				fp := getOrCreate(key)
				fp.outflow += bond.InvestmentAmount
			}
			for _, p := range bond.InterestPayouts {
				if p.ScheduledDate.Before(cutoff) || p.Status != "received" {
					continue
				}
				key := monthKey(p.ScheduledDate)
				fp := getOrCreate(key)
				fp.inflow += p.Amount
			}
			for _, pr := range bond.PrincipalRepayments {
				if pr.ScheduledDate.Before(cutoff) || pr.Status != "received" {
					continue
				}
				key := monthKey(pr.ScheduledDate)
				fp := getOrCreate(key)
				fp.inflow += pr.Amount
			}
		}
	}

	// 4. Fixed Deposits
	fds, err := s.fdRepo.GetAll(ctx, userID)
	if err == nil {
		for _, fd := range fds {
			if fd.StartDate.After(cutoff) || fd.StartDate.Equal(cutoff) {
				key := monthKey(fd.StartDate)
				fp := getOrCreate(key)
				fp.outflow += fd.PrincipalAmount
			}
			if fd.Status == "matured" && (fd.MaturityDate.After(cutoff) || fd.MaturityDate.Equal(cutoff)) {
				key := monthKey(fd.MaturityDate)
				fp := getOrCreate(key)
				fp.inflow += fd.MaturityAmount
			}
		}
	}

	// 5. Provident Fund monthly contributions
	pfs, err := s.pfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, pf := range pfs {
			for _, fy := range pf.FinancialYearEntries {
				for _, mc := range fy.MonthlyContributions {
					t := parsePFMonth(mc.Month, fy.FinancialYear)
					if t.Before(cutoff) {
						continue
					}
					key := monthKey(t)
					fp := getOrCreate(key)
					fp.outflow += mc.EmployeeContribution + mc.EmployerContribution
				}
			}
		}
	}

	// 6. NPS contributions
	npsAccounts, err := s.npsRepo.GetAll(ctx, userID)
	if err == nil {
		for _, nps := range npsAccounts {
			for _, c := range nps.Contributions {
				if c.Date.Before(cutoff) {
					continue
				}
				key := monthKey(c.Date)
				fp := getOrCreate(key)
				fp.outflow += c.Amount
			}
		}
	}

	// 7. Stock dividends (inflow)
	if err == nil {
		for _, stock := range stocks {
			for _, div := range stock.Dividends {
				if div.Date.Before(cutoff) {
					continue
				}
				key := monthKey(div.Date)
				fp := getOrCreate(key)
				fp.inflow += div.TotalAmount
			}
		}
	}

	// 8. FD interest payouts — non-cumulative FDs (inflow)
	if err == nil {
		for _, fd := range fds {
			for _, p := range fd.InterestPayouts {
				if p.ScheduledDate.Before(cutoff) || p.Status != "received" {
					continue
				}
				key := monthKey(p.ScheduledDate)
				fp := getOrCreate(key)
				fp.inflow += p.Amount
			}
		}
	}

	// 9. Home Loan EMIs and prepayments (outflow)
	homeLoans, err := s.homeLoanRepo.GetAll(ctx, userID)
	if err == nil {
		for _, loan := range homeLoans {
			for _, emi := range loan.EMIsPaid {
				if emi.PaidDate == nil || emi.PaidDate.Before(cutoff) {
					continue
				}
				key := monthKey(*emi.PaidDate)
				fp := getOrCreate(key)
				fp.outflow += emi.EMIAmount
			}
			for _, prep := range loan.Prepayments {
				if prep.Date.Before(cutoff) {
					continue
				}
				key := monthKey(prep.Date)
				fp := getOrCreate(key)
				fp.outflow += prep.Amount
			}
		}
	}

	// 10. Personal Loan EMIs and prepayments (outflow)
	personalLoans, err := s.personalLoanRepo.GetAll(ctx, userID)
	if err == nil {
		for _, loan := range personalLoans {
			for _, emi := range loan.EMIsPaid {
				if emi.PaidDate == nil || emi.PaidDate.Before(cutoff) {
					continue
				}
				key := monthKey(*emi.PaidDate)
				fp := getOrCreate(key)
				fp.outflow += emi.EMIAmount
			}
			for _, prep := range loan.Prepayments {
				if prep.Date.Before(cutoff) {
					continue
				}
				key := monthKey(prep.Date)
				fp := getOrCreate(key)
				fp.outflow += prep.Amount
			}
		}
	}

	// 11. Credit Card statement payments (outflow)
	creditCards, err := s.creditCardRepo.GetAll(ctx, userID)
	if err == nil {
		for _, cc := range creditCards {
			for _, stmt := range cc.Statements {
				if !stmt.IsPaid || stmt.PaidDate == nil || stmt.PaidDate.Before(cutoff) {
					continue
				}
				key := monthKey(*stmt.PaidDate)
				fp := getOrCreate(key)
				fp.outflow += stmt.AmountPaid
			}
		}
	}
	// Convert map to sorted slice
	result := make([]MonthlyCashflow, 0, len(flows))
	for month, fp := range flows {
		result = append(result, MonthlyCashflow{
			Month:   month,
			Inflow:  fp.inflow,
			Outflow: fp.outflow,
			Net:     fp.inflow - fp.outflow,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Month < result[j].Month
	})

	return result, nil
}

// GetMonthDetail returns itemized transactions for a specific month (format "2025-01")
func (s *CashflowService) GetMonthDetail(ctx context.Context, userID string, month string) ([]CashflowDetail, error) {
	// Parse month to get start/end
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("invalid month format, expected YYYY-MM")
	}
	end := start.AddDate(0, 1, 0)

	inMonth := func(t time.Time) bool {
		return !t.Before(start) && t.Before(end)
	}

	var details []CashflowDetail

	// 1. Mutual Fund transactions
	funds, err := s.mfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, fund := range funds {
			for _, tx := range fund.Transactions {
				if !inMonth(tx.Date) {
					continue
				}
				dir := "outflow"
				switch tx.Type {
				case "redemption", "switch_out", "dividend":
					dir = "inflow"
				}
				details = append(details, CashflowDetail{
					Date:      tx.Date.Format("2006-01-02"),
					Category:  "Mutual Fund",
					Name:      fund.FundName,
					Type:      string(tx.Type),
					Amount:    math.Abs(tx.Amount),
					Direction: dir,
				})
			}
		}
	}

	// 2. Stock transactions
	stocks, err := s.stockRepo.GetAll(ctx, userID)
	if err == nil {
		for _, stock := range stocks {
			for _, tx := range stock.Transactions {
				if !inMonth(tx.Date) {
					continue
				}
				dir := "outflow"
				if tx.Type == "sell" {
					dir = "inflow"
				}
				total := tx.PricePerShare * float64(tx.Quantity)
				details = append(details, CashflowDetail{
					Date:      tx.Date.Format("2006-01-02"),
					Category:  "Stock",
					Name:      stock.Symbol,
					Type:      string(tx.Type),
					Amount:    total,
					Direction: dir,
				})
			}
		}
	}

	// 3. Corporate Bonds
	bonds, err := s.bondRepo.GetAll(ctx, userID)
	if err == nil {
		for _, bond := range bonds {
			if inMonth(bond.PurchaseDate) {
				details = append(details, CashflowDetail{
					Date:      bond.PurchaseDate.Format("2006-01-02"),
					Category:  "Corporate Bond",
					Name:      bond.BondName,
					Type:      "purchase",
					Amount:    bond.InvestmentAmount,
					Direction: "outflow",
				})
			}
			for _, p := range bond.InterestPayouts {
				if p.Status == "received" && inMonth(p.ScheduledDate) {
					details = append(details, CashflowDetail{
						Date:      p.ScheduledDate.Format("2006-01-02"),
						Category:  "Corporate Bond",
						Name:      bond.BondName,
						Type:      "interest",
						Amount:    p.Amount,
						Direction: "inflow",
					})
				}
			}
			for _, pr := range bond.PrincipalRepayments {
				if pr.Status == "received" && inMonth(pr.ScheduledDate) {
					details = append(details, CashflowDetail{
						Date:      pr.ScheduledDate.Format("2006-01-02"),
						Category:  "Corporate Bond",
						Name:      bond.BondName,
						Type:      "principal repayment",
						Amount:    pr.Amount,
						Direction: "inflow",
					})
				}
			}
		}
	}

	// 4. Fixed Deposits
	fds, err := s.fdRepo.GetAll(ctx, userID)
	if err == nil {
		for _, fd := range fds {
			if inMonth(fd.StartDate) {
				details = append(details, CashflowDetail{
					Date:      fd.StartDate.Format("2006-01-02"),
					Category:  "Fixed Deposit",
					Name:      fd.BankName + " FD",
					Type:      "deposit",
					Amount:    fd.PrincipalAmount,
					Direction: "outflow",
				})
			}
			if fd.Status == "matured" && inMonth(fd.MaturityDate) {
				details = append(details, CashflowDetail{
					Date:      fd.MaturityDate.Format("2006-01-02"),
					Category:  "Fixed Deposit",
					Name:      fd.BankName + " FD",
					Type:      "maturity",
					Amount:    fd.MaturityAmount,
					Direction: "inflow",
				})
			}
		}
	}

	// 5. Provident Fund
	pfs, err := s.pfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, pf := range pfs {
			for _, fy := range pf.FinancialYearEntries {
				for _, mc := range fy.MonthlyContributions {
					t := parsePFMonth(mc.Month, fy.FinancialYear)
					if !inMonth(t) {
						continue
					}
					label := string(pf.AccountType) + " " + pf.AccountNumber
					if mc.EmployeeContribution > 0 {
						details = append(details, CashflowDetail{
							Date:      t.Format("2006-01-02"),
							Category:  "Provident Fund",
							Name:      label,
							Type:      "employee contribution",
							Amount:    mc.EmployeeContribution,
							Direction: "outflow",
						})
					}
					if mc.EmployerContribution > 0 {
						details = append(details, CashflowDetail{
							Date:      t.Format("2006-01-02"),
							Category:  "Provident Fund",
							Name:      label,
							Type:      "employer contribution",
							Amount:    mc.EmployerContribution,
							Direction: "outflow",
						})
					}
				}
			}
		}
	}

	// 6. NPS contributions
	npsAccounts, err := s.npsRepo.GetAll(ctx, userID)
	if err == nil {
		for _, nps := range npsAccounts {
			for _, c := range nps.Contributions {
				if !inMonth(c.Date) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      c.Date.Format("2006-01-02"),
					Category:  "NPS",
					Name:      nps.AccountHolderName + " (" + nps.PRAN + ")",
					Type:      c.Type + " contribution",
					Amount:    c.Amount,
					Direction: "outflow",
				})
			}
		}
	}

	// 7. Stock dividends (inflow)
	if err == nil {
		for _, stock := range stocks {
			for _, div := range stock.Dividends {
				if !inMonth(div.Date) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      div.Date.Format("2006-01-02"),
					Category:  "Stock",
					Name:      stock.Symbol,
					Type:      "dividend",
					Amount:    div.TotalAmount,
					Direction: "inflow",
				})
			}
		}
	}

	// 8. FD interest payouts (inflow)
	if err == nil {
		for _, fd := range fds {
			for _, p := range fd.InterestPayouts {
				if p.Status != "received" || !inMonth(p.ScheduledDate) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      p.ScheduledDate.Format("2006-01-02"),
					Category:  "Fixed Deposit",
					Name:      fd.BankName + " FD",
					Type:      "interest payout",
					Amount:    p.Amount,
					Direction: "inflow",
				})
			}
		}
	}

	// 9. Home Loan EMIs and prepayments (outflow)
	homeLoans, err := s.homeLoanRepo.GetAll(ctx, userID)
	if err == nil {
		for _, loan := range homeLoans {
			for _, emi := range loan.EMIsPaid {
				if emi.PaidDate == nil || !inMonth(*emi.PaidDate) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      emi.PaidDate.Format("2006-01-02"),
					Category:  "Home Loan",
					Name:      loan.BankName + " EMI",
					Type:      "emi",
					Amount:    emi.EMIAmount,
					Direction: "outflow",
				})
			}
			for _, prep := range loan.Prepayments {
				if !inMonth(prep.Date) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      prep.Date.Format("2006-01-02"),
					Category:  "Home Loan",
					Name:      loan.BankName + " Prepayment",
					Type:      "prepayment",
					Amount:    prep.Amount,
					Direction: "outflow",
				})
			}
		}
	}

	// 10. Personal Loan EMIs and prepayments (outflow)
	personalLoans, err := s.personalLoanRepo.GetAll(ctx, userID)
	if err == nil {
		for _, loan := range personalLoans {
			for _, emi := range loan.EMIsPaid {
				if emi.PaidDate == nil || !inMonth(*emi.PaidDate) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      emi.PaidDate.Format("2006-01-02"),
					Category:  "Personal Loan",
					Name:      loan.LenderName + " EMI",
					Type:      "emi",
					Amount:    emi.EMIAmount,
					Direction: "outflow",
				})
			}
			for _, prep := range loan.Prepayments {
				if !inMonth(prep.Date) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      prep.Date.Format("2006-01-02"),
					Category:  "Personal Loan",
					Name:      loan.LenderName + " Prepayment",
					Type:      "prepayment",
					Amount:    prep.Amount,
					Direction: "outflow",
				})
			}
		}
	}

	// 11. Credit Card statement payments (outflow)
	creditCards, err := s.creditCardRepo.GetAll(ctx, userID)
	if err == nil {
		for _, cc := range creditCards {
			for _, stmt := range cc.Statements {
				if !stmt.IsPaid || stmt.PaidDate == nil || !inMonth(*stmt.PaidDate) {
					continue
				}
				details = append(details, CashflowDetail{
					Date:      stmt.PaidDate.Format("2006-01-02"),
					Category:  "Credit Card",
					Name:      cc.CardName + " Payment",
					Type:      "bill payment",
					Amount:    stmt.AmountPaid,
					Direction: "outflow",
				})
			}
		}
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].Date < details[j].Date
	})

	return details, nil
}
