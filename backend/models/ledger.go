package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AmountPaise is integer INR minor units; bills and transfers are not expenses.
type LedgerEntry struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	UserID      string             `json:"user_id" bson:"user_id"`
	SourceKey   string             `json:"source_key" bson:"source_key"`
	Source      string             `json:"source" bson:"source"`
	Status      string             `json:"status" bson:"status"`
	Kind        string             `json:"kind" bson:"kind"`
	AmountPaise int64              `json:"amount_paise" bson:"amount_paise"`
	Date        string             `json:"date" bson:"date"`
	Description string             `json:"description" bson:"description"`
	Category    string             `json:"category" bson:"category"`
	Bank        string             `json:"bank" bson:"bank"`
	Account     string             `json:"account" bson:"account"`
	DueDate     string             `json:"due_date" bson:"due_date"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

type LedgerRequest struct {
	SourceKey   string `json:"source_key"`
	Source      string `json:"source"`
	Status      string `json:"status"`
	Kind        string `json:"kind"`
	AmountPaise int64  `json:"amount_paise"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Bank        string `json:"bank"`
	Account     string `json:"account"`
	DueDate     string `json:"due_date"`
}
