package services

import (
	"fmt"
	"strings"
	"time"

	"investment-tracker/models"
)

// ComputeMutualFundXIRR computes XIRR for a mutual fund from its transactions.
func ComputeMutualFundXIRR(fund *models.MutualFund) float64 {
	if len(fund.Transactions) == 0 || fund.CurrentValue <= 0 {
		return 0
	}

	var cashflows []Cashflow
	for _, txn := range fund.Transactions {
		switch txn.Type {
		case models.TransactionPurchase, models.TransactionSIP, models.TransactionSwitchIn:
			cashflows = append(cashflows, Cashflow{Date: txn.Date, Amount: -txn.Amount})
		case models.TransactionRedemption, models.TransactionSwitchOut:
			cashflows = append(cashflows, Cashflow{Date: txn.Date, Amount: txn.Amount})
		}
	}

	// Terminal value: current portfolio value as inflow today
	cashflows = append(cashflows, Cashflow{Date: time.Now(), Amount: fund.CurrentValue})

	xirr, err := CalculateXIRR(cashflows)
	if err != nil {
		return 0
	}
	return xirr * 100 // Return as percentage
}

// ComputeStockXIRR computes XIRR for a stock from its transactions.
func ComputeStockXIRR(stock *models.Stock) float64 {
	if len(stock.Transactions) == 0 || stock.CurrentValue <= 0 {
		return 0
	}

	var cashflows []Cashflow
	for _, txn := range stock.Transactions {
		switch txn.Type {
		case "buy":
			cashflows = append(cashflows, Cashflow{Date: txn.Date, Amount: -txn.Amount})
		case "sell":
			cashflows = append(cashflows, Cashflow{Date: txn.Date, Amount: txn.Amount})
		}
	}

	cashflows = append(cashflows, Cashflow{Date: time.Now(), Amount: stock.CurrentValue})

	xirr, err := CalculateXIRR(cashflows)
	if err != nil {
		return 0
	}
	return xirr * 100
}

// ComputeCorporateBondXIRR computes XIRR for a corporate bond.
func ComputeCorporateBondXIRR(bond *models.CorporateBond) float64 {
	var cashflows []Cashflow

	// Initial investment as outflow
	cashflows = append(cashflows, Cashflow{Date: bond.PurchaseDate, Amount: -bond.InvestmentAmount})

	// Received interest payouts as inflows
	for _, p := range bond.InterestPayouts {
		if p.Status == "received" && p.ReceivedDate != nil {
			cashflows = append(cashflows, Cashflow{Date: *p.ReceivedDate, Amount: p.Amount})
		}
	}

	// Received principal repayments as inflows
	for _, p := range bond.PrincipalRepayments {
		if p.Status == "received" && p.ReceivedDate != nil {
			cashflows = append(cashflows, Cashflow{Date: *p.ReceivedDate, Amount: p.Amount})
		}
	}

	// Terminal value: remaining principal
	if bond.RemainingPrincipal > 0 {
		cashflows = append(cashflows, Cashflow{Date: time.Now(), Amount: bond.RemainingPrincipal})
	}

	if len(cashflows) < 2 {
		return 0
	}

	xirr, err := CalculateXIRR(cashflows)
	if err != nil {
		return 0
	}
	return xirr * 100
}

// ComputeFixedDepositXIRR computes XIRR for a fixed deposit.
func ComputeFixedDepositXIRR(fd *models.FixedDeposit) float64 {
	var cashflows []Cashflow

	// Principal as outflow at start
	cashflows = append(cashflows, Cashflow{Date: fd.StartDate, Amount: -fd.PrincipalAmount})

	// Terminal value
	if fd.Status == "matured" {
		cashflows = append(cashflows, Cashflow{Date: fd.MaturityDate, Amount: fd.MaturityAmount})
	} else {
		// For active FDs, use current accrued value
		now := time.Now()
		if now.After(fd.MaturityDate) {
			cashflows = append(cashflows, Cashflow{Date: fd.MaturityDate, Amount: fd.MaturityAmount})
		} else {
			// Pro-rata accrued value
			totalDays := fd.MaturityDate.Sub(fd.StartDate).Hours() / 24
			elapsedDays := now.Sub(fd.StartDate).Hours() / 24
			if totalDays > 0 {
				fraction := elapsedDays / totalDays
				accruedValue := fd.PrincipalAmount + (fd.MaturityAmount-fd.PrincipalAmount)*fraction
				cashflows = append(cashflows, Cashflow{Date: now, Amount: accruedValue})
			}
		}
	}

	if len(cashflows) < 2 {
		return 0
	}

	xirr, err := CalculateXIRR(cashflows)
	if err != nil {
		return 0
	}
	return xirr * 100
}

// ComputeProvidentFundXIRR computes XIRR for a provident fund account.
func ComputeProvidentFundXIRR(pf *models.ProvidentFund) float64 {
	if pf.CurrentBalance <= 0 {
		return 0
	}

	var cashflows []Cashflow

	for _, fy := range pf.FinancialYearEntries {
		for _, mc := range fy.MonthlyContributions {
			date := parsePFMonth(mc.Month, fy.FinancialYear)
			if date.IsZero() {
				continue
			}
			total := mc.EmployeeContribution + mc.EmployerContribution
			if total > 0 {
				cashflows = append(cashflows, Cashflow{Date: date, Amount: -total})
			}
		}
	}

	if len(cashflows) == 0 {
		return 0
	}

	cashflows = append(cashflows, Cashflow{Date: time.Now(), Amount: pf.CurrentBalance})

	xirr, err := CalculateXIRR(cashflows)
	if err != nil {
		return 0
	}
	return xirr * 100
}

// ComputeNPSXIRR computes XIRR for an NPS account.
func ComputeNPSXIRR(nps *models.NPSAccount) float64 {
	if len(nps.Contributions) == 0 || nps.CurrentValue <= 0 {
		return 0
	}

	var cashflows []Cashflow
	for _, c := range nps.Contributions {
		cashflows = append(cashflows, Cashflow{Date: c.Date, Amount: -c.Amount})
	}

	cashflows = append(cashflows, Cashflow{Date: time.Now(), Amount: nps.CurrentValue})

	xirr, err := CalculateXIRR(cashflows)
	if err != nil {
		return 0
	}
	return xirr * 100
}

// parsePFMonth converts a PF month string like "Apr-2025" and financial year to a time.Time.
func parsePFMonth(monthStr string, fy string) time.Time {
	// Try parsing "Apr-2025", "Apr 2025", "2025-04" formats
	formats := []string{"Jan-2006", "Jan 2006", "January-2006", "January 2006", "2006-01"}
	for _, f := range formats {
		if t, err := time.Parse(f, monthStr); err == nil {
			return t
		}
	}

	// Try extracting month name and computing year from FY
	monthNames := map[string]time.Month{
		"jan": time.January, "feb": time.February, "mar": time.March,
		"apr": time.April, "may": time.May, "jun": time.June,
		"jul": time.July, "aug": time.August, "sep": time.September,
		"oct": time.October, "nov": time.November, "dec": time.December,
	}

	lower := strings.ToLower(monthStr)
	for name, month := range monthNames {
		if strings.HasPrefix(lower, name) {
			// Parse FY like "2024-2025" or "2024-25"
			var startYear int
			if _, err := fmt.Sscanf(fy, "%d-", &startYear); err == nil {
				year := startYear
				if month >= time.April {
					// Apr-Mar: first half of FY
				} else {
					year++ // Jan-Mar: second half of FY
				}
				return time.Date(year, month, 15, 0, 0, 0, 0, time.Local) // Mid-month
			}
		}
	}

	return time.Time{}
}
