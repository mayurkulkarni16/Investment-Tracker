package repository

import (
	"context"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GoalRepo struct {
	collection *mongo.Collection
}

func NewGoalRepo(db *mongo.Database) *GoalRepo {
	return &GoalRepo{collection: db.Collection("goals")}
}

func (r *GoalRepo) Create(ctx context.Context, goal *models.Goal) error {
	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, goal)
	if err != nil {
		return err
	}
	goal.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *GoalRepo) GetAll(ctx context.Context) ([]models.Goal, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var goals []models.Goal
	if err := cursor.All(ctx, &goals); err != nil {
		return nil, err
	}
	return goals, nil
}

func (r *GoalRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Goal, error) {
	var goal models.Goal
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&goal); err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *GoalRepo) Update(ctx context.Context, goal *models.Goal) error {
	goal.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": goal.ID}, goal)
	return err
}

func (r *GoalRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
