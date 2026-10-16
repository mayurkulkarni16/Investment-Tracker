// One-time migration: backfill units on SIP installments where units=0 and nav>0.
// Run once from the backend directory:
//
//	go run ./cmd/migrate_sip_units/main.go
package main

import (
	"context"
	"log"
	"os"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("DB_NAME", "investment_tracker")

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	col := client.Database(dbName).Collection("sips")

	cursor, err := col.Find(context.Background(), bson.M{})
	if err != nil {
		log.Fatalf("Failed to fetch SIPs: %v", err)
	}
	defer cursor.Close(context.Background())

	var sips []models.SIP
	if err := cursor.All(context.Background(), &sips); err != nil {
		log.Fatalf("Failed to decode SIPs: %v", err)
	}

	updated := 0
	skipped := 0
	for i := range sips {
		changed := false
		for j := range sips[i].Installments {
			inst := &sips[i].Installments[j]
			if inst.Units == 0 && inst.NAV > 0 {
				inst.Units = inst.Amount / inst.NAV
				changed = true
			}
		}
		if !changed {
			skipped++
			continue
		}
		_, err := col.ReplaceOne(
			context.Background(),
			bson.M{"_id": sips[i].ID},
			sips[i],
		)
		if err != nil {
			log.Printf("Failed to update SIP %s: %v", sips[i].ID.Hex(), err)
		} else {
			updated++
			log.Printf("Updated SIP: %s (%s)", sips[i].FundName, sips[i].ID.Hex())
		}
	}

	log.Printf("Done. Updated: %d, Skipped (already had units or no NAV): %d", updated, skipped)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
