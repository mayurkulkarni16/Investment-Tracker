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

func (r *SIPRepo) GetAll(ctx context.Context, userID string) ([]models.SIP, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter))
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
	if err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id})).Decode(&sip); err != nil {
		return nil, err
	}
	return &sip, nil
}

func (r *SIPRepo) Update(ctx context.Context, sip *models.SIP) error {
	sip.UpdatedAt = time.Now()
	res, err := r.collection.ReplaceOne(ctx, scoped(ctx, bson.M{"_id": sip.ID}), sip)
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *SIPRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}
