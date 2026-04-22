package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CorporateBondRepo struct {
	collection *mongo.Collection
}

func NewCorporateBondRepo(db *mongo.Database) *CorporateBondRepo {
	return &CorporateBondRepo{
		collection: db.Collection("corporate_bonds"),
	}
}

func (r *CorporateBondRepo) Create(ctx context.Context, bond *models.CorporateBond) error {
	bond.CreatedAt = time.Now()
	bond.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, bond)
	if err != nil {
		return err
	}
	bond.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *CorporateBondRepo) GetAll(ctx context.Context) ([]models.CorporateBond, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var bonds []models.CorporateBond
	if err := cursor.All(ctx, &bonds); err != nil {
		return nil, err
	}
	return bonds, nil
}

func (r *CorporateBondRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.CorporateBond, error) {
	var bond models.CorporateBond
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&bond)
	if err != nil {
		return nil, err
	}
	return &bond, nil
}

func (r *CorporateBondRepo) Update(ctx context.Context, bond *models.CorporateBond) error {
	bond.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": bond.ID}, bond)
	return err
}

func (r *CorporateBondRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
