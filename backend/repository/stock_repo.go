package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type StockRepo struct {
	collection *mongo.Collection
}

func NewStockRepo(db *mongo.Database) *StockRepo {
	return &StockRepo{
		collection: db.Collection("stocks"),
	}
}

func (r *StockRepo) Create(ctx context.Context, stock *models.Stock) error {
	stock.CreatedAt = time.Now()
	stock.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, stock)
	if err != nil {
		return err
	}
	stock.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *StockRepo) GetAll(ctx context.Context, userID string) ([]models.Stock, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stocks []models.Stock
	if err := cursor.All(ctx, &stocks); err != nil {
		return nil, err
	}
	return stocks, nil
}

func (r *StockRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Stock, error) {
	var stock models.Stock
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&stock)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (r *StockRepo) Update(ctx context.Context, stock *models.Stock) error {
	stock.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": stock.ID}, stock)
	return err
}

func (r *StockRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
