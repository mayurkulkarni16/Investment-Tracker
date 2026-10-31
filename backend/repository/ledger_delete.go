package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Delete removes a ledger entry owned by userID.
func (r *LedgerRepo) Delete(ctx context.Context, userID string, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, scoped(ctx, bson.M{"_id": id, "user_id": userID}))
	if err != nil {
		return err
	}
	return requireMatch(res.DeletedCount)
}
