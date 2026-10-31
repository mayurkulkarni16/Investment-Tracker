package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationSettings controls delivery of reminders outside the app.
type NotificationSettings struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          string             `json:"user_id" bson:"user_id"`
	EmailEnabled    bool               `json:"email_enabled" bson:"email_enabled"`
	Email           string             `json:"email" bson:"email"` // defaults to the account email
	TelegramEnabled bool               `json:"telegram_enabled" bson:"telegram_enabled"`
	TelegramChatID  string             `json:"telegram_chat_id" bson:"telegram_chat_id"`
	LastDispatchAt  *time.Time         `json:"last_dispatch_at,omitempty" bson:"last_dispatch_at,omitempty"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`

	// Server capabilities (not stored)
	EmailAvailable    bool `json:"email_available" bson:"-"`
	TelegramAvailable bool `json:"telegram_available" bson:"-"`
}

type NotificationSettingsRequest struct {
	EmailEnabled    bool   `json:"email_enabled"`
	Email           string `json:"email"`
	TelegramEnabled bool   `json:"telegram_enabled"`
	TelegramChatID  string `json:"telegram_chat_id"`
}

// Document is metadata for a file stored in the GridFS "documents" bucket.
type Document struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      string             `json:"user_id" bson:"user_id"`
	FileID      primitive.ObjectID `json:"-" bson:"file_id"`
	Title       string             `json:"title" bson:"title"`
	FileName    string             `json:"file_name" bson:"file_name"`
	ContentType string             `json:"content_type" bson:"content_type"`
	Size        int64              `json:"size" bson:"size"`
	Category    string             `json:"category" bson:"category"` // agreement, receipt, policy, statement, tax, id_proof, other
	AssetType   string             `json:"asset_type" bson:"asset_type"`
	AssetID     string             `json:"asset_id" bson:"asset_id"`
	ExpiryDate  *time.Time         `json:"expiry_date,omitempty" bson:"expiry_date,omitempty"`
	Notes       string             `json:"notes" bson:"notes"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
}
