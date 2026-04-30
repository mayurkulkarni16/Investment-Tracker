package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SIPInstallment struct {
	InstallmentID string    `json:"installment_id" bson:"installment_id"`
	Date          time.Time `json:"date" bson:"date"`
	Amount        float64   `json:"amount" bson:"amount"`
	NAV           float64   `json:"nav" bson:"nav"`
	Units         float64   `json:"units" bson:"units"`
	Status        string    `json:"status" bson:"status"` // success, failed, skipped
}

type SIP struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     string             `json:"user_id" bson:"user_id"`
	FundName   string             `json:"fund_name" bson:"fund_name"`
	FundID     string             `json:"fund_id" bson:"fund_id"` // reference to MF ObjectID
	AMCCode    string             `json:"amc_code" bson:"amc_code"`
	SchemeCode int                `json:"scheme_code" bson:"scheme_code"`

	Amount    float64    `json:"amount" bson:"amount"`
	Frequency string     `json:"frequency" bson:"frequency"` // monthly, weekly, quarterly
	SIPDate   int        `json:"sip_date" bson:"sip_date"`   // day of month
	StartDate time.Time  `json:"start_date" bson:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty" bson:"end_date,omitempty"`

	TotalInstallments     int     `json:"total_installments" bson:"total_installments"`
	CompletedInstallments int     `json:"completed_installments" bson:"completed_installments"`
	MissedInstallments    int     `json:"missed_installments" bson:"missed_installments"`
	TotalInvested         float64 `json:"total_invested" bson:"total_invested"`

	Installments []SIPInstallment `json:"installments" bson:"installments"`

	// Computed
	NextSIPDate  time.Time `json:"next_sip_date" bson:"-"`
	MonthsActive int       `json:"months_active" bson:"-"`

	Status    string    `json:"status" bson:"status"` // active, paused, completed, stopped
	Notes     string    `json:"notes" bson:"notes"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type CreateSIPRequest struct {
	FundName   string  `json:"fund_name" binding:"required"`
	FundID     string  `json:"fund_id"`
	AMCCode    string  `json:"amc_code"`
	SchemeCode int     `json:"scheme_code"`
	Amount     float64 `json:"amount" binding:"required"`
	Frequency  string  `json:"frequency" binding:"required"`
	SIPDate    int     `json:"sip_date" binding:"required"`
	StartDate  string  `json:"start_date" binding:"required"`
	EndDate    string  `json:"end_date"`
	Notes      string  `json:"notes"`
}

type UpdateSIPRequest struct {
	Amount  float64 `json:"amount"`
	SIPDate int     `json:"sip_date"`
	EndDate string  `json:"end_date"`
	Status  string  `json:"status"`
	Notes   string  `json:"notes"`
}

type RecordSIPInstallmentRequest struct {
	Date   string  `json:"date" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
	NAV    float64 `json:"nav"`
	Units  float64 `json:"units"`
	Status string  `json:"status" binding:"required"` // success, failed, skipped
}
