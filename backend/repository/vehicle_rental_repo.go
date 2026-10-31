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

type VehicleRentalRepo struct {
	collection *mongo.Collection
}

func NewVehicleRentalRepo(db *mongo.Database) *VehicleRentalRepo {
	return &VehicleRentalRepo{collection: db.Collection("vehicle_rentals")}
}

func (r *VehicleRentalRepo) Create(ctx context.Context, v *models.VehicleRental) error {
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, v)
	if err != nil {
		return err
	}
	v.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *VehicleRentalRepo) GetAll(ctx context.Context, userID string) ([]models.VehicleRental, error) {
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

	var items []models.VehicleRental
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *VehicleRentalRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.VehicleRental, error) {
	var v models.VehicleRental
	if err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id})).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VehicleRentalRepo) Update(ctx context.Context, v *models.VehicleRental) error {
	v.UpdatedAt = time.Now()
	res, err := r.collection.ReplaceOne(ctx, scoped(ctx, bson.M{"_id": v.ID}), v)
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *VehicleRentalRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}
