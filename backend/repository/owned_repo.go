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

// OwnedRepo is a generic user-scoped CRUD repository for simple collections
// whose documents carry "_id", "user_id", "created_at" and "updated_at".
type OwnedRepo[T any] struct {
	collection *mongo.Collection
	sort       bson.D
}

func NewOwnedRepo[T any](db *mongo.Database, name string, sort bson.D) *OwnedRepo[T] {
	return &OwnedRepo[T]{collection: db.Collection(name), sort: sort}
}

func (r *OwnedRepo[T]) Collection() *mongo.Collection { return r.collection }

// Insert stores doc and returns its new ID. Callers set user_id and timestamps.
func (r *OwnedRepo[T]) Insert(ctx context.Context, doc *T) (primitive.ObjectID, error) {
	res, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, _ := res.InsertedID.(primitive.ObjectID)
	return id, nil
}

func (r *OwnedRepo[T]) Find(ctx context.Context, userID string, filter bson.M) ([]T, error) {
	if filter == nil {
		filter = bson.M{}
	}
	if userID != "" {
		filter["user_id"] = userID
	}
	opts := options.Find()
	if len(r.sort) > 0 {
		opts.SetSort(r.sort)
	}
	cursor, err := r.collection.Find(ctx, scoped(ctx, filter), opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []T{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *OwnedRepo[T]) GetAll(ctx context.Context, userID string) ([]T, error) {
	return r.Find(ctx, userID, nil)
}

func (r *OwnedRepo[T]) GetByID(ctx context.Context, id primitive.ObjectID) (*T, error) {
	var doc T
	if err := r.collection.FindOne(ctx, scoped(ctx, bson.M{"_id": id})).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *OwnedRepo[T]) Replace(ctx context.Context, id primitive.ObjectID, doc *T) error {
	res, err := r.collection.ReplaceOne(ctx, scoped(ctx, bson.M{"_id": id}), doc)
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *OwnedRepo[T]) Set(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()
	res, err := r.collection.UpdateOne(ctx, scoped(ctx, bson.M{"_id": id}), bson.M{"$set": fields})
	if err != nil {
		return err
	}
	return requireMatch(res.MatchedCount)
}

func (r *OwnedRepo[T]) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}

// Upsert replaces the single document matching filter (scoped to the owner) or inserts doc.
func (r *OwnedRepo[T]) Upsert(ctx context.Context, filter bson.M, set bson.M) error {
	set["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(ctx, scoped(ctx, filter), bson.M{"$set": set}, options.Update().SetUpsert(true))
	return err
}

func (r *OwnedRepo[T]) FindOne(ctx context.Context, filter bson.M) (*T, error) {
	var doc T
	if err := r.collection.FindOne(ctx, scoped(ctx, filter)).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *OwnedRepo[T]) DeleteMany(ctx context.Context, filter bson.M) error {
	_, err := r.collection.DeleteMany(ctx, scoped(ctx, filter))
	return err
}

// Typed repositories for the personal-finance collections.
type (
	BankAccountRepo          = OwnedRepo[models.BankAccount]
	BudgetRepo               = OwnedRepo[models.Budget]
	InsuranceRepo            = OwnedRepo[models.InsurancePolicy]
	GoldRepo                 = OwnedRepo[models.GoldHolding]
	PropertyRepo             = OwnedRepo[models.Property]
	ProfileAssignmentRepo    = OwnedRepo[models.ProfileAssignment]
	NotificationSettingsRepo = OwnedRepo[models.NotificationSettings]
	DocumentRepo             = OwnedRepo[models.Document]
	MemberIncomeRepo         = OwnedRepo[models.MemberIncome]
	TaxSettingsRepo          = OwnedRepo[models.TaxSettings]
)

func NewTaxSettingsRepo(db *mongo.Database) *TaxSettingsRepo {
	return NewOwnedRepo[models.TaxSettings](db, "tax_settings", nil)
}

func NewMemberIncomeRepo(db *mongo.Database) *MemberIncomeRepo {
	r := NewOwnedRepo[models.MemberIncome](db, "member_incomes", bson.D{{Key: "fy", Value: 1}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "profile_id", Value: 1}, {Key: "fy", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return r
}

func NewBankAccountRepo(db *mongo.Database) *BankAccountRepo {
	return NewOwnedRepo[models.BankAccount](db, "bank_accounts", bson.D{{Key: "account_type", Value: 1}, {Key: "name", Value: 1}})
}

func NewBudgetRepo(db *mongo.Database) *BudgetRepo {
	return NewOwnedRepo[models.Budget](db, "budgets", bson.D{{Key: "category", Value: 1}})
}

func NewInsuranceRepo(db *mongo.Database) *InsuranceRepo {
	return NewOwnedRepo[models.InsurancePolicy](db, "insurance_policies", bson.D{{Key: "policy_type", Value: 1}, {Key: "start_date", Value: 1}})
}

func NewGoldRepo(db *mongo.Database) *GoldRepo {
	return NewOwnedRepo[models.GoldHolding](db, "gold_holdings", bson.D{{Key: "purchase_date", Value: 1}})
}

func NewPropertyRepo(db *mongo.Database) *PropertyRepo {
	return NewOwnedRepo[models.Property](db, "properties", bson.D{{Key: "purchase_date", Value: 1}})
}

func NewProfileAssignmentRepo(db *mongo.Database) *ProfileAssignmentRepo {
	r := NewOwnedRepo[models.ProfileAssignment](db, "profile_assignments", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "asset_type", Value: 1}, {Key: "asset_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return r
}

func NewNotificationSettingsRepo(db *mongo.Database) *NotificationSettingsRepo {
	return NewOwnedRepo[models.NotificationSettings](db, "notification_settings", nil)
}

func NewDocumentRepo(db *mongo.Database) *DocumentRepo {
	return NewOwnedRepo[models.Document](db, "documents", bson.D{{Key: "created_at", Value: -1}})
}
