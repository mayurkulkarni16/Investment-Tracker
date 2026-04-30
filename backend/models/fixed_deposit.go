package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InterestType string

const (
	InterestCumulative    InterestType = "cumulative"
	InterestNonCumulative InterestType = "non_cumulative"
)

type FDStatus string

const (
	FDActive          FDStatus = "active"
	FDMatured         FDStatus = "matured"
	FDPrematureClosed FDStatus = "premature_closed"
)

type FixedDeposit struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          string             `json:"user_id" bson:"user_id"`
	BankName        string             `json:"bank_name" bson:"bank_name"`
	FDNumber        string             `json:"fd_number" bson:"fd_number"`
	PrincipalAmount float64            `json:"principal_amount" bson:"principal_amount"`
	InterestRate    float64            `json:"interest_rate" bson:"interest_rate"`
	StartDate       time.Time          `json:"start_date" bson:"start_date"`
	MaturityDate    time.Time          `json:"maturity_date" bson:"maturity_date"`
	TenureMonths    int                `json:"tenure_months" bson:"tenure_months"`
	InterestType    InterestType       `json:"interest_type" bson:"interest_type"`
	PayoutFrequency *PayoutFrequency   `json:"payout_frequency,omitempty" bson:"payout_frequency,omitempty"`
	MaturityAmount  float64            `json:"maturity_amount" bson:"maturity_amount"`
	InterestEarned  float64            `json:"interest_earned" bson:"interest_earned"`
	XIRR            float64            `json:"xirr" bson:"-"`
	IsAutoRenewed   bool               `json:"is_auto_renewed" bson:"is_auto_renewed"`
	Status          FDStatus           `json:"status" bson:"status"`
	Notes           string             `json:"notes" bson:"notes"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreateFixedDepositRequest struct {
	BankName        string           `json:"bank_name" binding:"required"`
	FDNumber        string           `json:"fd_number"`
	PrincipalAmount float64          `json:"principal_amount" binding:"required"`
	InterestRate    float64          `json:"interest_rate" binding:"required"`
	StartDate       string           `json:"start_date" binding:"required"`
	MaturityDate    string           `json:"maturity_date" binding:"required"`
	TenureMonths    int              `json:"tenure_months" binding:"required"`
	InterestType    InterestType     `json:"interest_type" binding:"required"`
	PayoutFrequency *PayoutFrequency `json:"payout_frequency,omitempty"`
	IsAutoRenewed   bool             `json:"is_auto_renewed"`
	Notes           string           `json:"notes"`
}
