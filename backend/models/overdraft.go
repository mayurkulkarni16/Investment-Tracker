package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OverdraftTxnType string

const (
	OverdraftWithdrawal OverdraftTxnType = "withdrawal" // money drawn; increases the principal
	OverdraftRepayment  OverdraftTxnType = "repayment"  // principal paid back
	OverdraftInterest   OverdraftTxnType = "interest"   // interest paid; principal unchanged
)

type OverdraftTransaction struct {
	TxnID  string           `json:"txn_id" bson:"txn_id"`
	Date   time.Time        `json:"date" bson:"date"`
	Type   OverdraftTxnType `json:"type" bson:"type"`
	Amount float64          `json:"amount" bson:"amount"`
	Notes  string           `json:"notes" bson:"notes"`

	// Computed (not stored)
	BalanceAfter float64 `json:"balance_after" bson:"-"`
}

// Overdraft is an interest-only credit line: interest accrues daily on the drawn
// balance and is paid monthly, while principal is repaid whenever the borrower wants.
type Overdraft struct {
	ID              primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID          string                 `json:"user_id" bson:"user_id"`
	LenderName      string                 `json:"lender_name" bson:"lender_name"`
	AccountNumber   string                 `json:"account_number" bson:"account_number"`
	SanctionedLimit float64                `json:"sanctioned_limit" bson:"sanctioned_limit"`
	InterestRate    float64                `json:"interest_rate" bson:"interest_rate"` // annual %
	OpenDate        time.Time              `json:"open_date" bson:"open_date"`
	Status          LoanStatus             `json:"status" bson:"status"` // active, closed
	Notes           string                 `json:"notes" bson:"notes"`
	Transactions    []OverdraftTransaction `json:"transactions" bson:"transactions"`
	CreatedAt       time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" bson:"updated_at"`

	// Computed (not stored)
	Outstanding          float64    `json:"outstanding" bson:"-"`
	AvailableLimit       float64    `json:"available_limit" bson:"-"`
	UtilizationPct       float64    `json:"utilization_pct" bson:"-"`
	TotalWithdrawn       float64    `json:"total_withdrawn" bson:"-"`
	TotalRepaid          float64    `json:"total_repaid" bson:"-"`
	TotalInterestPaid    float64    `json:"total_interest_paid" bson:"-"`
	MonthlyInterest      float64    `json:"monthly_interest" bson:"-"` // at today's balance and rate
	AccruedInterest      float64    `json:"accrued_interest" bson:"-"` // since the last interest payment (estimate)
	LastInterestPaidDate *time.Time `json:"last_interest_paid_date,omitempty" bson:"-"`
}

type CreateOverdraftRequest struct {
	LenderName      string  `json:"lender_name" binding:"required"`
	AccountNumber   string  `json:"account_number"`
	SanctionedLimit float64 `json:"sanctioned_limit" binding:"required,gt=0"`
	InterestRate    float64 `json:"interest_rate" binding:"min=0,max=60"`
	OpenDate        string  `json:"open_date" binding:"required"`
	OpeningBalance  float64 `json:"opening_balance" binding:"min=0"` // amount already drawn on the open date
	Notes           string  `json:"notes"`
}

type UpdateOverdraftRequest struct {
	LenderName      string     `json:"lender_name" binding:"required"`
	AccountNumber   string     `json:"account_number"`
	SanctionedLimit float64    `json:"sanctioned_limit" binding:"required,gt=0"`
	InterestRate    float64    `json:"interest_rate" binding:"min=0,max=60"`
	OpenDate        string     `json:"open_date" binding:"required"`
	Status          LoanStatus `json:"status"`
	Notes           string     `json:"notes"`
}

type AddOverdraftTransactionRequest struct {
	Type   OverdraftTxnType `json:"type" binding:"required,oneof=withdrawal repayment interest"`
	Date   string           `json:"date" binding:"required"`
	Amount float64          `json:"amount" binding:"required,gt=0"`
	Notes  string           `json:"notes"`
}
