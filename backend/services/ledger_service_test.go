package services

import (
	"encoding/json"
	"strings"
	"testing"

	"investment-tracker/models"
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
	}
	raw, err := json.Marshal([]models.LedgerEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	items, err := decodeLedgerBackup(raw)
	if err != nil || len(items) != 1 {
		t.Fatalf("reviewed SMS backup rejected: %v", err)
	}
	entry.AmountPaise = -1
	raw, err = json.Marshal([]models.LedgerEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeLedgerBackup(raw); !IsInvalidInput(err) {
		t.Fatalf("invalid backup amount accepted: %v", err)
	}
}
