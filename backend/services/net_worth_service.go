package services

import (
	"context"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type NetWorthService struct {
	snapshotRepo     *repository.NetWorthRepo
	mfRepo           *repository.MutualFundRepo
	stockRepo        *repository.StockRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	npsRepo          *repository.NPSRepo
	bondRepo         *repository.CorporateBondRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	creditCardRepo   *repository.CreditCardRepo
}

func NewNetWorthService(
	snapshotRepo *repository.NetWorthRepo,
	mfRepo *repository.MutualFundRepo,
	stockRepo *repository.StockRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	npsRepo *repository.NPSRepo,
	bondRepo *repository.CorporateBondRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	creditCardRepo *repository.CreditCardRepo,
) *NetWorthService {
	return &NetWorthService{
		snapshotRepo:     snapshotRepo,
		mfRepo:           mfRepo,
		stockRepo:        stockRepo,
		fdRepo:           fdRepo,
		pfRepo:           pfRepo,
		npsRepo:          npsRepo,
		bondRepo:         bondRepo,
		homeLoanRepo:     homeLoanRepo,
		personalLoanRepo: personalLoanRepo,
		creditCardRepo:   creditCardRepo,
	}
}

func (s *NetWorthService) GetCurrent(ctx context.Context) (*models.NetWorthCurrent, error) {
	nw := &models.NetWorthCurrent{
		Assets:      make(map[string]float64),
		Liabilities: make(map[string]float64),
	}

	// Assets
	mfs, _ := s.mfRepo.GetAll(ctx)
	for _, mf := range mfs {
		nw.Assets["Mutual Funds"] += mf.CurrentValue
	}

	stocks, _ := s.stockRepo.GetAll(ctx)
	for _, stock := range stocks {
		nw.Assets["Stocks"] += stock.CurrentValue
	}

	fds, _ := s.fdRepo.GetAll(ctx)
	for _, fd := range fds {
		if fd.Status == "active" {
			nw.Assets["Fixed Deposits"] += fd.PrincipalAmount
		}
	}

	pfs, _ := s.pfRepo.GetAll(ctx)
	for _, pf := range pfs {
		nw.Assets["Provident Fund"] += pf.CurrentBalance
	}

	npsAccounts, _ := s.npsRepo.GetAll(ctx)
	for _, nps := range npsAccounts {
		nw.Assets["NPS"] += nps.CurrentValue
	}

	bonds, _ := s.bondRepo.GetAll(ctx)
	for _, bond := range bonds {
		nw.Assets["Corporate Bonds"] += bond.RemainingPrincipal
	}

	// Liabilities
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx)
	for _, loan := range homeLoans {
		if loan.Status == "active" {
			nw.Liabilities["Home Loans"] += loan.OutstandingPrincipal
		}
	}

	personalLoans, _ := s.personalLoanRepo.GetAll(ctx)
	for _, loan := range personalLoans {
		if loan.Status == "active" {
			nw.Liabilities["Personal Loans"] += loan.OutstandingPrincipal
		}
	}

	creditCards, _ := s.creditCardRepo.GetAll(ctx)
	for _, card := range creditCards {
		if card.Status == "active" {
			nw.Liabilities["Credit Cards"] += card.CurrentOutstanding
		}
	}

	for _, v := range nw.Assets {
		nw.TotalAssets += v
	}
	for _, v := range nw.Liabilities {
		nw.TotalLiabilities += v
	}
	nw.NetWorth = nw.TotalAssets - nw.TotalLiabilities

	return nw, nil
}

func (s *NetWorthService) TakeSnapshot(ctx context.Context) (*models.NetWorthSnapshot, error) {
	current, err := s.GetCurrent(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	snapshot := &models.NetWorthSnapshot{
		Date:                  now,
		Month:                 now.Format("2006-01"),
		MutualFunds:           current.Assets["Mutual Funds"],
		Stocks:                current.Assets["Stocks"],
		FixedDeposits:         current.Assets["Fixed Deposits"],
		ProvidentFund:         current.Assets["Provident Fund"],
		NPS:                   current.Assets["NPS"],
		CorporateBonds:        current.Assets["Corporate Bonds"],
		TotalAssets:           current.TotalAssets,
		HomeLoans:             current.Liabilities["Home Loans"],
		PersonalLoans:         current.Liabilities["Personal Loans"],
		CreditCardOutstanding: current.Liabilities["Credit Cards"],
		TotalLiabilities:      current.TotalLiabilities,
		NetWorth:              current.NetWorth,
	}

	if err := s.snapshotRepo.Upsert(ctx, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *NetWorthService) GetHistory(ctx context.Context) ([]models.NetWorthSnapshot, error) {
	return s.snapshotRepo.GetAll(ctx)
}
