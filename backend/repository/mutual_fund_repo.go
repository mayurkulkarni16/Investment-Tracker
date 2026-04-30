package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MutualFundRepo struct {
	collection *mongo.Collection
}

func NewMutualFundRepo(db *mongo.Database) *MutualFundRepo {
	return &MutualFundRepo{
		collection: db.Collection("mutual_funds"),
	}
}

func (r *MutualFundRepo) Create(ctx context.Context, mf *models.MutualFund) error {
	mf.CreatedAt = time.Now()
	mf.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, mf)
	if err != nil {
		return err
	}
	mf.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MutualFundRepo) GetAll(ctx context.Context, userID string) ([]models.MutualFund, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var funds []models.MutualFund
	if err := cursor.All(ctx, &funds); err != nil {
		return nil, err
	}
	return funds, nil
}

func (r *MutualFundRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.MutualFund, error) {
	var mf models.MutualFund
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&mf)
	if err != nil {
		return nil, err
	}
	return &mf, nil
}

func (r *MutualFundRepo) Update(ctx context.Context, mf *models.MutualFund) error {
	mf.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": mf.ID}, mf)
	return err
}

func (r *MutualFundRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *MutualFundRepo) GetByFolioNumber(ctx context.Context, userID string, folioNumber string) (*models.MutualFund, error) {
	var mf models.MutualFund
	err := r.collection.FindOne(ctx, bson.M{"folio_number": folioNumber, "user_id": userID}).Decode(&mf)
	if err != nil {
		return nil, err
	}
	return &mf, nil
}
