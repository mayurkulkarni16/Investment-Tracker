package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GoldHolding covers physical gold, digital gold, Sovereign Gold Bonds, gold ETFs and gold funds.
type GoldHolding struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         string             `json:"user_id" bson:"user_id"`
	Name           string             `json:"name" bson:"name"`
	Form           string             `json:"form" bson:"form"`                     // physical, digital, sgb, etf, mutual_fund
	Quantity       float64            `json:"quantity" bson:"quantity"`             // grams (physical/digital/sgb) or units (etf/mutual_fund)
	Purity         float64            `json:"purity" bson:"purity"`                 // karat: 24, 22, 18 (physical); 24 for others
	PurchasePrice  float64            `json:"purchase_price" bson:"purchase_price"` // total cost incl. making charges
	PurchaseDate   time.Time          `json:"purchase_date" bson:"purchase_date"`
	CurrentPrice   float64            `json:"current_price" bson:"current_price"` // per gram (of this purity) or per unit
	PriceUpdatedAt *time.Time         `json:"price_updated_at,omitempty" bson:"price_updated_at,omitempty"`
	Symbol         string             `json:"symbol" bson:"symbol"`               // NSE symbol for ETFs, e.g. GOLDBEES
	InterestRate   float64            `json:"interest_rate" bson:"interest_rate"` // SGB: 2.5
	MaturityDate   *time.Time         `json:"maturity_date,omitempty" bson:"maturity_date,omitempty"`
	Notes          string             `json:"notes" bson:"notes"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`

	// Computed
	CurrentValue   float64 `json:"current_value" bson:"-"`
	GainLoss       float64 `json:"gain_loss" bson:"-"`
	GainLossPct    float64 `json:"gain_loss_pct" bson:"-"`
	CAGR           float64 `json:"cagr" bson:"-"`
	InterestEarned float64 `json:"interest_earned" bson:"-"` // SGB interest paid so far (estimate)
}

type GoldHoldingRequest struct {
	Name          string  `json:"name" binding:"required"`
	Form          string  `json:"form" binding:"required"`
	Quantity      float64 `json:"quantity" binding:"required,gt=0"`
	Purity        float64 `json:"purity"`
	PurchasePrice float64 `json:"purchase_price" binding:"min=0"`
	PurchaseDate  string  `json:"purchase_date" binding:"required"`
	CurrentPrice  float64 `json:"current_price" binding:"min=0"`
	Symbol        string  `json:"symbol"`
	InterestRate  float64 `json:"interest_rate"`
	MaturityDate  string  `json:"maturity_date"`
	Notes         string  `json:"notes"`
}

// GoldRateRequest sets today's 24K price per gram; physical, digital and SGB
// holdings are revalued pro rata by purity.
type GoldRateRequest struct {
	Rate24KPerGram float64 `json:"rate_24k_per_gram" binding:"required,gt=0"`
}

// Property is a real-estate holding (home, land, commercial).
type Property struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         string             `json:"user_id" bson:"user_id"`
	Name           string             `json:"name" bson:"name"`
	PropertyType   string             `json:"property_type" bson:"property_type"` // residential, commercial, land, other
	Location       string             `json:"location" bson:"location"`
	PurchasePrice  float64            `json:"purchase_price" bson:"purchase_price"` // incl. stamp duty and registration
	PurchaseDate   time.Time          `json:"purchase_date" bson:"purchase_date"`
	CurrentValue   float64            `json:"current_value" bson:"current_value"` // full property value
	ValueUpdatedAt time.Time          `json:"value_updated_at" bson:"value_updated_at"`
	OwnershipPct   float64            `json:"ownership_pct" bson:"ownership_pct"`
	HomeLoanID     string             `json:"home_loan_id" bson:"home_loan_id"`
	IsSelfOccupied bool               `json:"is_self_occupied" bson:"is_self_occupied"`
	MonthlyRent    float64            `json:"monthly_rent" bson:"monthly_rent"`
	AnnualCosts    float64            `json:"annual_costs" bson:"annual_costs"` // maintenance, property tax
	Notes          string             `json:"notes" bson:"notes"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`

	// Computed
	OwnedValue      float64 `json:"owned_value" bson:"-"`
	GainLoss        float64 `json:"gain_loss" bson:"-"`
	CAGR            float64 `json:"cagr" bson:"-"`
	NetRentalYield  float64 `json:"net_rental_yield" bson:"-"` // % of current value
	LoanOutstanding float64 `json:"loan_outstanding" bson:"-"`
	Equity          float64 `json:"equity" bson:"-"` // owned value - loan outstanding
}

type PropertyRequest struct {
	Name           string  `json:"name" binding:"required"`
	PropertyType   string  `json:"property_type" binding:"required"`
	Location       string  `json:"location"`
	PurchasePrice  float64 `json:"purchase_price" binding:"min=0"`
	PurchaseDate   string  `json:"purchase_date" binding:"required"`
	CurrentValue   float64 `json:"current_value" binding:"min=0"`
	ValueUpdatedAt string  `json:"value_updated_at"`
	OwnershipPct   float64 `json:"ownership_pct"`
	HomeLoanID     string  `json:"home_loan_id"`
	IsSelfOccupied bool    `json:"is_self_occupied"`
	MonthlyRent    float64 `json:"monthly_rent" binding:"min=0"`
	AnnualCosts    float64 `json:"annual_costs" binding:"min=0"`
	Notes          string  `json:"notes"`
}
