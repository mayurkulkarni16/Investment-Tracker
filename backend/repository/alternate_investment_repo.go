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

type AlternateInvestmentRepo struct {
	collection *mongo.Collection
}

func NewAlternateInvestmentRepo(db *mongo.Database) *AlternateInvestmentRepo {
	return &AlternateInvestmentRepo{collection: db.Collection("vehicle_rentals")}
}

func (r *AlternateInvestmentRepo) Create(ctx context.Context, v *models.AlternateInvestment) error {
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, v)
	if err != nil {
		return err
	}
	v.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AlternateInvestmentRepo) GetAll(ctx context.Context, userID string) ([]models.AlternateInvestment, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	opts := options.Find().SetSort(bson.D{{Key: "purchase_date", Value: 1}})
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter), opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.AlternateInvestment
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlternateInvestmentRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.AlternateInvestment, error) {
	var v models.AlternateInvestment
	if err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id})).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *AlternateInvestmentRepo) Update(ctx context.Context, v *models.AlternateInvestment) error {
	v.UpdatedAt = time.Now()
	res, err := r.collection.ReplaceOne(ctx, scoped(ctx, bson.M{"_id": v.ID}), v)
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *AlternateInvestmentRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}
