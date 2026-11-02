package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func validLedgerRequest() models.LedgerRequest {
	return models.LedgerRequest{
		SourceKey: strings.Repeat("a", 64), Source: "manual", Status: "confirmed",
		Kind: "expense", AmountPaise: 125050, Date: "2026-10-21",
		Description: "Groceries", Category: "Food", Bank: "HDFC", Account: "1234",
	}
}

func TestLedgerValidation(t *testing.T) {
	tests := []struct {
		name string
		edit func(*models.LedgerRequest)
	}{
		{"zero", func(r *models.LedgerRequest) { r.AmountPaise = 0 }},
		{"negative", func(r *models.LedgerRequest) { r.AmountPaise = -1 }},
		{"above limit", func(r *models.LedgerRequest) { r.AmountPaise = 1_000_000_000_001 }},
		{"invalid date", func(r *models.LedgerRequest) { r.Date = "2026-02-30" }},
		{"invalid due date", func(r *models.LedgerRequest) { r.DueDate = "tomorrow" }},
		{"full account", func(r *models.LedgerRequest) { r.Account = "123456789012" }},
		{"blank description", func(r *models.LedgerRequest) { r.Description = "  " }},
		{"oversize description", func(r *models.LedgerRequest) { r.Description = strings.Repeat("a", 201) }},
		{"unknown status", func(r *models.LedgerRequest) { r.Status = "paid" }},
		{"unknown kind", func(r *models.LedgerRequest) { r.Kind = "debit" }},
		{"invalid source key", func(r *models.LedgerRequest) { r.SourceKey = "short" }},
		{"unknown source", func(r *models.LedgerRequest) { r.Source = "external" }},
		{"unreviewed sms", func(r *models.LedgerRequest) { r.Source = "sms" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validLedgerRequest()
			tc.edit(&req)
			if err := validateLedger(req, true); !IsInvalidInput(err) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
	for _, kind := range []string{"income", "expense", "refund", "bill", "transfer"} {
		req := validLedgerRequest()
		req.Kind = kind
		if err := validateLedger(req, true); err != nil {
			t.Fatalf("%s rejected: %v", kind, err)
		}
		req.Source, req.Status = "sms", "pending"
		if err := validateLedger(req, true); err != nil {
			t.Fatalf("pending SMS rejected: %v", err)
		}
		req.Status = "confirmed"
		if err := validateLedger(req, false); err != nil {
			t.Fatalf("reviewed SMS update rejected: %v", err)
		}
	}
}

func TestLedgerIdentityIsUserBoundAndRepeatable(t *testing.T) {
	key := strings.Repeat("b", 64)
	if ledgerID("user1", key) != ledgerID("user1", key) {
		t.Fatal("rescan must retain identity")
	}
	if ledgerID("user1", key) == ledgerID("user2", key) {
		t.Fatal("identity must be scoped to user")
	}
	if ledgerID("user1", key) == ledgerID("user1", strings.Repeat("c", 64)) {
		t.Fatal("different messages must have separate identities")
	}
}

func TestLedgerBackupPreservesReviewedSMSAndValidatesAmounts(t *testing.T) {
	entry := models.LedgerEntry{
		SourceKey: strings.Repeat("a", 64), Source: "sms", Status: "confirmed",
		Kind: "expense", AmountPaise: 100, Date: "2026-10-21", Description: "Test",
		LedgerAttribution: models.LedgerAttribution{
			ProfileID: primitive.NewObjectID().Hex(), LinkedAccountType: "bank_account",
			LinkedAccountID: primitive.NewObjectID().Hex(), SMSReceivedAt: "2026-10-21T10:00:00Z",
		},
	}
	raw, err := json.Marshal([]models.LedgerEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	items, err := decodeLedgerBackup(raw)
	if err != nil || len(items) != 1 {
		t.Fatalf("reviewed SMS backup rejected: %v", err)
	}
	if got := items[0].(*models.LedgerEntry).LedgerAttribution; got != entry.LedgerAttribution {
		t.Fatalf("backup lost attribution: %+v", got)
	}
	entry.SMSReceivedAt = "yesterday"
	raw, err = json.Marshal([]models.LedgerEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeLedgerBackup(raw); !IsInvalidInput(err) {
		t.Fatalf("invalid backup timestamp accepted: %v", err)
	}
	entry.SMSReceivedAt = "2026-10-21T10:00:00Z"
	entry.AmountPaise = -1
	raw, err = json.Marshal([]models.LedgerEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeLedgerBackup(raw); !IsInvalidInput(err) {
		t.Fatalf("invalid backup amount accepted: %v", err)
	}
}

func TestLedgerAttributionValidationAndLegacyUpdates(t *testing.T) {
	profileID, accountID := primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex()
	original := models.LedgerEntry{LedgerAttribution: models.LedgerAttribution{
		ProfileID: profileID, LinkedAccountType: "bank_account", LinkedAccountID: accountID,
		SMSReceivedAt: "2026-10-21T10:00:00Z",
	}}
	if got := ledgerAttribution(models.LedgerRequest{}, &original); got != original.LedgerAttribution {
		t.Fatalf("legacy update cleared attribution: %+v", got)
	}
	empty := ""
	req := models.LedgerRequest{ProfileID: &empty, LinkedAccountType: &empty, LinkedAccountID: &empty}
	cleared := ledgerAttribution(req, &original)
	if cleared.ProfileID != "" || cleared.LinkedAccountType != "" || cleared.LinkedAccountID != "" || cleared.SMSReceivedAt != original.SMSReceivedAt {
		t.Fatalf("explicit clear failed: %+v", cleared)
	}
	if err := validateLedgerAttribution(original.LedgerAttribution, "sms"); err != nil {
		t.Fatalf("valid attribution rejected: %v", err)
	}
	cases := []struct {
		name   string
		value  models.LedgerAttribution
		source string
	}{
		{"invalid profile", models.LedgerAttribution{ProfileID: "other"}, "manual"},
		{"missing account ID", models.LedgerAttribution{LinkedAccountType: "bank_account"}, "manual"},
		{"missing account type", models.LedgerAttribution{LinkedAccountID: accountID}, "manual"},
		{"invalid account type", models.LedgerAttribution{LinkedAccountType: "loan", LinkedAccountID: accountID}, "manual"},
		{"invalid account ID", models.LedgerAttribution{LinkedAccountType: "credit_card", LinkedAccountID: "other"}, "manual"},
		{"invalid timestamp", models.LedgerAttribution{SMSReceivedAt: "2026-10-21"}, "sms"},
		{"manual with SMS time", models.LedgerAttribution{SMSReceivedAt: "2026-10-21T10:00:00Z"}, "manual"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateLedgerAttribution(tc.value, tc.source); !IsInvalidInput(err) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
	if err := validateLedgerAttribution(models.LedgerAttribution{
		ProfileID: profileID, LinkedAccountType: "bank_account", LinkedAccountID: accountID,
	}, "manual"); err != nil {
		t.Fatalf("manual ledger assignment rejected: %v", err)
	}
}

func TestLedgerAttributionIsFlattenedInJSONAndBSON(t *testing.T) {
	entry := models.LedgerEntry{LedgerAttribution: models.LedgerAttribution{
		ProfileID: primitive.NewObjectID().Hex(), LinkedAccountType: "credit_card",
		LinkedAccountID: primitive.NewObjectID().Hex(), SMSReceivedAt: "2026-10-21T10:00:00Z",
	}}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	var jsonDoc map[string]interface{}
	if err := json.Unmarshal(raw, &jsonDoc); err != nil {
		t.Fatal(err)
	}
	bsonRaw, err := bson.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	var bsonDoc bson.M
	if err := bson.Unmarshal(bsonRaw, &bsonDoc); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"profile_id", "linked_account_type", "linked_account_id", "sms_received_at"} {
		if jsonDoc[field] == nil || bsonDoc[field] == nil {
			t.Fatalf("%s is not flattened: JSON=%v BSON=%v", field, jsonDoc, bsonDoc)
		}
	}
}

func TestAccountTotalsRespectExplicitLedgerLinks(t *testing.T) {
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	accountID, otherID := primitive.NewObjectID(), primitive.NewObjectID()
	acc := models.BankAccount{ID: accountID, LastFour: "1234"}
	entries := []models.LedgerEntry{
		{Status: "confirmed", Kind: "expense", Date: "2026-10-01", Account: "1234", AmountPaise: 500, LedgerAttribution: models.LedgerAttribution{LinkedAccountType: "bank_account", LinkedAccountID: accountID.Hex()}},
		{Status: "confirmed", Kind: "expense", Date: "2026-10-02", Account: "1234", AmountPaise: 300, LedgerAttribution: models.LedgerAttribution{LinkedAccountType: "bank_account", LinkedAccountID: otherID.Hex()}},
		{Status: "confirmed", Kind: "expense", Date: "2026-10-03", Account: "1234", AmountPaise: 400, LedgerAttribution: models.LedgerAttribution{LinkedAccountType: "credit_card", LinkedAccountID: otherID.Hex()}},
		{Status: "confirmed", Kind: "expense", Date: "2026-10-04", Account: "1234", AmountPaise: 200},
		{Status: "pending", Kind: "expense", Date: "2026-10-05", Account: "1234", AmountPaise: 100},
	}
	(&AccountService{}).decorate(&acc, entries, now)
	if acc.MonthOutflow != 5 {
		t.Fatalf("expected explicitly linked expenses only, got %v", acc.MonthOutflow)
	}
	cash := models.BankAccount{ID: otherID}
	entries = []models.LedgerEntry{{Status: "confirmed", Kind: "income", Date: "2026-10-01", AmountPaise: 12500,
		LedgerAttribution: models.LedgerAttribution{LinkedAccountType: "bank_account", LinkedAccountID: otherID.Hex()}}}
	(&AccountService{}).decorate(&cash, entries, now)
	if cash.MonthInflow != 125 {
		t.Fatalf("explicit account link without last four was ignored: %v", cash.MonthInflow)
	}
}
