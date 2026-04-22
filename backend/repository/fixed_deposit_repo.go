package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FixedDepositRepo struct {
	collection *mongo.Collection
}

func NewFixedDepositRepo(db *mongo.Database) *FixedDepositRepo {
	return &FixedDepositRepo{
		collection: db.Collection("fixed_deposits"),
	}
}

func (r *FixedDepositRepo) Create(ctx context.Context, fd *models.FixedDeposit) error {
	fd.CreatedAt = time.Now()
	fd.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, fd)
	if err != nil {
		return err
	}
	fd.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *FixedDepositRepo) GetAll(ctx context.Context) ([]models.FixedDeposit, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var fds []models.FixedDeposit
	if err := cursor.All(ctx, &fds); err != nil {
		return nil, err
	}
	return fds, nil
}

func (r *FixedDepositRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.FixedDeposit, error) {
	var fd models.FixedDeposit
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&fd)
	if err != nil {
		return nil, err
	}
	return &fd, nil
}

func (r *FixedDepositRepo) Update(ctx context.Context, fd *models.FixedDeposit) error {
	fd.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": fd.ID}, fd)
	return err
}

func (r *FixedDepositRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
