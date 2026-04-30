package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type NPSRepo struct {
	collection *mongo.Collection
}

func NewNPSRepo(db *mongo.Database) *NPSRepo {
	return &NPSRepo{collection: db.Collection("nps_accounts")}
}

func (r *NPSRepo) Create(ctx context.Context, account *models.NPSAccount) error {
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, account)
	if err != nil {
		return err
	}
	account.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *NPSRepo) GetAll(ctx context.Context, userID string) ([]models.NPSAccount, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var accounts []models.NPSAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *NPSRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.NPSAccount, error) {
	var account models.NPSAccount
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&account); err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *NPSRepo) Update(ctx context.Context, account *models.NPSAccount) error {
	account.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": account.ID}, account)
	return err
}

func (r *NPSRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
