package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionType string

const (
	TransactionPurchase   TransactionType = "purchase"
	TransactionSIP        TransactionType = "sip"
	TransactionRedemption TransactionType = "redemption"
	TransactionSwitchIn   TransactionType = "switch_in"
	TransactionSwitchOut  TransactionType = "switch_out"
)

type FundType string

const (
	FundTypeEquity FundType = "Equity"
	FundTypeDebt   FundType = "Debt"
	FundTypeHybrid FundType = "Hybrid"
	FundTypeELSS   FundType = "ELSS"
	FundTypeIndex  FundType = "Index"
	FundTypeLiquid FundType = "Liquid"
)

type MFTransaction struct {
	TransactionID string          `json:"transaction_id" bson:"transaction_id"`
	Date          time.Time       `json:"date" bson:"date"`
	Type          TransactionType `json:"type" bson:"type"`
	Amount        float64         `json:"amount" bson:"amount"`
	NAVAtPurchase float64         `json:"nav_at_purchase" bson:"nav_at_purchase"`
	Units         float64         `json:"units" bson:"units"`
	LockInEnd     *time.Time      `json:"lock_in_end,omitempty" bson:"lock_in_end,omitempty"`
}

type MutualFund struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FundName        string             `json:"fund_name" bson:"fund_name"`
	AMC             string             `json:"amc" bson:"amc"`
	FundType        FundType           `json:"fund_type" bson:"fund_type"`
	SchemeCode      string             `json:"scheme_code" bson:"scheme_code"`
	FolioNumber     string             `json:"folio_number" bson:"folio_number"`
	IsELSS          bool               `json:"is_elss" bson:"is_elss"`
	LockInEndDate   *time.Time         `json:"lock_in_end_date,omitempty" bson:"lock_in_end_date,omitempty"`
	Transactions    []MFTransaction    `json:"transactions" bson:"transactions"`
	TotalUnits      float64            `json:"total_units" bson:"total_units"`
	TotalInvested   float64            `json:"total_invested" bson:"total_invested"`
	CurrentNAV      float64            `json:"current_nav" bson:"current_nav"`
	CurrentValue    float64            `json:"current_value" bson:"current_value"`
	NAVLastUpdated  *time.Time         `json:"nav_last_updated,omitempty" bson:"nav_last_updated,omitempty"`
	GainLoss        float64            `json:"gain_loss" bson:"gain_loss"`
	GainLossPercent float64            `json:"gain_loss_percent" bson:"gain_loss_percent"`
	Notes           string             `json:"notes" bson:"notes"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreateMutualFundRequest struct {
	FundName    string   `json:"fund_name" binding:"required"`
	AMC         string   `json:"amc" binding:"required"`
	FundType    FundType `json:"fund_type" binding:"required"`
	SchemeCode  string   `json:"scheme_code" binding:"required"`
	FolioNumber string   `json:"folio_number"`
	IsELSS      bool     `json:"is_elss"`
	Notes       string   `json:"notes"`
}

type AddMFTransactionRequest struct {
	Date          string          `json:"date" binding:"required"`
	Type          TransactionType `json:"type" binding:"required"`
	Amount        float64         `json:"amount" binding:"required"`
	NAVAtPurchase float64         `json:"nav_at_purchase" binding:"required"`
	Units         float64         `json:"units"`
}

type UpdateMutualFundRequest struct {
	FundName    string   `json:"fund_name"`
	AMC         string   `json:"amc"`
	FundType    FundType `json:"fund_type"`
	SchemeCode  string   `json:"scheme_code"`
	FolioNumber string   `json:"folio_number"`
	IsELSS      bool     `json:"is_elss"`
	Notes       string   `json:"notes"`
}

// CAS Import types
type ImportCASTransaction struct {
	Date   string  `json:"date"`
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
	NAV    float64 `json:"nav"`
	Units  float64 `json:"units"`
}

type ImportCASFund struct {
	FundName     string                 `json:"fund_name"`
	Category     string                 `json:"category"`
	FolioNumber  string                 `json:"folio_number"`
	Transactions []ImportCASTransaction `json:"transactions"`
}

type ImportCASRequest struct {
	Funds []ImportCASFund `json:"funds"`
}

type ImportResult struct {
	FundsCreated      int      `json:"funds_created"`
	FundsUpdated      int      `json:"funds_updated"`
	TransactionsAdded int      `json:"transactions_added"`
	Errors            []string `json:"errors,omitempty"`
}
