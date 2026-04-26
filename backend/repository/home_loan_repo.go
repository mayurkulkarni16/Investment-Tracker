package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type HomeLoanRepo struct {
	collection *mongo.Collection
}

func NewHomeLoanRepo(db *mongo.Database) *HomeLoanRepo {
	return &HomeLoanRepo{
		collection: db.Collection("home_loans"),
	}
}

func (r *HomeLoanRepo) Create(ctx context.Context, loan *models.HomeLoan) error {
	loan.CreatedAt = time.Now()
	loan.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, loan)
	if err != nil {
		return err
	}
	loan.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *HomeLoanRepo) GetAll(ctx context.Context) ([]models.HomeLoan, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var loans []models.HomeLoan
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, err
	}
	return loans, nil
}

func (r *HomeLoanRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.HomeLoan, error) {
	var loan models.HomeLoan
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&loan)
	if err != nil {
		return nil, err
	}
	return &loan, nil
}

func (r *HomeLoanRepo) Update(ctx context.Context, loan *models.HomeLoan) error {
	loan.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": loan.ID}, loan)
	return err
}

func (r *HomeLoanRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
