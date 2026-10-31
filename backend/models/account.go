package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Bank accounts, cash in hand and wallets. Balances are user-entered (or
// observed from an SMS "Avl Bal") snapshots, not live bank feeds.
type BankAccount struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          string             `json:"user_id" bson:"user_id"`
	Name            string             `json:"name" bson:"name"`
	Institution     string             `json:"institution" bson:"institution"`
	AccountType     string             `json:"account_type" bson:"account_type"` // savings, current, salary, cash, wallet, other
	LastFour        string             `json:"last_four" bson:"last_four"`
	Balance         float64            `json:"balance" bson:"balance"`
	BalanceAsOf     time.Time          `json:"balance_as_of" bson:"balance_as_of"`
	BalanceHistory  []BalancePoint     `json:"balance_history" bson:"balance_history"`
	InterestRate    float64            `json:"interest_rate" bson:"interest_rate"`
	IsEmergencyFund bool               `json:"is_emergency_fund" bson:"is_emergency_fund"`
	Status          string             `json:"status" bson:"status"` // active, closed
	Notes           string             `json:"notes" bson:"notes"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`

	// Computed from confirmed ledger entries matching LastFour, current month.
	MonthInflow  float64 `json:"month_inflow" bson:"-"`
	MonthOutflow float64 `json:"month_outflow" bson:"-"`
	DaysStale    int     `json:"days_stale" bson:"-"`
}

type BalancePoint struct {
	Date    time.Time `json:"date" bson:"date"`
	Balance float64   `json:"balance" bson:"balance"`
	Source  string    `json:"source" bson:"source"` // manual, sms
}

type BankAccountRequest struct {
	Name            string  `json:"name" binding:"required"`
	Institution     string  `json:"institution"`
	AccountType     string  `json:"account_type" binding:"required"`
	LastFour        string  `json:"last_four"`
	Balance         float64 `json:"balance"`
	BalanceAsOf     string  `json:"balance_as_of"`
	InterestRate    float64 `json:"interest_rate"`
	IsEmergencyFund bool    `json:"is_emergency_fund"`
	Status          string  `json:"status"`
	Notes           string  `json:"notes"`
}

type UpdateBalanceRequest struct {
	Balance float64 `json:"balance"`
	AsOf    string  `json:"as_of"`
	Source  string  `json:"source"`
}

// Budget is a monthly spending limit for one ledger expense category.
type Budget struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       string             `json:"user_id" bson:"user_id"`
	Category     string             `json:"category" bson:"category"`
	MonthlyLimit float64            `json:"monthly_limit" bson:"monthly_limit"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

type BudgetRequest struct {
	Category     string  `json:"category" binding:"required"`
	MonthlyLimit float64 `json:"monthly_limit" binding:"required,gt=0"`
}

// LedgerMonth summarises confirmed ledger entries for one calendar month (rupees).
type LedgerMonth struct {
	Month          string             `json:"month"` // 2026-10
	Income         float64            `json:"income"`
	Expense        float64            `json:"expense"`
	Bills          float64            `json:"bills"`
	Transfers      float64            `json:"transfers"`
	NetSavings     float64            `json:"net_savings"`
	SavingsRatePct float64            `json:"savings_rate_pct"`
	ByCategory     map[string]float64 `json:"by_category"`
	IncomeByCat    map[string]float64 `json:"income_by_category"`
	EntryCount     int                `json:"entry_count"`
}

type BudgetStatus struct {
	Category  string  `json:"category"`
	Limit     float64 `json:"limit"`
	Spent     float64 `json:"spent"`
	Remaining float64 `json:"remaining"`
	UsedPct   float64 `json:"used_pct"`
	Status    string  `json:"status"` // ok, warning (>=80%), over
}

type LedgerSummary struct {
	Months             []LedgerMonth   `json:"months"` // oldest first
	CurrentMonth       LedgerMonth     `json:"current_month"`
	AvgMonthlyIncome   float64         `json:"avg_monthly_income"`  // last 6 complete+current months with data
	AvgMonthlyExpense  float64         `json:"avg_monthly_expense"` // same window
	AvgSavingsRatePct  float64         `json:"avg_savings_rate_pct"`
	TopCategories      []CategorySpend `json:"top_categories"` // current month
	Budgets            []BudgetStatus  `json:"budgets"`        // current month
	PendingReviewCount int             `json:"pending_review_count"`
}

type CategorySpend struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Pct      float64 `json:"pct"`
}
