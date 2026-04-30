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
	return &NotificationRepo{collection: db.Collection("notifications")}
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
	cursor, err := r.collection.Find(ctx, filter, opts)
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
	cursor, err := r.collection.Find(ctx, filter, opts)
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
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"is_read": true}})
	return err
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID string) error {
	filter := bson.M{"is_read": false}
	if userID != "" {
		filter["user_id"] = userID
	}
	_, err := r.collection.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"is_read": true}})
	return err
}

func (r *NotificationRepo) DeleteOlderThan(ctx context.Context, before time.Time) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"date": bson.M{"$lt": before}, "is_read": true})
	return err
}
