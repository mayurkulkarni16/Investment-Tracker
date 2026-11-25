package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OverdraftRepo struct {
	collection *mongo.Collection
}

func NewOverdraftRepo(db *mongo.Database) *OverdraftRepo {
	return &OverdraftRepo{collection: db.Collection("overdrafts")}
}

func (r *OverdraftRepo) Create(ctx context.Context, od *models.Overdraft) error {
	od.CreatedAt = time.Now()
	od.UpdatedAt = od.CreatedAt
	result, err := r.collection.InsertOne(ctx, od)
	if err != nil {
		return err
	}
	od.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *OverdraftRepo) GetAll(ctx context.Context, userID string) ([]models.Overdraft, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.Overdraft
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *OverdraftRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Overdraft, error) {
	var od models.Overdraft
	if err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id})).Decode(&od); err != nil {
		return nil, err
	}
	return &od, nil
}

func (r *OverdraftRepo) Update(ctx context.Context, od *models.Overdraft) error {
	od.UpdatedAt = time.Now()
	res, err := r.collection.ReplaceOne(ctx, scoped(ctx, bson.M{"_id": od.ID}), od)
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *OverdraftRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}
