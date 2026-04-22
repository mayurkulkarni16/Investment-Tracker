package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProvidentFundRepo struct {
	collection *mongo.Collection
}

func NewProvidentFundRepo(db *mongo.Database) *ProvidentFundRepo {
	return &ProvidentFundRepo{
		collection: db.Collection("provident_fund_entries"),
	}
}

func (r *ProvidentFundRepo) Create(ctx context.Context, pf *models.ProvidentFund) error {
	pf.CreatedAt = time.Now()
	pf.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, pf)
	if err != nil {
		return err
	}
	pf.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProvidentFundRepo) GetAll(ctx context.Context) ([]models.ProvidentFund, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var pfs []models.ProvidentFund
	if err := cursor.All(ctx, &pfs); err != nil {
		return nil, err
	}
	return pfs, nil
}

func (r *ProvidentFundRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.ProvidentFund, error) {
	var pf models.ProvidentFund
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pf)
	if err != nil {
		return nil, err
	}
	return &pf, nil
}

func (r *ProvidentFundRepo) Update(ctx context.Context, pf *models.ProvidentFund) error {
	pf.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": pf.ID}, pf)
	return err
}

func (r *ProvidentFundRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
