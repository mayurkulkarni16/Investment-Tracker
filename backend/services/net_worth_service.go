package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type NetWorthService struct {
	snapshotRepo      *repository.NetWorthRepo
	mfRepo            *repository.MutualFundRepo
	stockRepo         *repository.StockRepo
	fdRepo            *repository.FixedDepositRepo
	pfRepo            *repository.ProvidentFundRepo
	npsRepo           *repository.NPSRepo
	bondRepo          *repository.CorporateBondRepo
	homeLoanRepo      *repository.HomeLoanRepo
	personalLoanRepo  *repository.PersonalLoanRepo
	creditCardRepo    *repository.CreditCardRepo
	vehicleRentalRepo *repository.VehicleRentalRepo
	bankRepo          *repository.BankAccountRepo
	goldRepo          *repository.GoldRepo
	propertyRepo      *repository.PropertyRepo
	insuranceRepo     *repository.InsuranceRepo
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
	vehicleRentalRepo *repository.VehicleRentalRepo,
) *NetWorthService {
	return &NetWorthService{
		snapshotRepo:      snapshotRepo,
		mfRepo:            mfRepo,
		stockRepo:         stockRepo,
		fdRepo:            fdRepo,
		pfRepo:            pfRepo,
		npsRepo:           npsRepo,
		bondRepo:          bondRepo,
		homeLoanRepo:      homeLoanRepo,
		personalLoanRepo:  personalLoanRepo,
		creditCardRepo:    creditCardRepo,
		vehicleRentalRepo: vehicleRentalRepo,
	}
}

// SetPersonalFinanceRepos plugs in the cash, gold, real-estate and insurance
// holdings so they count towards net worth.
func (s *NetWorthService) SetPersonalFinanceRepos(bank *repository.BankAccountRepo, gold *repository.GoldRepo, property *repository.PropertyRepo, insurance *repository.InsuranceRepo) {
	s.bankRepo, s.goldRepo, s.propertyRepo, s.insuranceRepo = bank, gold, property, insurance
}

func nonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ListItems returns every asset and liability with its current value.
func (s *NetWorthService) ListItems(ctx context.Context, userID string) []models.AssetItem {
	var items []models.AssetItem
	add := func(assetType, id, name, category string, value float64, liability bool) {
		if value <= 0 {
			return
		}
		items = append(items, models.AssetItem{AssetType: assetType, AssetID: id, Name: name, Category: category, Value: round2(value), IsLiability: liability})
	}
	now := time.Now()

	mfs, _ := s.mfRepo.GetAll(ctx, userID)
	for _, mf := range mfs {
		add("mutual_fund", mf.ID.Hex(), mf.FundName, "Mutual Funds", mf.CurrentValue, false)
	}
	stocks, _ := s.stockRepo.GetAll(ctx, userID)
	for _, st := range stocks {
		add("stock", st.ID.Hex(), nonEmpty(st.StockName, st.Symbol), "Stocks", st.CurrentValue, false)
	}
	fds, _ := s.fdRepo.GetAll(ctx, userID)
	for i := range fds {
		NormalizeFixedDeposit(&fds[i], now)
		add("fixed_deposit", fds[i].ID.Hex(), fds[i].BankName+" FD", "Fixed Deposits", FixedDepositCurrentValue(&fds[i], now), false)
	}
	pfs, _ := s.pfRepo.GetAll(ctx, userID)
	for _, pf := range pfs {
		add("provident_fund", pf.ID.Hex(), string(pf.AccountType)+" "+nonEmpty(pf.EmployerName, pf.AccountNumber), "Provident Fund", pf.CurrentBalance, false)
	}
	npsAccounts, _ := s.npsRepo.GetAll(ctx, userID)
	for _, n := range npsAccounts {
		add("nps", n.ID.Hex(), "NPS "+n.PRAN, "NPS", n.CurrentValue, false)
	}
	bonds, _ := s.bondRepo.GetAll(ctx, userID)
	for i := range bonds {
		NormalizeCorporateBond(&bonds[i])
		add("corporate_bond", bonds[i].ID.Hex(), bonds[i].BondName, "Corporate Bonds", bonds[i].RemainingPrincipal, false)
	}
	rentals, _ := s.vehicleRentalRepo.GetAll(ctx, userID)
	for i := range rentals {
		ComputeVehicleRental(&rentals[i], now)
		add("vehicle_rental", rentals[i].ID.Hex(), nonEmpty(rentals[i].VehicleModel, "Vehicle rental"), "Vehicle Rentals", rentals[i].CurrentValue, false)
	}
	if s.bankRepo != nil {
		accounts, _ := s.bankRepo.GetAll(ctx, userID)
		for _, a := range accounts {
			if a.Status != "closed" {
				add("bank_account", a.ID.Hex(), a.Name, "Bank & Cash", a.Balance, false)
			}
		}
	}
	if s.goldRepo != nil {
		gold, _ := s.goldRepo.GetAll(ctx, userID)
		for i := range gold {
			ComputeGold(&gold[i], now)
			add("gold", gold[i].ID.Hex(), gold[i].Name, "Gold", gold[i].CurrentValue, false)
		}
	}
	if s.propertyRepo != nil {
		props, _ := s.propertyRepo.GetAll(ctx, userID)
		for i := range props {
			ComputeProperty(&props[i], nil, now)
			add("property", props[i].ID.Hex(), props[i].Name, "Real Estate", props[i].OwnedValue, false)
		}
	}
	if s.insuranceRepo != nil {
		policies, _ := s.insuranceRepo.GetAll(ctx, userID)
		for _, p := range policies {
			if p.Status == "" || p.Status == "active" {
				add("insurance", p.ID.Hex(), nonEmpty(p.PolicyName, p.Insurer), "Insurance", p.CurrentValue, false)
			}
		}
	}

	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	for _, l := range homeLoans {
		if l.Status == models.LoanActive {
			add("home_loan", l.ID.Hex(), l.BankName+" home loan", "Home Loans", l.OutstandingPrincipal, true)
		}
	}
	personalLoans, _ := s.personalLoanRepo.GetAll(ctx, userID)
	for _, l := range personalLoans {
		if l.Status == "active" {
			add("personal_loan", l.ID.Hex(), l.LenderName+" loan", "Personal Loans", l.OutstandingPrincipal, true)
		}
	}
	cards, _ := s.creditCardRepo.GetAll(ctx, userID)
	for _, c := range cards {
		if c.Status == "active" {
			add("credit_card", c.ID.Hex(), nonEmpty(c.CardName, c.BankName), "Credit Cards", c.CurrentOutstanding, true)
		}
	}
	return items
}

func (s *NetWorthService) GetCurrent(ctx context.Context, userID string) (*models.NetWorthCurrent, error) {
	nw := &models.NetWorthCurrent{
		Assets:      make(map[string]float64),
		Liabilities: make(map[string]float64),
	}
	for _, it := range s.ListItems(ctx, userID) {
		if it.IsLiability {
			nw.Liabilities[it.Category] += it.Value
			nw.TotalLiabilities += it.Value
		} else {
			nw.Assets[it.Category] += it.Value
			nw.TotalAssets += it.Value
		}
	}
	nw.NetWorth = nw.TotalAssets - nw.TotalLiabilities
	return nw, nil
}

func (s *NetWorthService) TakeSnapshot(ctx context.Context, userID string) (*models.NetWorthSnapshot, error) {
	if userID == "" {
		return nil, errors.New("net worth snapshot requires a user")
	}
	current, err := s.GetCurrent(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	snapshot := &models.NetWorthSnapshot{
		UserID:                userID,
		Date:                  now,
		Month:                 now.Format("2006-01"),
		MutualFunds:           current.Assets["Mutual Funds"],
		Stocks:                current.Assets["Stocks"],
		FixedDeposits:         current.Assets["Fixed Deposits"],
		ProvidentFund:         current.Assets["Provident Fund"],
		NPS:                   current.Assets["NPS"],
		CorporateBonds:        current.Assets["Corporate Bonds"],
		VehicleRentals:        current.Assets["Vehicle Rentals"],
		BankCash:              current.Assets["Bank & Cash"],
		Gold:                  current.Assets["Gold"],
		RealEstate:            current.Assets["Real Estate"],
		Insurance:             current.Assets["Insurance"],
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

func (s *NetWorthService) GetHistory(ctx context.Context, userID string) ([]models.NetWorthSnapshot, error) {
	return s.snapshotRepo.GetAll(ctx, userID)
}

func (s *NetWorthService) DeleteOrphanSnapshots(ctx context.Context) (int64, error) {
	return s.snapshotRepo.DeleteOrphans(ctx)
}
