package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LedgerRepo struct{ collection *mongo.Collection }

func NewLedgerRepo(db *mongo.Database) *LedgerRepo {
	return &LedgerRepo{collection: db.Collection("ledger_entries")}
}

func (r *LedgerRepo) GetAll(ctx context.Context, userID string) ([]models.LedgerEntry, error) {
	cursor, err := r.collection.Find(ctx, scoped(ctx, bson.M{"user_id": userID}), options.Find().SetSort(bson.D{{Key: "date", Value: -1}, {Key: "_id", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	entries := []models.LedgerEntry{}
	err = cursor.All(ctx, &entries)
	return entries, err
}

func (r *LedgerRepo) GetByID(ctx context.Context, userID string, id primitive.ObjectID) (*models.LedgerEntry, error) {
	var entry models.LedgerEntry
	err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id, "user_id": userID})).Decode(&entry)
	return &entry, err
}

func (r *LedgerRepo) Create(ctx context.Context, entry *models.LedgerEntry) (*models.LedgerEntry, error) {
	// The deterministic, user-bound _id makes scan/retry insertion atomic.
	filter := scoped(ctx, bson.M{"_id": entry.ID, "user_id": entry.UserID})
	var result models.LedgerEntry
	err := r.collection.FindOneAndUpdate(ctx, filter, bson.M{"$setOnInsert": entry},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&result)
	if mongo.IsDuplicateKeyError(err) {
		err = r.collection.FindOne(ctx, filter).Decode(&result)
	}
	return &result, err
}

func (r *LedgerRepo) Update(ctx context.Context, userID string, id primitive.ObjectID, req models.LedgerRequest, attribution models.LedgerAttribution, updatedAt time.Time) (*models.LedgerEntry, error) {
	var result models.LedgerEntry
	err := r.collection.FindOneAndUpdate(ctx, scoped(ctx, bson.M{"_id": id, "user_id": userID}), bson.M{"$set": bson.M{
		"status": req.Status, "kind": req.Kind, "amount_paise": req.AmountPaise,
		"date": req.Date, "description": req.Description, "category": req.Category,
		"bank": req.Bank, "account": req.Account, "due_date": req.DueDate, "updated_at": updatedAt,
		"profile_id": attribution.ProfileID, "linked_account_type": attribution.LinkedAccountType,
		"linked_account_id": attribution.LinkedAccountID, "sms_received_at": attribution.SMSReceivedAt,
	}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&result)
	return &result, err
}
