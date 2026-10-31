package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ownerKey struct{}

// WithOwner returns a context that restricts all user-owned repository
// operations to documents whose user_id equals userID. Request handlers must
// use such a context; background jobs (e.g. the scheduler) use an unscoped one.
func WithOwner(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ownerKey{}, userID)
}

// OwnerFrom returns the owner bound to ctx by WithOwner, if any.
func OwnerFrom(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(ownerKey{}).(string)
	return uid, ok && uid != ""
}

// scoped adds the context owner's user_id to filter, overriding any caller-supplied value.
func scoped(ctx context.Context, filter bson.M) bson.M {
	if uid, ok := OwnerFrom(ctx); ok {
		filter["user_id"] = uid
	}
	return filter
}

func requireMatch(matched int64) error {
	if matched == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
