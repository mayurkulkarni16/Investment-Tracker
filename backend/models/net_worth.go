package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NetWorthSnapshot struct {
	ID    primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Date  time.Time          `json:"date" bson:"date"`
	Month string             `json:"month" bson:"month"` // 2026-04

	// Assets
	MutualFunds    float64 `json:"mutual_funds" bson:"mutual_funds"`
	Stocks         float64 `json:"stocks" bson:"stocks"`
	FixedDeposits  float64 `json:"fixed_deposits" bson:"fixed_deposits"`
	ProvidentFund  float64 `json:"provident_fund" bson:"provident_fund"`
	NPS            float64 `json:"nps" bson:"nps"`
	CorporateBonds float64 `json:"corporate_bonds" bson:"corporate_bonds"`
	OtherAssets    float64 `json:"other_assets" bson:"other_assets"`
	TotalAssets    float64 `json:"total_assets" bson:"total_assets"`

	// Liabilities
	HomeLoans             float64 `json:"home_loans" bson:"home_loans"`
	PersonalLoans         float64 `json:"personal_loans" bson:"personal_loans"`
	CreditCardOutstanding float64 `json:"credit_card_outstanding" bson:"credit_card_outstanding"`
	OtherLiabilities      float64 `json:"other_liabilities" bson:"other_liabilities"`
	TotalLiabilities      float64 `json:"total_liabilities" bson:"total_liabilities"`

	NetWorth  float64   `json:"net_worth" bson:"net_worth"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type NetWorthCurrent struct {
	Assets           map[string]float64 `json:"assets"`
	Liabilities      map[string]float64 `json:"liabilities"`
	TotalAssets      float64            `json:"total_assets"`
	TotalLiabilities float64            `json:"total_liabilities"`
	NetWorth         float64            `json:"net_worth"`
}
