package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CardTransaction struct {
	TransactionID string    `json:"transaction_id" bson:"transaction_id"`
	Date          time.Time `json:"date" bson:"date"`
	Description   string    `json:"description" bson:"description"`
	Amount        float64   `json:"amount" bson:"amount"`
	Category      string    `json:"category" bson:"category"` // food, shopping, travel, fuel, bills, emi, entertainment, health, others
	IsEMI         bool      `json:"is_emi" bson:"is_emi"`
	EMIReference  string    `json:"emi_reference,omitempty" bson:"emi_reference,omitempty"`
}

type CardStatement struct {
	StatementID   string            `json:"statement_id" bson:"statement_id"`
	Month         string            `json:"month" bson:"month"` // 2026-04
	StatementDate time.Time         `json:"statement_date" bson:"statement_date"`
	DueDate       time.Time         `json:"due_date" bson:"due_date"`
	TotalAmount   float64           `json:"total_amount" bson:"total_amount"`
	MinimumDue    float64           `json:"minimum_due" bson:"minimum_due"`
	AmountPaid    float64           `json:"amount_paid" bson:"amount_paid"`
	PaidDate      *time.Time        `json:"paid_date,omitempty" bson:"paid_date,omitempty"`
	Transactions  []CardTransaction `json:"transactions" bson:"transactions"`
	IsPaid        bool              `json:"is_paid" bson:"is_paid"`
	PaidFull      bool              `json:"paid_full" bson:"paid_full"`
}

type CardEMI struct {
	EMIID           string    `json:"emi_id" bson:"emi_id"`
	Description     string    `json:"description" bson:"description"`
	MerchantName    string    `json:"merchant_name" bson:"merchant_name"`
	OriginalAmount  float64   `json:"original_amount" bson:"original_amount"`
	EMIAmount       float64   `json:"emi_amount" bson:"emi_amount"`
	TenureMonths    int       `json:"tenure_months" bson:"tenure_months"`
	RemainingMonths int       `json:"remaining_months" bson:"remaining_months"`
	InterestRate    float64   `json:"interest_rate" bson:"interest_rate"` // 0 for no-cost EMI
	ProcessingFee   float64   `json:"processing_fee" bson:"processing_fee"`
	StartDate       time.Time `json:"start_date" bson:"start_date"`
	TotalPaid       float64   `json:"total_paid" bson:"total_paid"`
	Status          string    `json:"status" bson:"status"` // active, completed, foreclosed
}

type CreditScoreEntry struct {
	ScoreID string    `json:"score_id" bson:"score_id"`
	Date    time.Time `json:"date" bson:"date"`
	Score   int       `json:"score" bson:"score"`   // 300-900
	Bureau  string    `json:"bureau" bson:"bureau"` // CIBIL, Experian, Equifax, CRIF
	Notes   string    `json:"notes" bson:"notes"`
}

type CreditCard struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CardName       string             `json:"card_name" bson:"card_name"`
	BankName       string             `json:"bank_name" bson:"bank_name"`
	CardNetwork    string             `json:"card_network" bson:"card_network"` // visa, mastercard, rupay, amex
	LastFourDigits string             `json:"last_four_digits" bson:"last_four_digits"`
	CardHolderName string             `json:"card_holder_name" bson:"card_holder_name"`
	CreditLimit    float64            `json:"credit_limit" bson:"credit_limit"`
	BillingDate    int                `json:"billing_date" bson:"billing_date"`       // day of month
	DueDateOffset  int                `json:"due_date_offset" bson:"due_date_offset"` // days after billing
	AnnualFee      float64            `json:"annual_fee" bson:"annual_fee"`
	JoiningDate    time.Time          `json:"joining_date" bson:"joining_date"`

	CurrentOutstanding float64 `json:"current_outstanding" bson:"current_outstanding"`
	AvailableCredit    float64 `json:"available_credit" bson:"available_credit"`
	UtilizationPct     float64 `json:"utilization_pct" bson:"-"` // computed

	RewardPoints     float64 `json:"reward_points" bson:"reward_points"`
	RewardPointValue float64 `json:"reward_point_value" bson:"reward_point_value"` // ₹ per point

	CardEMIs     []CardEMI          `json:"card_emis" bson:"card_emis"`
	Statements   []CardStatement    `json:"statements" bson:"statements"`
	CreditScores []CreditScoreEntry `json:"credit_scores" bson:"credit_scores"`

	// Spending analytics (computed)
	TotalSpentThisMonth float64            `json:"total_spent_this_month" bson:"-"`
	SpendByCategory     map[string]float64 `json:"spend_by_category" bson:"-"`
	CreditScoreTips     []string           `json:"credit_score_tips" bson:"-"`

	Status    string    `json:"status" bson:"status"` // active, closed, blocked
	Notes     string    `json:"notes" bson:"notes"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type CreateCreditCardRequest struct {
	CardName         string  `json:"card_name" binding:"required"`
	BankName         string  `json:"bank_name" binding:"required"`
	CardNetwork      string  `json:"card_network" binding:"required"`
	LastFourDigits   string  `json:"last_four_digits"`
	CardHolderName   string  `json:"card_holder_name"`
	CreditLimit      float64 `json:"credit_limit" binding:"required"`
	BillingDate      int     `json:"billing_date" binding:"required"`
	DueDateOffset    int     `json:"due_date_offset"`
	AnnualFee        float64 `json:"annual_fee"`
	JoiningDate      string  `json:"joining_date" binding:"required"`
	RewardPointValue float64 `json:"reward_point_value"`
	Notes            string  `json:"notes"`
}

type UpdateCreditCardRequest struct {
	CardName           string  `json:"card_name"`
	CreditLimit        float64 `json:"credit_limit"`
	BillingDate        int     `json:"billing_date"`
	DueDateOffset      int     `json:"due_date_offset"`
	AnnualFee          float64 `json:"annual_fee"`
	RewardPoints       float64 `json:"reward_points"`
	CurrentOutstanding float64 `json:"current_outstanding"`
	Notes              string  `json:"notes"`
}

type AddCardStatementRequest struct {
	Month         string  `json:"month" binding:"required"` // 2026-04
	StatementDate string  `json:"statement_date" binding:"required"`
	DueDate       string  `json:"due_date" binding:"required"`
	TotalAmount   float64 `json:"total_amount" binding:"required"`
	MinimumDue    float64 `json:"minimum_due" binding:"required"`
}

type PayStatementRequest struct {
	StatementID string  `json:"statement_id" binding:"required"`
	AmountPaid  float64 `json:"amount_paid" binding:"required"`
	PaidDate    string  `json:"paid_date" binding:"required"`
}

type AddCardTransactionRequest struct {
	StatementID string  `json:"statement_id" binding:"required"`
	Date        string  `json:"date" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	Category    string  `json:"category" binding:"required"`
}

type AddCardEMIRequest struct {
	Description    string  `json:"description" binding:"required"`
	MerchantName   string  `json:"merchant_name"`
	OriginalAmount float64 `json:"original_amount" binding:"required"`
	TenureMonths   int     `json:"tenure_months" binding:"required"`
	InterestRate   float64 `json:"interest_rate"`
	ProcessingFee  float64 `json:"processing_fee"`
	StartDate      string  `json:"start_date" binding:"required"`
}

type AddCreditScoreRequest struct {
	Date   string `json:"date" binding:"required"`
	Score  int    `json:"score" binding:"required"`
	Bureau string `json:"bureau" binding:"required"`
	Notes  string `json:"notes"`
}
