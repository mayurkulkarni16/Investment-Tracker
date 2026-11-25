package main

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMigrationDryRunApplyAndRerun(t *testing.T) {
	uri := os.Getenv("ALT_PAYOUT_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set ALT_PAYOUT_TEST_MONGO_URI to run the MongoDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)
	db := client.Database("test_alt_payout_" + primitive.NewObjectID().Hex())
	defer db.Drop(ctx)
	t.Setenv("MONGO_URI", uri)
	t.Setenv("DB_NAME", db.Name())
	repo := repository.NewAlternateInvestmentRepo(db)
	day := func(y int, month time.Month, d int) time.Time {
		return time.Date(y, month, d, 0, 0, 0, 0, time.UTC)
	}
	first, second := day(2026, 12, 22), day(2027, 1, 22)
	received := day(2026, 12, 24)
	active := models.AlternateInvestment{
		UserID: "test-owner", PurchaseDate: day(2026, 11, 22), PayoutStartDate: first,
		AssetType: models.AltAssetLivestock, Units: 1, MonthlyPayoutPerUnit: 20000, TenureMonths: 2,
		Payouts: []models.AltPayout{
			{PayoutID: "first", ScheduledDate: first, Amount: 20000, Status: models.PayoutReceived, ReceivedAmount: 19000, ReceivedDate: &received},
			{PayoutID: "second", ScheduledDate: second, Amount: 20000, Status: models.PayoutPending},
		},
	}
	completed := active
	completed.Payouts = append([]models.AltPayout(nil), active.Payouts...)
	completed.Payouts[1].Status = models.PayoutReceived
	for _, v := range []*models.AlternateInvestment{&active, &completed} {
		if err := repo.Create(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	read := func(id primitive.ObjectID) *models.AlternateInvestment {
		t.Helper()
		v, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	stale := read(active.ID)
	originalCompleted := read(completed.ID)
	if err := run(false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stale, read(active.ID)) {
		t.Fatal("dry run changed the record")
	}
	if err := run(true); err != nil {
		t.Fatal(err)
	}
	got := read(active.ID)
	if got.ScheduleVersion != models.AltCalendarScheduleVersion || !got.PayoutStartDate.Equal(day(2027, 1, 5)) ||
		len(got.Payouts) != 3 || !reflect.DeepEqual(got.Payouts[0], stale.Payouts[0]) ||
		got.Payouts[1].PayoutID != "second" || !got.Payouts[1].ScheduledDate.Equal(day(2027, 2, 5)) ||
		!got.Payouts[2].ScheduledDate.Equal(day(2027, 3, 5)) || got.Payouts[2].Amount != 4285.71 {
		t.Fatalf("unexpected migration result: version=%d, first date=%v, payouts=%+v", got.ScheduleVersion, got.PayoutStartDate, got.Payouts)
	}
	if !reflect.DeepEqual(originalCompleted, read(completed.ID)) {
		t.Fatal("a completed investment was changed")
	}
	if err := repo.Update(ctx, stale); !errors.Is(err, repository.ErrAlternateInvestmentChanged) {
		t.Fatalf("stale edit error = %v, want a conflict", err)
	}
	if err := run(true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, read(active.ID)) {
		t.Fatal("rerun changed an already migrated record")
	}
}
