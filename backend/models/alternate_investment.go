package models

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Alternate investments are operator-managed physical assets that pay a fixed
// monthly amount: EV fleet leasing, cow ownership schemes and similar.
// Documents live in the "vehicle_rentals" collection and keep the original
// bson field names, so records created before the generalisation still load.

type AlternateInvestmentStatus string

const (
	AltInvestmentActive    AlternateInvestmentStatus = "active"
	AltInvestmentCompleted AlternateInvestmentStatus = "completed"
)

type AltAssetType string

const (
	AltAssetVehicle   AltAssetType = "vehicle"
	AltAssetLivestock AltAssetType = "livestock"
	AltAssetOther     AltAssetType = "other"
)

// AltValuationMethod controls how CurrentValue (book value) is derived.
type AltValuationMethod string

const (
	// Straight-line from purchase cost down to the buyback value over the term.
	AltValuationStraightLine AltValuationMethod = "straight_line"
	// Held at the purchase cost until a buyback is received.
	AltValuationAtCost AltValuationMethod = "at_cost"
	// The investor enters the current value.
	AltValuationManual AltValuationMethod = "manual"
)

// AltPayoutOverdueGraceDays is how long after its scheduled date a pending
// payout is still considered "due" rather than "overdue".
const AltPayoutOverdueGraceDays = 5

// AltOpenEndedLookaheadMonths is how far ahead of today the payout schedule of
// an open-ended investment (no fixed tenure) is generated.
const AltOpenEndedLookaheadMonths = 12

// AltPayout is one scheduled monthly payment from the operator.
// Payouts are never auto-marked as received; the investor confirms each one so
// that missed payments surface as overdue.
type AltPayout struct {
	PayoutID       string       `json:"payout_id" bson:"payout_id"`
	ScheduledDate  time.Time    `json:"scheduled_date" bson:"scheduled_date"`
	Amount         float64      `json:"amount" bson:"amount"`
	Status         PayoutStatus `json:"status" bson:"status"`
	ReceivedDate   *time.Time   `json:"received_date,omitempty" bson:"received_date,omitempty"`
	ReceivedAmount float64      `json:"received_amount" bson:"received_amount"`
	IsOverdue      bool         `json:"is_overdue" bson:"-"`
}

type AlternateInvestment struct {
	ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID               string             `json:"user_id" bson:"user_id"`
	AssetType            AltAssetType       `json:"asset_type" bson:"asset_type,omitempty"`
	Provider             string             `json:"provider" bson:"provider"`
	AssetName            string             `json:"asset_name" bson:"vehicle_model"`
	Units                int                `json:"units" bson:"units"`
	UnitPrice            float64            `json:"unit_price" bson:"unit_price"`
	GSTPercent           float64            `json:"gst_percent" bson:"gst_percent"`
	OtherCharges         float64            `json:"other_charges" bson:"other_charges"`
	InvestmentAmount     float64            `json:"investment_amount" bson:"investment_amount"`
	PurchaseDate         time.Time          `json:"purchase_date" bson:"purchase_date"`
	PayoutStartDate      time.Time          `json:"payout_start_date" bson:"payout_start_date"`
	TenureMonths         int                `json:"tenure_months" bson:"tenure_months"` // 0 = open-ended
	MonthlyPayoutPerUnit float64            `json:"monthly_payout_per_unit" bson:"monthly_payout_per_unit"`
	BuybackPercent       float64            `json:"buyback_percent" bson:"buyback_percent"`
	BuybackAmount        float64            `json:"buyback_amount" bson:"buyback_amount"`
	BuybackStatus        PayoutStatus       `json:"buyback_status" bson:"buyback_status"`
	BuybackReceivedDate  *time.Time         `json:"buyback_received_date,omitempty" bson:"buyback_received_date,omitempty"`
	BuybackReceived      float64            `json:"buyback_received" bson:"buyback_received"`
	ValuationMethod      AltValuationMethod `json:"valuation_method" bson:"valuation_method,omitempty"`
	ManualValue          float64            `json:"manual_value" bson:"manual_value"`
	// Livestock only: stated milk yield and price per animal, used to sanity-check the payout.
	DailyYieldLitres float64     `json:"daily_yield_litres" bson:"daily_yield_litres"`
	PricePerLitre    float64     `json:"price_per_litre" bson:"price_per_litre"`
	Identifiers      []string    `json:"identifiers" bson:"registration_numbers"`
	AgreementNumber  string      `json:"agreement_number" bson:"agreement_number"`
	Payouts          []AltPayout `json:"payouts" bson:"payouts"`
	Notes            string      `json:"notes" bson:"notes"`
	CreatedAt        time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" bson:"updated_at"`

	// Computed on read (see services.ComputeAlternateInvestment); not persisted.
	Status               AlternateInvestmentStatus `json:"status" bson:"-"`
	OpenEnded            bool                      `json:"open_ended" bson:"-"`
	AgreementEndDate     time.Time                 `json:"agreement_end_date" bson:"-"`
	MonthlyPayout        float64                   `json:"monthly_payout" bson:"-"`
	ExpectedTotalPayout  float64                   `json:"expected_total_payout" bson:"-"`
	TotalPayoutsReceived float64                   `json:"total_payouts_received" bson:"-"`
	PayoutsReceivedCount int                       `json:"payouts_received_count" bson:"-"`
	OverdueCount         int                       `json:"overdue_count" bson:"-"`
	OverdueAmount        float64                   `json:"overdue_amount" bson:"-"`
	NextPayoutDate       *time.Time                `json:"next_payout_date,omitempty" bson:"-"`
	CurrentValue         float64                   `json:"current_value" bson:"-"`
	TotalReturn          float64                   `json:"total_return" bson:"-"`
	CapitalRecoveredPct  float64                   `json:"capital_recovered_pct" bson:"-"`
	BreakEvenMonths      float64                   `json:"break_even_months" bson:"-"`
	GrossMonthlyRevenue  float64                   `json:"gross_monthly_revenue" bson:"-"`
	XIRR                 float64                   `json:"xirr" bson:"-"`
}

// UnmarshalJSON also accepts the pre-rename "vehicle_model" and
// "registration_numbers" keys so older backups restore without data loss.
func (a *AlternateInvestment) UnmarshalJSON(data []byte) error {
	type plain AlternateInvestment
	aux := struct {
		*plain
		LegacyVehicleModel        string   `json:"vehicle_model"`
		LegacyRegistrationNumbers []string `json:"registration_numbers"`
	}{plain: (*plain)(a)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if a.AssetName == "" {
		a.AssetName = aux.LegacyVehicleModel
	}
	if len(a.Identifiers) == 0 {
		a.Identifiers = aux.LegacyRegistrationNumbers
	}
	return nil
}

type CreateAlternateInvestmentRequest struct {
	AssetType            AltAssetType       `json:"asset_type" binding:"omitempty,oneof=vehicle livestock other"`
	Provider             string             `json:"provider" binding:"required"`
	AssetName            string             `json:"asset_name"`
	Units                int                `json:"units" binding:"required,min=1,max=1000"`
	UnitPrice            float64            `json:"unit_price" binding:"required,gt=0"`
	GSTPercent           float64            `json:"gst_percent" binding:"min=0,max=100"`
	OtherCharges         float64            `json:"other_charges" binding:"min=0"`
	PurchaseDate         string             `json:"purchase_date" binding:"required"`
	PayoutStartDate      string             `json:"payout_start_date" binding:"required"`
	TenureMonths         int                `json:"tenure_months" binding:"min=0,max=240"`
	MonthlyPayoutPerUnit float64            `json:"monthly_payout_per_unit" binding:"required,gt=0"`
	BuybackPercent       float64            `json:"buyback_percent" binding:"min=0,max=100"`
	ValuationMethod      AltValuationMethod `json:"valuation_method" binding:"omitempty,oneof=straight_line at_cost manual"`
	ManualValue          float64            `json:"manual_value" binding:"min=0"`
	DailyYieldLitres     float64            `json:"daily_yield_litres" binding:"min=0"`
	PricePerLitre        float64            `json:"price_per_litre" binding:"min=0"`
	Identifiers          []string           `json:"identifiers"`
	AgreementNumber      string             `json:"agreement_number"`
	Notes                string             `json:"notes"`
}

// UpdateAltPayoutRequest marks a payout (or the buyback) as received, or
// reverts it to pending when Status is "pending".
type UpdateAltPayoutRequest struct {
	Status       PayoutStatus `json:"status" binding:"required,oneof=pending received"`
	ReceivedDate string       `json:"received_date"`
	Amount       float64      `json:"amount" binding:"min=0"`
}
