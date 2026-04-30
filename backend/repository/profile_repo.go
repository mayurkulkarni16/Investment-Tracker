package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProfileRepo struct {
	collection *mongo.Collection
}

func NewProfileRepo(db *mongo.Database) *ProfileRepo {
	return &ProfileRepo{collection: db.Collection("profiles")}
}

func (r *ProfileRepo) Create(ctx context.Context, profile *models.Profile) error {
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, profile)
	if err != nil {
		return err
	}
	profile.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProfileRepo) GetAll(ctx context.Context, userID string) ([]models.Profile, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var profiles []models.Profile
	if err := cursor.All(ctx, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *ProfileRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Profile, error) {
	var profile models.Profile
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepo) Update(ctx context.Context, profile *models.Profile) error {
	profile.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": profile.ID}, profile)
	return err
}

func (r *ProfileRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
