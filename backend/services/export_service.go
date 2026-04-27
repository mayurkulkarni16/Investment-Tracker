package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"investment-tracker/repository"
)

type ExportService struct {
	mfRepo           *repository.MutualFundRepo
	stockRepo        *repository.StockRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	bondRepo         *repository.CorporateBondRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	npsRepo          *repository.NPSRepo
}

func NewExportService(
	mfRepo *repository.MutualFundRepo,
	stockRepo *repository.StockRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	bondRepo *repository.CorporateBondRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	npsRepo *repository.NPSRepo,
) *ExportService {
	return &ExportService{
		mfRepo: mfRepo, stockRepo: stockRepo, fdRepo: fdRepo, pfRepo: pfRepo,
		bondRepo: bondRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		npsRepo: npsRepo,
	}
}

func (s *ExportService) ExportCSV(ctx context.Context, module string) ([]byte, string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	filename := fmt.Sprintf("investment_tracker_%s_%s.csv", module, time.Now().Format("20060102"))

	switch module {
	case "mutual_funds":
		s.exportMutualFunds(ctx, w)
	case "stocks":
		s.exportStocks(ctx, w)
	case "fixed_deposits":
		s.exportFDs(ctx, w)
	case "home_loans":
		s.exportHomeLoans(ctx, w)
	case "personal_loans":
		s.exportPersonalLoans(ctx, w)
	case "nps":
		s.exportNPS(ctx, w)
	case "all":
		filename = fmt.Sprintf("investment_tracker_all_%s.csv", time.Now().Format("20060102"))
		s.exportAll(ctx, w)
	default:
		s.exportAll(ctx, w)
	}

	w.Flush()
	return buf.Bytes(), filename, nil
}

func (s *ExportService) exportMutualFunds(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"Type", "Fund Name", "AMC", "Fund Type", "Folio", "Total Invested", "Current Value", "Gain/Loss %", "Total Units", "NAV"})
	mfs, _ := s.mfRepo.GetAll(ctx)
	for _, mf := range mfs {
		w.Write([]string{
			"Mutual Fund", mf.FundName, mf.AMC, string(mf.FundType), mf.FolioNumber,
			fmt.Sprintf("%.2f", mf.TotalInvested), fmt.Sprintf("%.2f", mf.CurrentValue),
			fmt.Sprintf("%.2f", mf.GainLossPercent), fmt.Sprintf("%.4f", mf.TotalUnits), fmt.Sprintf("%.4f", mf.CurrentNAV),
		})
	}
}

func (s *ExportService) exportStocks(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"Type", "Symbol", "Name", "Exchange", "Quantity", "Avg Buy Price", "Current Price", "Invested", "Current Value", "Gain %"})
	stocks, _ := s.stockRepo.GetAll(ctx)
	for _, stock := range stocks {
		gainPct := 0.0
		if stock.TotalInvested > 0 {
			gainPct = (stock.CurrentValue - stock.TotalInvested) / stock.TotalInvested * 100
		}
		w.Write([]string{
			"Stock", stock.Symbol, stock.StockName, string(stock.Exchange),
			fmt.Sprintf("%d", stock.TotalQuantity), fmt.Sprintf("%.2f", stock.AvgBuyPrice),
			fmt.Sprintf("%.2f", stock.CurrentPrice), fmt.Sprintf("%.2f", stock.TotalInvested),
			fmt.Sprintf("%.2f", stock.CurrentValue), fmt.Sprintf("%.2f", gainPct),
		})
	}
}

func (s *ExportService) exportFDs(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"Type", "Bank", "FD Number", "Principal", "Interest Rate", "Maturity Amount", "Start Date", "Maturity Date", "Status"})
	fds, _ := s.fdRepo.GetAll(ctx)
	for _, fd := range fds {
		w.Write([]string{
			"Fixed Deposit", fd.BankName, fd.FDNumber,
			fmt.Sprintf("%.2f", fd.PrincipalAmount), fmt.Sprintf("%.2f", fd.InterestRate),
			fmt.Sprintf("%.2f", fd.MaturityAmount), fd.StartDate.Format("2006-01-02"),
			fd.MaturityDate.Format("2006-01-02"), string(fd.Status),
		})
	}
}

func (s *ExportService) exportHomeLoans(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"Type", "Bank", "Account", "Sanctioned", "Disbursed", "Outstanding", "Rate", "EMI", "Tenure", "Status"})
	loans, _ := s.homeLoanRepo.GetAll(ctx)
	for _, loan := range loans {
		w.Write([]string{
			"Home Loan", loan.BankName, loan.LoanAccountNumber,
			fmt.Sprintf("%.2f", loan.SanctionedAmount), fmt.Sprintf("%.2f", loan.DisbursedAmount),
			fmt.Sprintf("%.2f", loan.OutstandingPrincipal), fmt.Sprintf("%.2f", loan.InterestRate),
			fmt.Sprintf("%.2f", loan.EMIAmount), fmt.Sprintf("%d months", loan.RemainingTenureMonths),
			string(loan.Status),
		})
	}
}

func (s *ExportService) exportPersonalLoans(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"Type", "Lender", "Account", "Disbursed", "Outstanding", "Rate", "EMI", "Status"})
	loans, _ := s.personalLoanRepo.GetAll(ctx)
	for _, loan := range loans {
		w.Write([]string{
			"Personal Loan", loan.LenderName, loan.LoanAccountNumber,
			fmt.Sprintf("%.2f", loan.DisbursedAmount), fmt.Sprintf("%.2f", loan.OutstandingPrincipal),
			fmt.Sprintf("%.2f", loan.InterestRate), fmt.Sprintf("%.2f", loan.EMIAmount),
			string(loan.Status),
		})
	}
}

func (s *ExportService) exportNPS(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"Type", "Holder", "PRAN", "Account Type", "Fund Manager", "Total Contribution", "Current Value"})
	accounts, _ := s.npsRepo.GetAll(ctx)
	for _, nps := range accounts {
		w.Write([]string{
			"NPS", nps.AccountHolderName, nps.PRAN, string(nps.AccountType),
			nps.FundManager, fmt.Sprintf("%.2f", nps.TotalContribution),
			fmt.Sprintf("%.2f", nps.CurrentValue),
		})
	}
}

func (s *ExportService) exportAll(ctx context.Context, w *csv.Writer) {
	w.Write([]string{"--- MUTUAL FUNDS ---"})
	s.exportMutualFunds(ctx, w)
	w.Write([]string{""})
	w.Write([]string{"--- STOCKS ---"})
	s.exportStocks(ctx, w)
	w.Write([]string{""})
	w.Write([]string{"--- FIXED DEPOSITS ---"})
	s.exportFDs(ctx, w)
	w.Write([]string{""})
	w.Write([]string{"--- HOME LOANS ---"})
	s.exportHomeLoans(ctx, w)
	w.Write([]string{""})
	w.Write([]string{"--- PERSONAL LOANS ---"})
	s.exportPersonalLoans(ctx, w)
	w.Write([]string{""})
	w.Write([]string{"--- NPS ---"})
	s.exportNPS(ctx, w)
}
