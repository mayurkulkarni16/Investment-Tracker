package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InsurancePolicy covers term, health, life (endowment/ULIP), vehicle and other policies.
type InsurancePolicy struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         string             `json:"user_id" bson:"user_id"`
	PolicyType     string             `json:"policy_type" bson:"policy_type"` // term, health, life, ulip, vehicle, home, personal_accident, other
	Insurer        string             `json:"insurer" bson:"insurer"`
	PolicyName     string             `json:"policy_name" bson:"policy_name"`
	PolicyNumber   string             `json:"policy_number" bson:"policy_number"`
	InsuredPersons string             `json:"insured_persons" bson:"insured_persons"` // free text, e.g. "Self, Spouse"
	// HealthCoverFor decides the 80D bucket of a health policy: "self_family"
	// (self, spouse, children; default) or "parents".
	HealthCoverFor   string     `json:"health_cover_for" bson:"health_cover_for,omitempty"`
	InsuredSenior    bool       `json:"insured_senior" bson:"insured_senior,omitempty"` // a covered person is 60+ (higher 80D limit)
	SumAssured       float64    `json:"sum_assured" bson:"sum_assured"`
	PremiumAmount    float64    `json:"premium_amount" bson:"premium_amount"`
	PremiumFrequency string     `json:"premium_frequency" bson:"premium_frequency"` // monthly, quarterly, half_yearly, yearly, single
	StartDate        time.Time  `json:"start_date" bson:"start_date"`
	EndDate          *time.Time `json:"end_date,omitempty" bson:"end_date,omitempty"`                 // cover end / maturity
	PremiumEndDate   *time.Time `json:"premium_end_date,omitempty" bson:"premium_end_date,omitempty"` // limited-pay policies
	CurrentValue     float64    `json:"current_value" bson:"current_value"`                           // surrender/fund value (endowment, ULIP)
	Nominee          string     `json:"nominee" bson:"nominee"`
	Status           string     `json:"status" bson:"status"` // active, lapsed, matured, surrendered, expired
	Notes            string     `json:"notes" bson:"notes"`
	CreatedAt        time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" bson:"updated_at"`

	// Computed
	NextPremiumDate *time.Time `json:"next_premium_date,omitempty" bson:"-"`
	AnnualPremium   float64    `json:"annual_premium" bson:"-"`
	TaxSection      string     `json:"tax_section" bson:"-"` // 80C, 80D or ""
	DaysToExpiry    *int       `json:"days_to_expiry,omitempty" bson:"-"`
}

type InsurancePolicyRequest struct {
	PolicyType       string  `json:"policy_type" binding:"required"`
	Insurer          string  `json:"insurer" binding:"required"`
	PolicyName       string  `json:"policy_name"`
	PolicyNumber     string  `json:"policy_number"`
	InsuredPersons   string  `json:"insured_persons"`
	HealthCoverFor   string  `json:"health_cover_for"`
	InsuredSenior    bool    `json:"insured_senior"`
	SumAssured       float64 `json:"sum_assured" binding:"min=0"`
	PremiumAmount    float64 `json:"premium_amount" binding:"min=0"`
	PremiumFrequency string  `json:"premium_frequency" binding:"required"`
	StartDate        string  `json:"start_date" binding:"required"`
	EndDate          string  `json:"end_date"`
	PremiumEndDate   string  `json:"premium_end_date"`
	CurrentValue     float64 `json:"current_value" binding:"min=0"`
	Nominee          string  `json:"nominee"`
	Status           string  `json:"status"`
	Notes            string  `json:"notes"`
}

type InsuranceSummary struct {
	TotalLifeCover         float64           `json:"total_life_cover"`   // term + life + ulip, active
	TotalHealthCover       float64           `json:"total_health_cover"` // health, active
	AnnualPremium          float64           `json:"annual_premium"`
	Premium80CThisFY       float64           `json:"premium_80c_this_fy"`
	Premium80DThisFY       float64           `json:"premium_80d_this_fy"`
	CashValue              float64           `json:"cash_value"`
	AnnualIncome           float64           `json:"annual_income"`        // used for adequacy
	AnnualIncomeSource     string            `json:"annual_income_source"` // ledger, query, none
	OutstandingLoans       float64           `json:"outstanding_loans"`
	RecommendedLifeCover   float64           `json:"recommended_life_cover"` // 10x income + loans
	LifeCoverGap           float64           `json:"life_cover_gap"`
	RecommendedHealthCover float64           `json:"recommended_health_cover"`
	HealthCoverGap         float64           `json:"health_cover_gap"`
	UpcomingPremiums       []UpcomingPremium `json:"upcoming_premiums"` // next 60 days
	Suggestions            []string          `json:"suggestions"`
}

type UpcomingPremium struct {
	PolicyID   string  `json:"policy_id"`
	PolicyName string  `json:"policy_name"`
	Insurer    string  `json:"insurer"`
	Date       string  `json:"date"`
	Amount     float64 `json:"amount"`
	DaysLeft   int     `json:"days_left"`
}
