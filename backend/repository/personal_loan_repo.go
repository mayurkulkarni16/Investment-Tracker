package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PersonalLoanRepo struct {
	collection *mongo.Collection
}

func NewPersonalLoanRepo(db *mongo.Database) *PersonalLoanRepo {
	return &PersonalLoanRepo{
		collection: db.Collection("personal_loans"),
	}
}

func (r *PersonalLoanRepo) Create(ctx context.Context, loan *models.PersonalLoan) error {
	loan.CreatedAt = time.Now()
	loan.UpdatedAt = time.Now()
	result, err := r.collection.InsertOne(ctx, loan)
	if err != nil {
		return err
	}
	loan.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *PersonalLoanRepo) GetAll(ctx context.Context, userID string) ([]models.PersonalLoan, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var loans []models.PersonalLoan
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, err
	}
	return loans, nil
}

func (r *PersonalLoanRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.PersonalLoan, error) {
	var loan models.PersonalLoan
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&loan); err != nil {
		return nil, err
	}
	return &loan, nil
}

func (r *PersonalLoanRepo) Update(ctx context.Context, loan *models.PersonalLoan) error {
	loan.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": loan.ID}, loan)
	return err
}

func (r *PersonalLoanRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
