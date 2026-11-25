package repository

import (
	"context"
	"errors"
	"time"

	"investment-tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AlternateInvestmentRepo struct {
	collection *mongo.Collection
}

var ErrAlternateInvestmentChanged = errors.New("investment changed; refresh and try again")

func NewAlternateInvestmentRepo(db *mongo.Database) *AlternateInvestmentRepo {
	return &AlternateInvestmentRepo{collection: db.Collection("vehicle_rentals")}
}

func (r *AlternateInvestmentRepo) Create(ctx context.Context, v *models.AlternateInvestment) error {
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	res, err := r.collection.InsertOne(ctx, v)
	if err != nil {
		return err
	}
	v.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AlternateInvestmentRepo) GetAll(ctx context.Context, userID string) ([]models.AlternateInvestment, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	opts := options.Find().SetSort(bson.D{{Key: "purchase_date", Value: 1}})
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter), opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.AlternateInvestment
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlternateInvestmentRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.AlternateInvestment, error) {
	var v models.AlternateInvestment
	if err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id})).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *AlternateInvestmentRepo) Update(ctx context.Context, v *models.AlternateInvestment) error {
	filter := alternateInvestmentRevision(v)
	v.UpdatedAt = nextAlternateInvestmentRevision(v.UpdatedAt)
	res, err := r.collection.ReplaceOne(ctx, scoped(ctx, filter), v)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrAlternateInvestmentChanged
	}
	return nil
}

// UpdateSchedule applies a migration without replacing any other stored fields.
func (r *AlternateInvestmentRepo) UpdateSchedule(ctx context.Context, v *models.AlternateInvestment) error {
	filter := alternateInvestmentRevision(v)
	filter["schedule_version"] = bson.M{"$in": bson.A{nil, 0}}
	res, err := r.collection.UpdateOne(ctx, scoped(ctx, filter), bson.M{"$set": bson.M{
		"schedule_version":  v.ScheduleVersion,
		"payout_start_date": v.PayoutStartDate,
		"payouts":           v.Payouts,
		"updated_at":        nextAlternateInvestmentRevision(v.UpdatedAt),
	}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrAlternateInvestmentChanged
	}
	return nil
}

func alternateInvestmentRevision(v *models.AlternateInvestment) bson.M {
	filter := bson.M{"_id": v.ID, "user_id": v.UserID, "updated_at": v.UpdatedAt}
	if v.UpdatedAt.IsZero() {
		filter["updated_at"] = bson.M{"$in": bson.A{nil, v.UpdatedAt}}
	}
	return filter
}

func nextAlternateInvestmentRevision(previous time.Time) time.Time {
	now := time.Now().Truncate(time.Millisecond)
	next := previous.Truncate(time.Millisecond).Add(time.Millisecond)
	if now.Before(next) {
		return next
	}
	return now
}

func (r *AlternateInvestmentRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}
