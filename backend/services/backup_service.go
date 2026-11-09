package services

import (
	"context"
	"encoding/json"
	"fmt"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BackupService struct {
	db                *mongo.Database
	mfRepo            *repository.MutualFundRepo
	stockRepo         *repository.StockRepo
	fdRepo            *repository.FixedDepositRepo
	pfRepo            *repository.ProvidentFundRepo
	bondRepo          *repository.CorporateBondRepo
	homeLoanRepo      *repository.HomeLoanRepo
	personalLoanRepo  *repository.PersonalLoanRepo
	npsRepo           *repository.NPSRepo
	creditCardRepo    *repository.CreditCardRepo
	goalRepo          *repository.GoalRepo
	sipRepo           *repository.SIPRepo
	profileRepo       *repository.ProfileRepo
	altInvestmentRepo *repository.AlternateInvestmentRepo
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
	altInvestmentRepo *repository.AlternateInvestmentRepo,
) *BackupService {
	return &BackupService{
		db: db, mfRepo: mfRepo, stockRepo: stockRepo, fdRepo: fdRepo, pfRepo: pfRepo,
		bondRepo: bondRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		npsRepo: npsRepo, creditCardRepo: creditCardRepo, goalRepo: goalRepo,
		sipRepo: sipRepo, profileRepo: profileRepo, altInvestmentRepo: altInvestmentRepo,
	}
}

type BackupData struct {
	Version              string               `json:"version"`
	MutualFunds          interface{}          `json:"mutual_funds"`
	Stocks               interface{}          `json:"stocks"`
	FixedDeposits        interface{}          `json:"fixed_deposits"`
	ProvidentFund        interface{}          `json:"provident_fund"`
	CorporateBonds       interface{}          `json:"corporate_bonds"`
	HomeLoans            interface{}          `json:"home_loans"`
	PersonalLoans        interface{}          `json:"personal_loans"`
	NPS                  interface{}          `json:"nps"`
	CreditCards          interface{}          `json:"credit_cards"`
	Goals                interface{}          `json:"goals"`
	SIPs                 interface{}          `json:"sips"`
	Profiles             interface{}          `json:"profiles"`
	AlternateInvestments interface{}          `json:"vehicle_rentals"` // pre-rename key, kept so old and new backups restore alike
	LedgerEntries        []models.LedgerEntry `json:"ledger_entries"`

	// Personal finance (documents/GridFS files are not included in backups)
	BankAccounts         []models.BankAccount          `json:"bank_accounts"`
	Budgets              []models.Budget               `json:"budgets"`
	InsurancePolicies    []models.InsurancePolicy      `json:"insurance_policies"`
	GoldHoldings         []models.GoldHolding          `json:"gold_holdings"`
	Properties           []models.Property             `json:"properties"`
	ProfileAssignments   []models.ProfileAssignment    `json:"profile_assignments"`
	NotificationSettings []models.NotificationSettings `json:"notification_settings"`
	MemberIncomes        []models.MemberIncome         `json:"member_incomes"`
	TaxSettings          []models.TaxSettings          `json:"tax_settings"`
}

func (s *BackupService) Export(ctx context.Context, userID string) ([]byte, error) {
	mfs, _ := s.mfRepo.GetAll(ctx, userID)
	stocks, _ := s.stockRepo.GetAll(ctx, userID)
	fds, _ := s.fdRepo.GetAll(ctx, userID)
	pfs, _ := s.pfRepo.GetAll(ctx, userID)
	bonds, _ := s.bondRepo.GetAll(ctx, userID)
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	personalLoans, _ := s.personalLoanRepo.GetAll(ctx, userID)
	nps, _ := s.npsRepo.GetAll(ctx, userID)
	creditCards, _ := s.creditCardRepo.GetAll(ctx, userID)
	goals, _ := s.goalRepo.GetAll(ctx, userID)
	sips, _ := s.sipRepo.GetAll(ctx, userID)
	profiles, _ := s.profileRepo.GetAll(ctx, userID)
	altInvestments, _ := s.altInvestmentRepo.GetAll(ctx, userID)
	ledgerEntries, err := repository.NewLedgerRepo(s.db).GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("export ledger: %w", err)
	}

	bankAccounts, _ := repository.NewBankAccountRepo(s.db).GetAll(ctx, userID)
	budgets, _ := repository.NewBudgetRepo(s.db).GetAll(ctx, userID)
	policies, _ := repository.NewInsuranceRepo(s.db).GetAll(ctx, userID)
	gold, _ := repository.NewGoldRepo(s.db).GetAll(ctx, userID)
	properties, _ := repository.NewPropertyRepo(s.db).GetAll(ctx, userID)
	assignments, _ := repository.NewProfileAssignmentRepo(s.db).GetAll(ctx, userID)
	notifSettings, _ := repository.NewNotificationSettingsRepo(s.db).GetAll(ctx, userID)
	memberIncomes, _ := repository.NewMemberIncomeRepo(s.db).GetAll(ctx, userID)
	taxSettings, _ := repository.NewTaxSettingsRepo(s.db).GetAll(ctx, userID)

	backup := BackupData{
		Version:              "1.0",
		MutualFunds:          mfs,
		Stocks:               stocks,
		FixedDeposits:        fds,
		ProvidentFund:        pfs,
		CorporateBonds:       bonds,
		HomeLoans:            homeLoans,
		PersonalLoans:        personalLoans,
		NPS:                  nps,
		CreditCards:          creditCards,
		Goals:                goals,
		SIPs:                 sips,
		Profiles:             profiles,
		AlternateInvestments: altInvestments,
		LedgerEntries:        ledgerEntries,

		BankAccounts:         bankAccounts,
		Budgets:              budgets,
		InsurancePolicies:    policies,
		GoldHoldings:         gold,
		Properties:           properties,
		ProfileAssignments:   assignments,
		NotificationSettings: notifSettings,
		MemberIncomes:        memberIncomes,
		TaxSettings:          taxSettings,
	}

	return json.MarshalIndent(backup, "", "  ")
}

type RestoreResult struct {
	CollectionsRestored int      `json:"collections_restored"`
	Errors              []string `json:"errors,omitempty"`
}

// restoreTarget maps a backup key to its collection and typed model, so that
// IDs and dates are decoded into proper BSON types rather than raw strings.
type restoreTarget struct {
	key        string
	collection string
	decode     func(json.RawMessage) ([]interface{}, error)
}

func decodeAs[T any](raw json.RawMessage) ([]interface{}, error) {
	var items []T
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	out := make([]interface{}, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out, nil
}

var restoreTargets = []restoreTarget{
	{"ledger_entries", "ledger_entries", decodeLedgerBackup},
	{"mutual_funds", "mutual_funds", decodeAs[models.MutualFund]},
	{"stocks", "stocks", decodeAs[models.Stock]},
	{"fixed_deposits", "fixed_deposits", decodeAs[models.FixedDeposit]},
	{"provident_fund", "provident_fund_entries", decodeAs[models.ProvidentFund]},
	{"corporate_bonds", "corporate_bonds", decodeAs[models.CorporateBond]},
	{"home_loans", "home_loans", decodeAs[models.HomeLoan]},
	{"personal_loans", "personal_loans", decodeAs[models.PersonalLoan]},
	{"nps", "nps_accounts", decodeAs[models.NPSAccount]},
	{"credit_cards", "credit_cards", decodeAs[models.CreditCard]},
	{"goals", "goals", decodeAs[models.Goal]},
	{"sips", "sips", decodeAs[models.SIP]},
	{"profiles", "profiles", decodeAs[models.Profile]},
	{"vehicle_rentals", "vehicle_rentals", decodeAs[models.AlternateInvestment]},
	{"bank_accounts", "bank_accounts", decodeAs[models.BankAccount]},
	{"budgets", "budgets", decodeAs[models.Budget]},
	{"insurance_policies", "insurance_policies", decodeAs[models.InsurancePolicy]},
	{"gold_holdings", "gold_holdings", decodeAs[models.GoldHolding]},
	{"properties", "properties", decodeAs[models.Property]},
	{"profile_assignments", "profile_assignments", decodeAs[models.ProfileAssignment]},
	{"notification_settings", "notification_settings", decodeAs[models.NotificationSettings]},
	{"member_incomes", "member_incomes", decodeAs[models.MemberIncome]},
	{"tax_settings", "tax_settings", decodeAs[models.TaxSettings]},
}

// Restore replaces the given user's data with the contents of a backup JSON payload.
// Only documents owned by userID are removed, and every restored document is
// re-assigned to userID regardless of the user_id present in the backup.
func (s *BackupService) Restore(ctx context.Context, userID string, data []byte) (*RestoreResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("restore requires a user")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, invalidf("invalid backup JSON: %v", err)
	}

	result := &RestoreResult{}

	for _, t := range restoreTargets {
		rawData, ok := raw[t.key]
		if !ok || string(rawData) == "null" {
			continue
		}

		items, err := t.decode(rawData)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse %s: %v", t.key, err))
			continue
		}
		if len(items) == 0 {
			if t.key == "ledger_entries" {
				if _, err := s.db.Collection(t.collection).DeleteMany(ctx, bson.M{"user_id": userID}); err != nil {
					return nil, fmt.Errorf("clear ledger: %w", err)
				}
				result.CollectionsRestored++
			}
			continue
		}

		docs, err := ownedDocs(items, userID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to prepare %s: %v", t.key, err))
			continue
		}
		if t.key == "ledger_entries" {
			for _, doc := range docs {
				entry := doc.(bson.M)
				key, ok := entry["source_key"].(string)
				if !ok || !ValidLedgerSourceKey(key) {
					return nil, invalidf("invalid ledger source key in backup")
				}
				entry["_id"] = ledgerID(userID, key)
			}
		}

		coll := s.db.Collection(t.collection)

		if _, err := coll.DeleteMany(ctx, bson.M{"user_id": userID}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to clear %s: %v", t.key, err))
			continue
		}

		// Unordered so that a document whose _id collides with another user's
		// record is skipped without aborting the rest of the collection.
		if _, err := coll.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert some %s: %v", t.key, err))
			continue
		}
		result.CollectionsRestored++
	}

	return result, nil
}

// ownedDocs encodes typed models to BSON and forces user_id to userID.
func ownedDocs(items []interface{}, userID string) ([]interface{}, error) {
	docs := make([]interface{}, len(items))
	for i, item := range items {
		b, err := bson.Marshal(item)
		if err != nil {
			return nil, err
		}
		var doc bson.M
		if err := bson.Unmarshal(b, &doc); err != nil {
			return nil, err
		}
		doc["user_id"] = userID
		docs[i] = doc
	}
	return docs, nil
}
