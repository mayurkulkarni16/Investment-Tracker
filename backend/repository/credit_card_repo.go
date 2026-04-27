package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreditCardRepo struct {
	collection *mongo.Collection
}

func NewCreditCardRepo(db *mongo.Database) *CreditCardRepo {
	return &CreditCardRepo{collection: db.Collection("credit_cards")}
}

func (r *CreditCardRepo) Create(ctx context.Context, card *models.CreditCard) error {
	card.CreatedAt = time.Now()
	card.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, card)
	if err != nil {
		return err
	}
	card.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *CreditCardRepo) GetAll(ctx context.Context) ([]models.CreditCard, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var cards []models.CreditCard
	if err := cursor.All(ctx, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *CreditCardRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.CreditCard, error) {
	var card models.CreditCard
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&card); err != nil {
		return nil, err
	}
	return &card, nil
}

func (r *CreditCardRepo) Update(ctx context.Context, card *models.CreditCard) error {
	card.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": card.ID}, card)
	return err
}

func (r *CreditCardRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
