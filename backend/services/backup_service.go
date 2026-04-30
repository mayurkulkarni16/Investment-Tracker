package services

import (
	"context"
	"encoding/json"
	"fmt"

	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// restoreDoc is a generic BSON document for restore operations.
type restoreDoc = bson.M

type BackupService struct {
	db               *mongo.Database
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
	db *mongo.Database,
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
		db: db, mfRepo: mfRepo, stockRepo: stockRepo, fdRepo: fdRepo, pfRepo: pfRepo,
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

type RestoreResult struct {
	CollectionsRestored int      `json:"collections_restored"`
	Errors              []string `json:"errors,omitempty"`
}

// Restore replaces all data from a backup JSON payload.
func (s *BackupService) Restore(ctx context.Context, data []byte) (*RestoreResult, error) {
	// Parse backup into a generic map to handle each collection
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid backup JSON: %w", err)
	}

	// Map backup keys to MongoDB collection names
	collectionMap := map[string]string{
		"mutual_funds":    "mutual_funds",
		"stocks":          "stocks",
		"fixed_deposits":  "fixed_deposits",
		"provident_fund":  "provident_fund_entries",
		"corporate_bonds": "corporate_bonds",
		"home_loans":      "home_loans",
		"personal_loans":  "personal_loans",
		"nps":             "nps_accounts",
		"credit_cards":    "credit_cards",
		"goals":           "goals",
		"sips":            "sips",
		"profiles":        "profiles",
	}

	result := &RestoreResult{}

	for key, collName := range collectionMap {
		rawData, ok := raw[key]
		if !ok || string(rawData) == "null" {
			continue
		}

		// Decode as array of generic BSON documents
		var docs []restoreDoc
		if err := json.Unmarshal(rawData, &docs); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse %s: %v", key, err))
			continue
		}

		if len(docs) == 0 {
			continue
		}

		coll := s.db.Collection(collName)

		// Drop existing data
		if _, err := coll.DeleteMany(ctx, bson.M{}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to clear %s: %v", key, err))
			continue
		}

		// Insert restored documents
		inserts := make([]interface{}, len(docs))
		for i := range docs {
			inserts[i] = docs[i]
		}
		if _, err := coll.InsertMany(ctx, inserts); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert %s: %v", key, err))
			continue
		}
		result.CollectionsRestored++
	}

	return result, nil
}
