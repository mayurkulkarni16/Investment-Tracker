package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SIPRepo struct {
	collection *mongo.Collection
}

func NewSIPRepo(db *mongo.Database) *SIPRepo {
	return &SIPRepo{collection: db.Collection("sips")}
}

func (r *SIPRepo) Create(ctx context.Context, sip *models.SIP) error {
	sip.CreatedAt = time.Now()
	sip.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, sip)
	if err != nil {
		return err
	}
	sip.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *SIPRepo) GetAll(ctx context.Context) ([]models.SIP, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var sips []models.SIP
	if err := cursor.All(ctx, &sips); err != nil {
		return nil, err
	}
	return sips, nil
}

func (r *SIPRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.SIP, error) {
	var sip models.SIP
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&sip); err != nil {
		return nil, err
	}
	return &sip, nil
}

func (r *SIPRepo) Update(ctx context.Context, sip *models.SIP) error {
	sip.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": sip.ID}, sip)
	return err
}

func (r *SIPRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
