package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PayoutFrequency string

const (
	PayoutMonthly    PayoutFrequency = "monthly"
	PayoutQuarterly  PayoutFrequency = "quarterly"
	PayoutBiannually PayoutFrequency = "biannually"
	PayoutAnnually   PayoutFrequency = "annually"
	PayoutAtMaturity PayoutFrequency = "at_maturity"
)

type MaturityType string

const (
	MaturityBullet    MaturityType = "bullet"
	MaturityStaggered MaturityType = "staggered"
)

type PayoutStatus string

const (
	PayoutPending  PayoutStatus = "pending"
	PayoutReceived PayoutStatus = "received"
)

type BondStatus string

const (
	BondActive           BondStatus = "active"
	BondMatured          BondStatus = "matured"
	BondPartiallyMatured BondStatus = "partially_matured"
)

type PrincipalRepayment struct {
	RepaymentID   string       `json:"repayment_id" bson:"repayment_id"`
	ScheduledDate time.Time    `json:"scheduled_date" bson:"scheduled_date"`
	Amount        float64      `json:"amount" bson:"amount"`
	Status        PayoutStatus `json:"status" bson:"status"`
	ReceivedDate  *time.Time   `json:"received_date,omitempty" bson:"received_date,omitempty"`
}

type InterestPayout struct {
	PayoutID        string       `json:"payout_id" bson:"payout_id"`
	ScheduledDate   time.Time    `json:"scheduled_date" bson:"scheduled_date"`
	PrincipalAtTime float64      `json:"principal_at_time" bson:"principal_at_time"`
	Amount          float64      `json:"amount" bson:"amount"`
	Status          PayoutStatus `json:"status" bson:"status"`
	ReceivedDate    *time.Time   `json:"received_date,omitempty" bson:"received_date,omitempty"`
}

type CorporateBond struct {
	ID                     primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID                 string               `json:"user_id" bson:"user_id"`
	BondName               string               `json:"bond_name" bson:"bond_name"`
	Issuer                 string               `json:"issuer" bson:"issuer"`
	PurchaseDate           time.Time            `json:"purchase_date" bson:"purchase_date"`
	InvestmentAmount       float64              `json:"investment_amount" bson:"investment_amount"`
	FaceValue              float64              `json:"face_value" bson:"face_value"`
	CouponRate             float64              `json:"coupon_rate" bson:"coupon_rate"`
	YieldToMaturity        float64              `json:"yield_to_maturity" bson:"yield_to_maturity"`
	InterestPayoutFreq     PayoutFrequency      `json:"interest_payout_frequency" bson:"interest_payout_frequency"`
	PayoutDay              int                  `json:"payout_day" bson:"payout_day"`
	MaturityDate           time.Time            `json:"maturity_date" bson:"maturity_date"`
	MaturityType           MaturityType         `json:"maturity_type" bson:"maturity_type"`
	PrincipalRepayments    []PrincipalRepayment `json:"principal_repayments" bson:"principal_repayments"`
	InterestPayouts        []InterestPayout     `json:"interest_payouts" bson:"interest_payouts"`
	RemainingPrincipal     float64              `json:"remaining_principal" bson:"remaining_principal"`
	TotalInterestEarned    float64              `json:"total_interest_earned" bson:"total_interest_earned"`
	TotalPrincipalReturned float64              `json:"total_principal_returned" bson:"total_principal_returned"`
	XIRR                   float64              `json:"xirr" bson:"-"`
	Status                 BondStatus           `json:"status" bson:"status"`
	Notes                  string               `json:"notes" bson:"notes"`
	CreatedAt              time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt              time.Time            `json:"updated_at" bson:"updated_at"`
}

type PrincipalRepaymentInput struct {
	ScheduledDate string  `json:"scheduled_date" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
}

type CreateCorporateBondRequest struct {
	BondName            string                    `json:"bond_name" binding:"required"`
	Issuer              string                    `json:"issuer" binding:"required"`
	PurchaseDate        string                    `json:"purchase_date" binding:"required"`
	InvestmentAmount    float64                   `json:"investment_amount" binding:"required"`
	FaceValue           float64                   `json:"face_value" binding:"min=0"`
	CouponRate          float64                   `json:"coupon_rate" binding:"min=0,max=100"` // 0 = zero-coupon
	YieldToMaturity     float64                   `json:"yield_to_maturity" binding:"min=0,max=100"`
	InterestPayoutFreq  PayoutFrequency           `json:"interest_payout_frequency" binding:"required"`
	PayoutDay           int                       `json:"payout_day" binding:"min=0,max=31"`
	MaturityDate        string                    `json:"maturity_date" binding:"required"`
	MaturityType        MaturityType              `json:"maturity_type" binding:"required"`
	PrincipalRepayments []PrincipalRepaymentInput `json:"principal_repayments"`
	Notes               string                    `json:"notes"`
}
