package services

import (
	"context"
	"encoding/json"

	"investment-tracker/repository"
)

type BackupService struct {
	mfRepo           *repository.MutualFundRepo
	stockRepo        *repository.StockRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	bondRepo         *repository.CorporateBondRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	npsRepo          *repository.NPSRepo
	creditCardRepo   *repository.CreditCardRepo
	goalRepo         *repository.GoalRepo
	sipRepo          *repository.SIPRepo
	profileRepo      *repository.ProfileRepo
}

func NewBackupService(
	mfRepo *repository.MutualFundRepo,
	stockRepo *repository.StockRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	bondRepo *repository.CorporateBondRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	npsRepo *repository.NPSRepo,
	creditCardRepo *repository.CreditCardRepo,
	goalRepo *repository.GoalRepo,
	sipRepo *repository.SIPRepo,
	profileRepo *repository.ProfileRepo,
) *BackupService {
	return &BackupService{
		mfRepo: mfRepo, stockRepo: stockRepo, fdRepo: fdRepo, pfRepo: pfRepo,
		bondRepo: bondRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		npsRepo: npsRepo, creditCardRepo: creditCardRepo, goalRepo: goalRepo,
		sipRepo: sipRepo, profileRepo: profileRepo,
	}
}

type BackupData struct {
	Version        string      `json:"version"`
	MutualFunds    interface{} `json:"mutual_funds"`
	Stocks         interface{} `json:"stocks"`
	FixedDeposits  interface{} `json:"fixed_deposits"`
	ProvidentFund  interface{} `json:"provident_fund"`
	CorporateBonds interface{} `json:"corporate_bonds"`
	HomeLoans      interface{} `json:"home_loans"`
	PersonalLoans  interface{} `json:"personal_loans"`
	NPS            interface{} `json:"nps"`
	CreditCards    interface{} `json:"credit_cards"`
	Goals          interface{} `json:"goals"`
	SIPs           interface{} `json:"sips"`
	Profiles       interface{} `json:"profiles"`
}

func (s *BackupService) Export(ctx context.Context) ([]byte, error) {
	mfs, _ := s.mfRepo.GetAll(ctx)
	stocks, _ := s.stockRepo.GetAll(ctx)
	fds, _ := s.fdRepo.GetAll(ctx)
	pfs, _ := s.pfRepo.GetAll(ctx)
	bonds, _ := s.bondRepo.GetAll(ctx)
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx)
	personalLoans, _ := s.personalLoanRepo.GetAll(ctx)
	nps, _ := s.npsRepo.GetAll(ctx)
	creditCards, _ := s.creditCardRepo.GetAll(ctx)
	goals, _ := s.goalRepo.GetAll(ctx)
	sips, _ := s.sipRepo.GetAll(ctx)
	profiles, _ := s.profileRepo.GetAll(ctx)

	backup := BackupData{
		Version:        "1.0",
		MutualFunds:    mfs,
		Stocks:         stocks,
		FixedDeposits:  fds,
		ProvidentFund:  pfs,
		CorporateBonds: bonds,
		HomeLoans:      homeLoans,
		PersonalLoans:  personalLoans,
		NPS:            nps,
		CreditCards:    creditCards,
		Goals:          goals,
		SIPs:           sips,
		Profiles:       profiles,
	}

	return json.MarshalIndent(backup, "", "  ")
}
