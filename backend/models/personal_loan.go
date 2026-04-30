package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonalLoan struct {
	ID                    primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID                string             `json:"user_id" bson:"user_id"`
	LenderName            string             `json:"lender_name" bson:"lender_name"`
	LoanAccountNumber     string             `json:"loan_account_number" bson:"loan_account_number"`
	LoanPurpose           string             `json:"loan_purpose" bson:"loan_purpose"` // personal, education, vehicle, medical, wedding, travel, other
	PrincipalAmount       float64            `json:"principal_amount" bson:"principal_amount"`
	DisbursedAmount       float64            `json:"disbursed_amount" bson:"disbursed_amount"`
	InterestRate          float64            `json:"interest_rate" bson:"interest_rate"`
	RateType              LoanRateType       `json:"rate_type" bson:"rate_type"`
	TenureMonths          int                `json:"tenure_months" bson:"tenure_months"`
	EMIAmount             float64            `json:"emi_amount" bson:"emi_amount"`
	EMIStartDate          time.Time          `json:"emi_start_date" bson:"emi_start_date"`
	DisbursementDate      time.Time          `json:"disbursement_date" bson:"disbursement_date"`
	ProcessingFee         float64            `json:"processing_fee" bson:"processing_fee"`
	ForeclosureCharges    float64            `json:"foreclosure_charges" bson:"foreclosure_charges"` // percentage
	RateChangeHistory     []RateChangeEntry  `json:"rate_change_history" bson:"rate_change_history"`
	EMIsPaid              []EMIEntry         `json:"emis_paid" bson:"emis_paid"`
	Prepayments           []Prepayment       `json:"prepayments" bson:"prepayments"`
	OutstandingPrincipal  float64            `json:"outstanding_principal" bson:"outstanding_principal"`
	TotalPrincipalPaid    float64            `json:"total_principal_paid" bson:"total_principal_paid"`
	TotalInterestPaid     float64            `json:"total_interest_paid" bson:"total_interest_paid"`
	TotalAmountPaid       float64            `json:"total_amount_paid" bson:"total_amount_paid"`
	TotalPrepayments      float64            `json:"total_prepayments" bson:"total_prepayments"`
	RemainingTenureMonths int                `json:"remaining_tenure_months" bson:"remaining_tenure_months"`
	LoanEndDate           time.Time          `json:"loan_end_date" bson:"loan_end_date"`
	Status                LoanStatus         `json:"status" bson:"status"`
	Notes                 string             `json:"notes" bson:"notes"`
	CreatedAt             time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreatePersonalLoanRequest struct {
	LenderName         string       `json:"lender_name" binding:"required"`
	LoanAccountNumber  string       `json:"loan_account_number"`
	LoanPurpose        string       `json:"loan_purpose"`
	PrincipalAmount    float64      `json:"principal_amount" binding:"required"`
	DisbursedAmount    float64      `json:"disbursed_amount" binding:"required"`
	InterestRate       float64      `json:"interest_rate" binding:"required"`
	RateType           LoanRateType `json:"rate_type" binding:"required"`
	TenureMonths       int          `json:"tenure_months" binding:"required"`
	EMIStartDate       string       `json:"emi_start_date" binding:"required"`
	DisbursementDate   string       `json:"disbursement_date" binding:"required"`
	ProcessingFee      float64      `json:"processing_fee"`
	ForeclosureCharges float64      `json:"foreclosure_charges"`
	Notes              string       `json:"notes"`
}

type UpdatePersonalLoanRequest struct {
	LenderName        string `json:"lender_name"`
	LoanAccountNumber string `json:"loan_account_number"`
	LoanPurpose       string `json:"loan_purpose"`
	Notes             string `json:"notes"`
}
