package services

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestLedgerAttributionIntegration(t *testing.T) {
	uri := os.Getenv("LEDGER_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set LEDGER_TEST_MONGO_URI to run the MongoDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetServerSelectionTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database("finboard_ledger_test_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := db.Drop(cleanup); err != nil {
			t.Errorf("drop test database %s: %v", db.Name(), err)
		}
		if err := client.Disconnect(cleanup); err != nil {
			t.Errorf("disconnect test database: %v", err)
		}
	})
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatal(err)
	}

	profiles := repository.NewProfileRepo(db)
	accounts := repository.NewBankAccountRepo(db)
	cards := repository.NewCreditCardRepo(db)
	assignments := repository.NewProfileAssignmentRepo(db)
	ledger := repository.NewLedgerRepo(db)
	service := NewLedgerService(ledger, profiles, accounts, cards, assignments)
	const owner = "ledger-owner-a"
	const otherOwner = "ledger-owner-b"
	member := &models.Profile{UserID: owner, Name: "Alice", Relationship: "self"}
	otherMember := &models.Profile{UserID: otherOwner, Name: "Other", Relationship: "self"}
	if err := profiles.Create(ctx, member); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Create(ctx, otherMember); err != nil {
		t.Fatal(err)
	}
	accountID, err := accounts.Insert(ctx, &models.BankAccount{UserID: owner, Name: "Savings", Institution: "HDFC Bank", LastFour: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	otherAccountID, err := accounts.Insert(ctx, &models.BankAccount{UserID: otherOwner, Name: "Private", LastFour: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	card := &models.CreditCard{UserID: owner, CardName: "Card", BankName: "HDFC", LastFourDigits: "9999"}
	if err := cards.Create(ctx, card); err != nil {
		t.Fatal(err)
	}
	if _, err := assignments.Insert(ctx, &models.ProfileAssignment{
		UserID: owner, AssetType: "bank_account", AssetID: accountID.Hex(), ProfileID: member.ID.Hex(),
	}); err != nil {
		t.Fatal(err)
	}

	ownerCtx := repository.WithOwner(ctx, owner)
	lookup, err := service.Context(ownerCtx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Members) != 1 || lookup.Members[0].ID != member.ID.Hex() || len(lookup.Accounts) != 2 {
		t.Fatalf("wrong or unscoped lookup: %+v", lookup)
	}
	var matched bool
	for _, item := range lookup.Accounts {
		if item.ID == accountID.Hex() {
			matched = len(item.MemberIDs) == 1 && item.MemberIDs[0] == member.ID.Hex() && item.LastFour == "1234"
		}
	}
	if !matched {
		t.Fatalf("account ownership not returned: %+v", lookup.Accounts)
	}

	memberID, accountHex, kind, received := member.ID.Hex(), accountID.Hex(), "bank_account", "2026-10-21T10:00:00Z"
	req := validLedgerRequest()
	req.Source, req.Status = "sms", "pending"
	req.ProfileID, req.LinkedAccountType, req.LinkedAccountID, req.SMSReceivedAt = &memberID, &kind, &accountHex, &received
	created, err := service.Create(ownerCtx, owner, req)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := service.Create(ownerCtx, owner, req)
	if err != nil || repeated.ID != created.ID {
		t.Fatalf("SMS rescan not idempotent: entry=%+v err=%v", repeated, err)
	}
	all, err := service.GetAll(ownerCtx, owner)
	if err != nil || len(all) != 1 {
		t.Fatalf("unexpected stored entries: %+v, %v", all, err)
	}

	bad := req
	bad.SourceKey = strings.Repeat("b", 64)
	foreignMember := otherMember.ID.Hex()
	bad.ProfileID = &foreignMember
	if _, err := service.Create(ownerCtx, owner, bad); !IsInvalidInput(err) {
		t.Fatalf("foreign profile accepted: %v", err)
	}
	bad.ProfileID = &memberID
	foreignAccount := otherAccountID.Hex()
	bad.LinkedAccountID = &foreignAccount
	if _, err := service.Create(ownerCtx, owner, bad); !IsInvalidInput(err) {
		t.Fatalf("foreign account accepted: %v", err)
	}
	bad.LinkedAccountID = &accountHex
	bad.Account = "0000"
	if _, err := service.Create(ownerCtx, owner, bad); !IsInvalidInput(err) {
		t.Fatalf("mismatched suffix accepted: %v", err)
	}
	bad.Account = ""
	if _, err := service.Create(ownerCtx, owner, bad); !IsInvalidInput(err) {
		t.Fatalf("cleared suffix on linked account accepted: %v", err)
	}

	legacy := req
	legacy.Status = "confirmed"
	legacy.ProfileID, legacy.LinkedAccountType, legacy.LinkedAccountID, legacy.SMSReceivedAt = nil, nil, nil, nil
	updated, err := service.Update(ownerCtx, owner, created.ID.Hex(), legacy)
	if err != nil || updated.ProfileID != memberID || updated.LinkedAccountID != accountHex || updated.SMSReceivedAt != received {
		t.Fatalf("legacy edit lost attribution: %+v, %v", updated, err)
	}
	otherCtx := repository.WithOwner(ctx, otherOwner)
	if _, err := service.Update(otherCtx, otherOwner, created.ID.Hex(), legacy); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("foreign ledger edit should fail: %v", err)
	}

	if err := profiles.Delete(ownerCtx, member.ID); err != nil {
		t.Fatal(err)
	}
	if err := accounts.Delete(ownerCtx, accountID); err != nil {
		t.Fatal(err)
	}
	updated, err = service.Update(ownerCtx, owner, created.ID.Hex(), legacy)
	if err != nil || updated.ProfileID != memberID || updated.LinkedAccountID != accountHex {
		t.Fatalf("historical links were lost after deletion: %+v, %v", updated, err)
	}
	legacy.Account = "0000"
	if _, err := service.Update(ownerCtx, owner, created.ID.Hex(), legacy); !IsInvalidInput(err) {
		t.Fatalf("changing suffix of deleted linked account should fail: %v", err)
	}
}
