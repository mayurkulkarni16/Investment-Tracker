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

type NetWorthRepo struct {
	collection *mongo.Collection
}

func NewNetWorthRepo(db *mongo.Database) *NetWorthRepo {
	return &NetWorthRepo{collection: db.Collection("net_worth_snapshots")}
}

func (r *NetWorthRepo) Create(ctx context.Context, snapshot *models.NetWorthSnapshot) error {
	snapshot.CreatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, snapshot)
	if err != nil {
		return err
	}
	snapshot.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *NetWorthRepo) GetAll(ctx context.Context) ([]models.NetWorthSnapshot, error) {
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var snapshots []models.NetWorthSnapshot
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (r *NetWorthRepo) GetByMonth(ctx context.Context, month string) (*models.NetWorthSnapshot, error) {
	var snapshot models.NetWorthSnapshot
	if err := r.collection.FindOne(ctx, bson.M{"month": month}).Decode(&snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *NetWorthRepo) Upsert(ctx context.Context, snapshot *models.NetWorthSnapshot) error {
	snapshot.CreatedAt = time.Now()
	opts := options.Replace().SetUpsert(true)
	_, err := r.collection.ReplaceOne(ctx, bson.M{"month": snapshot.Month}, snapshot, opts)
	return err
}

func (r *NetWorthRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
