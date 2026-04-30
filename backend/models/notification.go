package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        string             `json:"user_id" bson:"user_id"`
	Type          string             `json:"type" bson:"type"` // emi_due, fd_maturity, sip_due, bond_coupon, credit_card_due, goal_milestone
	Title         string             `json:"title" bson:"title"`
	Message       string             `json:"message" bson:"message"`
	Date          time.Time          `json:"date" bson:"date"`
	ReferenceType string             `json:"reference_type" bson:"reference_type"` // home_loan, personal_loan, etc.
	ReferenceID   string             `json:"reference_id" bson:"reference_id"`
	IsRead        bool               `json:"is_read" bson:"is_read"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}
