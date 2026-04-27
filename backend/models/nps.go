package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NPSAccountType string

const (
	NPSTierI  NPSAccountType = "tier_1"
	NPSTierII NPSAccountType = "tier_2"
)

type NPSFundChoice string

const (
	NPSEquity         NPSFundChoice = "E"
	NPSCorporateBond  NPSFundChoice = "C"
	NPSGovtSecurities NPSFundChoice = "G"
	NPSAlternate      NPSFundChoice = "A"
)

type NPSContribution struct {
	ContributionID string    `json:"contribution_id" bson:"contribution_id"`
	Date           time.Time `json:"date" bson:"date"`
	Amount         float64   `json:"amount" bson:"amount"`
	Type           string    `json:"type" bson:"type"` // self, employer
	FY             string    `json:"fy" bson:"fy"`     // 2025-26
}

type NPSAccount struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AccountHolderName string             `json:"account_holder_name" bson:"account_holder_name"`
	PRAN              string             `json:"pran" bson:"pran"` // Permanent Retirement Account Number
	AccountType       NPSAccountType     `json:"account_type" bson:"account_type"`
	FundManager       string             `json:"fund_manager" bson:"fund_manager"` // SBI, LIC, UTI, HDFC, ICICI, Kotak, Birla
	DateOfJoining     time.Time          `json:"date_of_joining" bson:"date_of_joining"`

	// Asset allocation percentages
	EquityPct        float64 `json:"equity_pct" bson:"equity_pct"`
	CorporateBondPct float64 `json:"corporate_bond_pct" bson:"corporate_bond_pct"`
	GovtSecPct       float64 `json:"govt_sec_pct" bson:"govt_sec_pct"`
	AlternatePct     float64 `json:"alternate_pct" bson:"alternate_pct"`

	Contributions []NPSContribution `json:"contributions" bson:"contributions"`

	TotalSelfContribution     float64 `json:"total_self_contribution" bson:"total_self_contribution"`
	TotalEmployerContribution float64 `json:"total_employer_contribution" bson:"total_employer_contribution"`
	TotalContribution         float64 `json:"total_contribution" bson:"total_contribution"`
	CurrentValue              float64 `json:"current_value" bson:"current_value"`

	// Tax benefits (computed, not stored)
	Section80CCD1  float64 `json:"section_80ccd1" bson:"-"`  // within 80C 1.5L
	Section80CCD1B float64 `json:"section_80ccd1b" bson:"-"` // additional 50K
	Section80CCD2  float64 `json:"section_80ccd2" bson:"-"`  // employer (no limit)

	Status    string    `json:"status" bson:"status"` // active, closed, withdrawn
	Notes     string    `json:"notes" bson:"notes"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type CreateNPSAccountRequest struct {
	AccountHolderName string         `json:"account_holder_name" binding:"required"`
	PRAN              string         `json:"pran" binding:"required"`
	AccountType       NPSAccountType `json:"account_type" binding:"required"`
	FundManager       string         `json:"fund_manager" binding:"required"`
	DateOfJoining     string         `json:"date_of_joining" binding:"required"`
	EquityPct         float64        `json:"equity_pct"`
	CorporateBondPct  float64        `json:"corporate_bond_pct"`
	GovtSecPct        float64        `json:"govt_sec_pct"`
	AlternatePct      float64        `json:"alternate_pct"`
	CurrentValue      float64        `json:"current_value"`
	Notes             string         `json:"notes"`
}

type UpdateNPSAccountRequest struct {
	AccountHolderName string  `json:"account_holder_name"`
	FundManager       string  `json:"fund_manager"`
	EquityPct         float64 `json:"equity_pct"`
	CorporateBondPct  float64 `json:"corporate_bond_pct"`
	GovtSecPct        float64 `json:"govt_sec_pct"`
	AlternatePct      float64 `json:"alternate_pct"`
	CurrentValue      float64 `json:"current_value"`
	Notes             string  `json:"notes"`
}

type AddNPSContributionRequest struct {
	Date   string  `json:"date" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
	Type   string  `json:"type" binding:"required"` // self, employer
}
