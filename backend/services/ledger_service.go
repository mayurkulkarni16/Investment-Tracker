package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type LedgerService struct {
	repo        *repository.LedgerRepo
	profiles    *repository.ProfileRepo
	accounts    *repository.BankAccountRepo
	cards       *repository.CreditCardRepo
	assignments *repository.ProfileAssignmentRepo
}

func NewLedgerService(repo *repository.LedgerRepo, profiles *repository.ProfileRepo, accounts *repository.BankAccountRepo, cards *repository.CreditCardRepo, assignments *repository.ProfileAssignmentRepo) *LedgerService {
	return &LedgerService{repo: repo, profiles: profiles, accounts: accounts, cards: cards, assignments: assignments}
}

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

func ledgerAttribution(req models.LedgerRequest, previous *models.LedgerEntry) models.LedgerAttribution {
	var a models.LedgerAttribution
	if previous != nil {
		a = previous.LedgerAttribution
	}
	if req.ProfileID != nil {
		a.ProfileID = *req.ProfileID
	}
	if req.LinkedAccountType != nil {
		a.LinkedAccountType = *req.LinkedAccountType
	}
	if req.LinkedAccountID != nil {
		a.LinkedAccountID = *req.LinkedAccountID
	}
	if req.SMSReceivedAt != nil {
		a.SMSReceivedAt = *req.SMSReceivedAt
	}
	return a
}

func validateLedgerAttribution(a models.LedgerAttribution, source string) error {
	if a.ProfileID != "" {
		if _, err := primitive.ObjectIDFromHex(a.ProfileID); err != nil {
			return invalidf("invalid family member")
		}
	}
	if (a.LinkedAccountType == "") != (a.LinkedAccountID == "") {
		return invalidf("linked account type and ID must be provided together")
	}
	if a.LinkedAccountID != "" {
		if a.LinkedAccountType != "bank_account" && a.LinkedAccountType != "credit_card" {
			return invalidf("linked account must be a bank account or credit card")
		}
		if _, err := primitive.ObjectIDFromHex(a.LinkedAccountID); err != nil {
			return invalidf("invalid linked account")
		}
	}
	if a.SMSReceivedAt != "" {
		if source != "sms" {
			return invalidf("SMS receipt time can only be recorded for SMS imports")
		}
		if _, err := time.Parse(time.RFC3339, a.SMSReceivedAt); err != nil {
			return invalidf("SMS receipt time must include a valid date, time and time zone")
		}
	}
	return nil
}

func (s *LedgerService) validateReferences(ctx context.Context, userID string, req models.LedgerRequest, a models.LedgerAttribution, previous *models.LedgerEntry) error {
	if a.ProfileID != "" && (previous == nil || a.ProfileID != previous.ProfileID) {
		id, _ := primitive.ObjectIDFromHex(a.ProfileID)
		profile, err := s.profiles.GetByID(ctx, id)
		if errors.Is(err, mongo.ErrNoDocuments) || (err == nil && profile.UserID != userID) {
			return invalidf("family member not found")
		}
		if err != nil {
			return fmt.Errorf("check ledger family member: %w", err)
		}
	}
	accountChanged := previous == nil || a.LinkedAccountID != previous.LinkedAccountID || a.LinkedAccountType != previous.LinkedAccountType
	suffixChanged := previous != nil && req.Account != previous.Account
	if a.LinkedAccountID == "" || (!accountChanged && !suffixChanged) {
		return nil
	}
	id, _ := primitive.ObjectIDFromHex(a.LinkedAccountID)
	var owner, lastFour string
	var err error
	if a.LinkedAccountType == "bank_account" {
		var account *models.BankAccount
		account, err = s.accounts.GetByID(ctx, id)
		if err == nil {
			owner, lastFour = account.UserID, account.LastFour
		}
	} else {
		var card *models.CreditCard
		card, err = s.cards.GetByID(ctx, id)
		if err == nil {
			owner, lastFour = card.UserID, card.LastFourDigits
		}
	}
	if errors.Is(err, mongo.ErrNoDocuments) || (err == nil && owner != userID) {
		return invalidf("linked account not found; unlink it or choose another account")
	}
	if err != nil {
		return fmt.Errorf("check linked ledger account: %w", err)
	}
	if lastFour != "" && req.Account != lastFour {
		return invalidf("last four digits do not match the linked account")
	}
	return nil
}

// Context returns only the account details needed to attribute ledger entries.
func (s *LedgerService) Context(ctx context.Context, userID string) (*models.LedgerContext, error) {
	if userID == "" {
		return nil, invalidf("user is required")
	}
	profiles, err := s.profiles.GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load ledger members: %w", err)
	}
	accounts, err := s.accounts.GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load ledger accounts: %w", err)
	}
	cards, err := s.cards.GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load ledger cards: %w", err)
	}
	assignments, err := s.assignments.GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load ledger ownership: %w", err)
	}

	out := &models.LedgerContext{Members: []models.LedgerMember{}, Accounts: []models.LedgerAccount{}}
	members := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		id := profile.ID.Hex()
		members[id] = true
		out.Members = append(out.Members, models.LedgerMember{ID: id, Name: profile.Name})
	}
	owners := make(map[string][]string)
	for _, assignment := range assignments {
		if assignment.AssetType != "bank_account" && assignment.AssetType != "credit_card" {
			continue
		}
		key := assignment.AssetType + ":" + assignment.AssetID
		seen := map[string]bool{}
		for _, share := range assignment.EffectiveShares() {
			if members[share.ProfileID] && !seen[share.ProfileID] {
				owners[key] = append(owners[key], share.ProfileID)
				seen[share.ProfileID] = true
			}
		}
	}
	add := func(kind, id, name, bank, lastFour, status string) {
		if !ledgerAccountPattern.MatchString(lastFour) {
			lastFour = ""
		}
		memberIDs := owners[kind+":"+id]
		if memberIDs == nil {
			memberIDs = []string{}
		}
		out.Accounts = append(out.Accounts, models.LedgerAccount{
			Type: kind, ID: id, Name: name, Bank: bank, LastFour: lastFour, Status: status, MemberIDs: memberIDs,
		})
	}
	for _, account := range accounts {
		add("bank_account", account.ID.Hex(), account.Name, account.Institution, account.LastFour, account.Status)
	}
	for _, card := range cards {
		add("credit_card", card.ID.Hex(), card.CardName, card.BankName, card.LastFourDigits, card.Status)
	}
	sort.SliceStable(out.Members, func(i, j int) bool { return out.Members[i].Name < out.Members[j].Name })
	sort.SliceStable(out.Accounts, func(i, j int) bool { return out.Accounts[i].Name < out.Accounts[j].Name })
	return out, nil
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
	attribution := ledgerAttribution(req, nil)
	if err := validateLedgerAttribution(attribution, req.Source); err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, userID, req, attribution, nil); err != nil {
		return nil, err
	}
	now := time.Now()
	return s.repo.Create(ctx, &models.LedgerEntry{
		ID: ledgerID(userID, req.SourceKey), UserID: userID, SourceKey: req.SourceKey, Source: req.Source,
		Status: req.Status, Kind: req.Kind, AmountPaise: req.AmountPaise, Date: req.Date,
		Description: strings.TrimSpace(req.Description), Category: req.Category,
		Bank: req.Bank, Account: req.Account, DueDate: req.DueDate, LedgerAttribution: attribution, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *LedgerService) Update(ctx context.Context, userID, id string, req models.LedgerRequest) (*models.LedgerEntry, error) {
	if userID == "" {
		return nil, invalidf("user is required")
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, notFoundf("ledger entry")
	}
	if err := validateLedger(req, false); err != nil {
		return nil, err
	}
	previous, err := s.repo.GetByID(ctx, userID, oid)
	if err != nil {
		return nil, err
	}
	attribution := ledgerAttribution(req, previous)
	if err := validateLedgerAttribution(attribution, previous.Source); err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, userID, req, attribution, previous); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, oid, req, attribution, time.Now())
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
		if err := validateLedgerAttribution(entry.LedgerAttribution, entry.Source); err != nil {
			return nil, err
		}
		items[i] = entry
	}
	return items, nil
}
