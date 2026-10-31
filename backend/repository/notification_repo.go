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

type NotificationRepo struct {
	collection *mongo.Collection
}

func NewNotificationRepo(db *mongo.Database) *NotificationRepo {
	coll := db.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Partial so that legacy notifications without a dedupe key don't conflict.
	_, _ = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "dedupe_key", Value: 1}},
		Options: options.Index().SetUnique(true).
			SetPartialFilterExpression(bson.M{"dedupe_key": bson.M{"$exists": true}}),
	})
	return &NotificationRepo{collection: coll}
}

// Upsert inserts the notification unless one with the same user and dedupe key
// already exists (read or unread). Returns true if a new notification was created.
func (r *NotificationRepo) Upsert(ctx context.Context, notif *models.Notification) (bool, error) {
	if notif.DedupeKey == "" {
		return true, r.Create(ctx, notif)
	}
	notif.CreatedAt = time.Now()
	doc := *notif
	doc.ID = primitive.NilObjectID
	res, err := r.collection.UpdateOne(ctx,
		bson.M{"user_id": notif.UserID, "dedupe_key": notif.DedupeKey},
		bson.M{"$setOnInsert": doc},
		options.Update().SetUpsert(true),
	)
	if mongo.IsDuplicateKeyError(err) {
		return false, nil // concurrent scan inserted it first
	}
	if err != nil {
		return false, err
	}
	if id, ok := res.UpsertedID.(primitive.ObjectID); ok {
		notif.ID = id
		return true, nil
	}
	return false, nil
}

func (r *NotificationRepo) Create(ctx context.Context, notif *models.Notification) error {
	notif.CreatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, notif)
	if err != nil {
		return err
	}
	notif.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *NotificationRepo) GetAll(ctx context.Context, userID string) ([]models.Notification, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}}).SetLimit(100)
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter), opts)
	if err != nil {
		return nil, err
	}
	var notifs []models.Notification
	if err := cursor.All(ctx, &notifs); err != nil {
		return nil, err
	}
	return notifs, nil
}

func (r *NotificationRepo) GetUnread(ctx context.Context, userID string) ([]models.Notification, error) {
	filter := bson.M{"is_read": false}
	if userID != "" {
		filter["user_id"] = userID
	}
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter), opts)
	if err != nil {
		return nil, err
	}
	var notifs []models.Notification
	if err := cursor.All(ctx, &notifs); err != nil {
		return nil, err
	}
	return notifs, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.UpdateOne(ctx, scoped(ctx, bson.M{"_id": id}), bson.M{"$set": bson.M{"is_read": true}})
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID string) error {
	filter := bson.M{"is_read": false}
	if userID != "" {
		filter["user_id"] = userID
	}
	_, err := r.collection.UpdateMany(ctx, scoped(ctx, filter), bson.M{"$set": bson.M{"is_read": true}})
	return err
}

func (r *NotificationRepo) DeleteOlderThan(ctx context.Context, before time.Time) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"date": bson.M{"$lt": before}, "is_read": true})
	return err
}
