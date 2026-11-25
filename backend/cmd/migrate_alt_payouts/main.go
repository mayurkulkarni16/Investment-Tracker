package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
	"investment-tracker/services"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	apply := flag.Bool("apply", false, "write changes (default: dry run)")
	flag.Parse()
	if err := run(*apply); err != nil {
		log.Fatal(err)
	}
}

func run(apply bool) error {
	uri, db := os.Getenv("MONGO_URI"), os.Getenv("DB_NAME")
	if uri == "" || db == "" {
		return fmt.Errorf("set MONGO_URI and DB_NAME explicitly before running this migration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("connect to MongoDB: %w", err)
	}
	defer client.Disconnect(context.Background())

	repo := repository.NewAlternateInvestmentRepo(client.Database(db))
	items, err := repo.GetAll(ctx, "")
	if err != nil {
		return fmt.Errorf("read investments: %w", err)
	}
	now := time.Now()
	changes := make([]*models.AlternateInvestment, 0, len(items))
	issues := 0
	for i := range items {
		v := &items[i]
		before := *v
		changed, err := services.UpgradeAlternateInvestmentSchedule(v, now)
		if err != nil {
			issues++
			log.Printf("%s needs review: %v", v.ID.Hex(), err)
			continue
		}
		if !changed {
			continue
		}
		report(&before, v, now)
		changes = append(changes, v)
	}
	log.Printf("records=%d changed=%d needs_review=%d apply=%t", len(items), len(changes), issues, apply)
	if issues != 0 {
		return fmt.Errorf("resolve the reported records before applying; no changes written")
	}
	if apply {
		for i, v := range changes {
			if err := repo.UpdateSchedule(ctx, v); err != nil {
				return fmt.Errorf("update investment %s after %d successful writes: %w (rerun after resolving)", v.ID.Hex(), i, err)
			}
		}
		log.Printf("updated=%d", len(changes))
	}
	return nil
}

func report(before, after *models.AlternateInvestment, now time.Time) {
	const day = "2006-01-02"
	log.Printf("%s first payout %s -> %s", after.ID.Hex(), before.PayoutStartDate.Format(day), after.PayoutStartDate.Format(day))
	for i, p := range after.Payouts {
		if i >= len(before.Payouts) {
			log.Printf("  installment %d added: %s %.2f", i+1, p.ScheduledDate.Format(day), p.Amount)
		} else if old := before.Payouts[i]; !old.ScheduledDate.Equal(p.ScheduledDate) || old.Amount != p.Amount {
			log.Printf("  installment %d: %s %.2f -> %s %.2f", i+1, old.ScheduledDate.Format(day), old.Amount, p.ScheduledDate.Format(day), p.Amount)
		}
	}
	if after.TenureMonths > 0 {
		preview := *after
		// Computed fields also set overdue flags, so keep the persisted rows untouched.
		preview.Payouts = append([]models.AltPayout(nil), after.Payouts...)
		services.ComputeAlternateInvestment(&preview, now)
		oldEnd := before.Payouts[len(before.Payouts)-1].ScheduledDate
		log.Printf("  agreement end (also pending buyback if applicable): %s -> %s", oldEnd.Format(day), preview.AgreementEndDate.Format(day))
	}
}
