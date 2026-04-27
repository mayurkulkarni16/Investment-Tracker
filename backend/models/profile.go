package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Profile struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name         string             `json:"name" bson:"name"`
	Relationship string             `json:"relationship" bson:"relationship"` // self, spouse, parent, child, sibling, other
	Color        string             `json:"color" bson:"color"`               // hex color for UI
	IsDefault    bool               `json:"is_default" bson:"is_default"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreateProfileRequest struct {
	Name         string `json:"name" binding:"required"`
	Relationship string `json:"relationship" binding:"required"`
	Color        string `json:"color"`
	IsDefault    bool   `json:"is_default"`
}

type UpdateProfileRequest struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	Color        string `json:"color"`
}
