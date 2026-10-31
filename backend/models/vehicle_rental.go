package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VehicleRentalStatus string

const (
	VehicleRentalActive    VehicleRentalStatus = "active"
	VehicleRentalCompleted VehicleRentalStatus = "completed"
)

// RentalOverdueGraceDays is how long after its scheduled date a pending payout
// is still considered "due" rather than "overdue".
const RentalOverdueGraceDays = 5

// RentalPayout is one scheduled monthly rental payment from the fleet operator.
// Payouts are never auto-marked as received; the investor confirms each one so
// that missed payments surface as overdue.
type RentalPayout struct {
	PayoutID       string       `json:"payout_id" bson:"payout_id"`
	ScheduledDate  time.Time    `json:"scheduled_date" bson:"scheduled_date"`
	Amount         float64      `json:"amount" bson:"amount"`
	Status         PayoutStatus `json:"status" bson:"status"`
	ReceivedDate   *time.Time   `json:"received_date,omitempty" bson:"received_date,omitempty"`
	ReceivedAmount float64      `json:"received_amount" bson:"received_amount"`
	IsOverdue      bool         `json:"is_overdue" bson:"-"`
}

type VehicleRental struct {
	ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID               string             `json:"user_id" bson:"user_id"`
	Provider             string             `json:"provider" bson:"provider"`
	VehicleModel         string             `json:"vehicle_model" bson:"vehicle_model"`
	Units                int                `json:"units" bson:"units"`
	UnitPrice            float64            `json:"unit_price" bson:"unit_price"`
	GSTPercent           float64            `json:"gst_percent" bson:"gst_percent"`
	OtherCharges         float64            `json:"other_charges" bson:"other_charges"`
	InvestmentAmount     float64            `json:"investment_amount" bson:"investment_amount"`
	PurchaseDate         time.Time          `json:"purchase_date" bson:"purchase_date"`
	PayoutStartDate      time.Time          `json:"payout_start_date" bson:"payout_start_date"`
	TenureMonths         int                `json:"tenure_months" bson:"tenure_months"`
	MonthlyPayoutPerUnit float64            `json:"monthly_payout_per_unit" bson:"monthly_payout_per_unit"`
	BuybackPercent       float64            `json:"buyback_percent" bson:"buyback_percent"`
	BuybackAmount        float64            `json:"buyback_amount" bson:"buyback_amount"`
	BuybackStatus        PayoutStatus       `json:"buyback_status" bson:"buyback_status"`
	BuybackReceivedDate  *time.Time         `json:"buyback_received_date,omitempty" bson:"buyback_received_date,omitempty"`
	BuybackReceived      float64            `json:"buyback_received" bson:"buyback_received"`
	RegistrationNumbers  []string           `json:"registration_numbers" bson:"registration_numbers"`
	AgreementNumber      string             `json:"agreement_number" bson:"agreement_number"`
	Payouts              []RentalPayout     `json:"payouts" bson:"payouts"`
	Notes                string             `json:"notes" bson:"notes"`
	CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at" bson:"updated_at"`

	// Computed on read (see services.ComputeVehicleRental); not persisted.
	Status               VehicleRentalStatus `json:"status" bson:"-"`
	AgreementEndDate     time.Time           `json:"agreement_end_date" bson:"-"`
	MonthlyPayout        float64             `json:"monthly_payout" bson:"-"`
	ExpectedTotalPayout  float64             `json:"expected_total_payout" bson:"-"`
	TotalPayoutsReceived float64             `json:"total_payouts_received" bson:"-"`
	PayoutsReceivedCount int                 `json:"payouts_received_count" bson:"-"`
	OverdueCount         int                 `json:"overdue_count" bson:"-"`
	OverdueAmount        float64             `json:"overdue_amount" bson:"-"`
	NextPayoutDate       *time.Time          `json:"next_payout_date,omitempty" bson:"-"`
	CurrentValue         float64             `json:"current_value" bson:"-"`
	TotalReturn          float64             `json:"total_return" bson:"-"`
	CapitalRecoveredPct  float64             `json:"capital_recovered_pct" bson:"-"`
	XIRR                 float64             `json:"xirr" bson:"-"`
}

type CreateVehicleRentalRequest struct {
	Provider             string   `json:"provider" binding:"required"`
	VehicleModel         string   `json:"vehicle_model"`
	Units                int      `json:"units" binding:"required,min=1,max=1000"`
	UnitPrice            float64  `json:"unit_price" binding:"required,gt=0"`
	GSTPercent           float64  `json:"gst_percent" binding:"min=0,max=100"`
	OtherCharges         float64  `json:"other_charges" binding:"min=0"`
	PurchaseDate         string   `json:"purchase_date" binding:"required"`
	PayoutStartDate      string   `json:"payout_start_date" binding:"required"`
	TenureMonths         int      `json:"tenure_months" binding:"required,min=1,max=240"`
	MonthlyPayoutPerUnit float64  `json:"monthly_payout_per_unit" binding:"required,gt=0"`
	BuybackPercent       float64  `json:"buyback_percent" binding:"min=0,max=100"`
	RegistrationNumbers  []string `json:"registration_numbers"`
	AgreementNumber      string   `json:"agreement_number"`
	Notes                string   `json:"notes"`
}

// UpdateRentalPayoutRequest marks a payout (or the buyback) as received, or
// reverts it to pending when Status is "pending".
type UpdateRentalPayoutRequest struct {
	Status       PayoutStatus `json:"status" binding:"required,oneof=pending received"`
	ReceivedDate string       `json:"received_date"`
	Amount       float64      `json:"amount" binding:"min=0"`
}
