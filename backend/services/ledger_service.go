package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LedgerService struct{ repo *repository.LedgerRepo }

func NewLedgerService(repo *repository.LedgerRepo) *LedgerService { return &LedgerService{repo: repo} }

var ledgerKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var ledgerAccountPattern = regexp.MustCompile(`^([0-9]{4})?$`)

func validateLedger(req models.LedgerRequest, creating bool) error {
	if req.Kind != "income" && req.Kind != "expense" && req.Kind != "refund" && req.Kind != "transfer" && req.Kind != "bill" {
		return invalidf("kind must be income, expense, refund, transfer or bill")
	}
	if req.Status != "pending" && req.Status != "confirmed" && req.Status != "rejected" {
		return invalidf("invalid ledger status")
	}
	if req.AmountPaise <= 0 || req.AmountPaise > 1_000_000_000_000 {
		return invalidf("amount must be positive and at most INR 10 billion")
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return invalidf("date must be YYYY-MM-DD")
	}
	if req.DueDate != "" {
		if _, err := time.Parse("2006-01-02", req.DueDate); err != nil {
			return invalidf("due date must be YYYY-MM-DD")
		}
	}
	if strings.TrimSpace(req.Description) == "" || len(req.Description) > 200 || len(req.Category) > 60 || len(req.Bank) > 60 {
		return invalidf("description is required (max 200 characters); category and bank max 60")
	}
	if !ledgerAccountPattern.MatchString(req.Account) {
		return invalidf("account must be empty or last four digits only")
	}
	if creating {
		if !ledgerKeyPattern.MatchString(req.SourceKey) || (req.Source != "manual" && req.Source != "sms") {
			return invalidf("invalid source or source key")
		}
		if req.Source == "sms" && req.Status != "pending" {
			return invalidf("SMS imports must be reviewed before confirmation")
		}
	}
	return nil
}

func ledgerID(userID, sourceKey string) primitive.ObjectID {
	hash := sha256.Sum256([]byte(userID + "\x00" + sourceKey))
	var id primitive.ObjectID
	copy(id[:], hash[:12])
	return id
}

func (s *LedgerService) GetAll(ctx context.Context, userID string) ([]models.LedgerEntry, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *LedgerService) Create(ctx context.Context, userID string, req models.LedgerRequest) (*models.LedgerEntry, error) {
	if userID == "" {
		return nil, invalidf("user is required")
	}
	if err := validateLedger(req, true); err != nil {
		return nil, err
	}
	now := time.Now()
	return s.repo.Create(ctx, &models.LedgerEntry{
		ID: ledgerID(userID, req.SourceKey), UserID: userID, SourceKey: req.SourceKey, Source: req.Source,
		Status: req.Status, Kind: req.Kind, AmountPaise: req.AmountPaise, Date: req.Date,
		Description: strings.TrimSpace(req.Description), Category: req.Category,
		Bank: req.Bank, Account: req.Account, DueDate: req.DueDate, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *LedgerService) Update(ctx context.Context, userID, id string, req models.LedgerRequest) (*models.LedgerEntry, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, notFoundf("ledger entry")
	}
	if err := validateLedger(req, false); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, oid, req, time.Now())
}

// Keep fingerprints portable across backup/restore and devices.
func ValidLedgerSourceKey(key string) bool {
	return ledgerKeyPattern.MatchString(key)
}

func decodeLedgerBackup(raw json.RawMessage) ([]interface{}, error) {
	var entries []models.LedgerEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	items := make([]interface{}, len(entries))
	for i := range entries {
		entry := &entries[i]
		if !ValidLedgerSourceKey(entry.SourceKey) || (entry.Source != "manual" && entry.Source != "sms") {
			return nil, invalidf("invalid ledger source in backup")
		}
		if err := validateLedger(models.LedgerRequest{
			Status: entry.Status, Kind: entry.Kind, AmountPaise: entry.AmountPaise,
			Date: entry.Date, Description: entry.Description, Category: entry.Category,
			Bank: entry.Bank, Account: entry.Account, DueDate: entry.DueDate,
		}, false); err != nil {
			return nil, err
		}
		items[i] = entry
	}
	return items, nil
}
