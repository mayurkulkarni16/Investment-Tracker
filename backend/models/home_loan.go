package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LoanRateType string

const (
	RateFixed    LoanRateType = "fixed"
	RateFloating LoanRateType = "floating"
)

type LoanStatus string

const (
	LoanActive     LoanStatus = "active"
	LoanClosed     LoanStatus = "closed"
	LoanForeclosed LoanStatus = "foreclosed"
)

type PrepaymentType string

const (
	PrepaymentPartPayment PrepaymentType = "part_payment"
	PrepaymentForeclosure PrepaymentType = "foreclosure"
)

type RateChangeEntry struct {
	EffectiveDate time.Time `json:"effective_date" bson:"effective_date"`
	OldRate       float64   `json:"old_rate" bson:"old_rate"`
	NewRate       float64   `json:"new_rate" bson:"new_rate"`
	NewEMI        float64   `json:"new_emi" bson:"new_emi"`
}

type DisbursementEntry struct {
	DisbursementID string    `json:"disbursement_id" bson:"disbursement_id"`
	Date           time.Time `json:"date" bson:"date"`
	Amount         float64   `json:"amount" bson:"amount"`
	Tranche        int       `json:"tranche" bson:"tranche"`               // 1, 2, 3...
	PreEMIAmount   float64   `json:"pre_emi_amount" bson:"pre_emi_amount"` // pro-rata interest for remaining days
	PreEMIDays     int       `json:"pre_emi_days" bson:"pre_emi_days"`
	DaysInMonth    int       `json:"days_in_month" bson:"days_in_month"`
	Notes          string    `json:"notes" bson:"notes"`
}

type PreEMIEntry struct {
	PreEMIID        string     `json:"pre_emi_id" bson:"pre_emi_id"`
	Month           string     `json:"month" bson:"month"` // "2026-04"
	DisbursedAtTime float64    `json:"disbursed_at_time" bson:"disbursed_at_time"`
	TrancheAmount   float64    `json:"tranche_amount" bson:"tranche_amount"` // amount of the new tranche
	InterestRate    float64    `json:"interest_rate" bson:"interest_rate"`
	InterestAmount  float64    `json:"interest_amount" bson:"interest_amount"`
	DaysCharged     int        `json:"days_charged" bson:"days_charged"`
	DaysInMonth     int        `json:"days_in_month" bson:"days_in_month"`
	Paid            bool       `json:"paid" bson:"paid"`
	PaidDate        *time.Time `json:"paid_date,omitempty" bson:"paid_date,omitempty"`
}

type EMIEntry struct {
	EMIID            string     `json:"emi_id" bson:"emi_id"`
	Month            string     `json:"month" bson:"month"` // "2026-04"
	DueDate          time.Time  `json:"due_date" bson:"due_date"`
	EMIAmount        float64    `json:"emi_amount" bson:"emi_amount"`
	PrincipalPortion float64    `json:"principal_portion" bson:"principal_portion"`
	InterestPortion  float64    `json:"interest_portion" bson:"interest_portion"`
	OutstandingAfter float64    `json:"outstanding_after" bson:"outstanding_after"`
	Paid             bool       `json:"paid" bson:"paid"`
	PaidDate         *time.Time `json:"paid_date,omitempty" bson:"paid_date,omitempty"`
}

type Prepayment struct {
	PrepaymentID string         `json:"prepayment_id" bson:"prepayment_id"`
	Date         time.Time      `json:"date" bson:"date"`
	Amount       float64        `json:"amount" bson:"amount"`
	Type         PrepaymentType `json:"type" bson:"type"`
	NewEMI       float64        `json:"new_emi,omitempty" bson:"new_emi,omitempty"`
	NewTenure    int            `json:"new_tenure,omitempty" bson:"new_tenure,omitempty"`
	Notes        string         `json:"notes" bson:"notes"`
}

type HomeLoan struct {
	ID                    primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID                string              `json:"user_id" bson:"user_id"`
	BankName              string              `json:"bank_name" bson:"bank_name"`
	LoanAccountNumber     string              `json:"loan_account_number" bson:"loan_account_number"`
	PropertyAddress       string              `json:"property_address" bson:"property_address"`
	LoanPurpose           string              `json:"loan_purpose" bson:"loan_purpose"` // purchase, construction, renovation
	SanctionedAmount      float64             `json:"sanctioned_amount" bson:"sanctioned_amount"`
	DisbursedAmount       float64             `json:"disbursed_amount" bson:"disbursed_amount"`
	InterestRate          float64             `json:"interest_rate" bson:"interest_rate"`
	RateType              LoanRateType        `json:"rate_type" bson:"rate_type"`
	TenureMonths          int                 `json:"tenure_months" bson:"tenure_months"`
	EMIAmount             float64             `json:"emi_amount" bson:"emi_amount"`
	EMIStartDate          time.Time           `json:"emi_start_date" bson:"emi_start_date"`
	DisbursementDate      time.Time           `json:"disbursement_date" bson:"disbursement_date"`
	CoBorrower            string              `json:"co_borrower" bson:"co_borrower"`
	IsUnderConstruction   bool                `json:"is_under_construction" bson:"is_under_construction"`
	FullEMIStarted        bool                `json:"full_emi_started" bson:"full_emi_started"`
	Disbursements         []DisbursementEntry `json:"disbursements" bson:"disbursements"`
	PreEMIsPaid           []PreEMIEntry       `json:"pre_emis_paid" bson:"pre_emis_paid"`
	TotalPreEMIPaid       float64             `json:"total_pre_emi_paid" bson:"total_pre_emi_paid"`
	RateChangeHistory     []RateChangeEntry   `json:"rate_change_history" bson:"rate_change_history"`
	EMIsPaid              []EMIEntry          `json:"emis_paid" bson:"emis_paid"`
	Prepayments           []Prepayment        `json:"prepayments" bson:"prepayments"`
	OutstandingPrincipal  float64             `json:"outstanding_principal" bson:"outstanding_principal"`
	TotalPrincipalPaid    float64             `json:"total_principal_paid" bson:"total_principal_paid"`
	TotalInterestPaid     float64             `json:"total_interest_paid" bson:"total_interest_paid"`
	TotalAmountPaid       float64             `json:"total_amount_paid" bson:"total_amount_paid"`
	TotalPrepayments      float64             `json:"total_prepayments" bson:"total_prepayments"`
	RemainingTenureMonths int                 `json:"remaining_tenure_months" bson:"remaining_tenure_months"`
	LoanEndDate           time.Time           `json:"loan_end_date" bson:"loan_end_date"`
	// Tax benefits (Section 24b interest deduction, 80C principal deduction)
	InterestPaidThisFY  float64    `json:"interest_paid_this_fy" bson:"-"`
	PrincipalPaidThisFY float64    `json:"principal_paid_this_fy" bson:"-"`
	PreEMIPaidThisFY    float64    `json:"pre_emi_paid_this_fy" bson:"-"`
	Status              LoanStatus `json:"status" bson:"status"`
	Notes               string     `json:"notes" bson:"notes"`
	CreatedAt           time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" bson:"updated_at"`
}

type CreateHomeLoanRequest struct {
	BankName            string       `json:"bank_name" binding:"required"`
	LoanAccountNumber   string       `json:"loan_account_number"`
	PropertyAddress     string       `json:"property_address"`
	LoanPurpose         string       `json:"loan_purpose"`
	SanctionedAmount    float64      `json:"sanctioned_amount" binding:"required"`
	DisbursedAmount     float64      `json:"disbursed_amount" binding:"required"`
	InterestRate        float64      `json:"interest_rate" binding:"required"`
	RateType            LoanRateType `json:"rate_type" binding:"required"`
	TenureMonths        int          `json:"tenure_months" binding:"required"`
	EMIStartDate        string       `json:"emi_start_date" binding:"required"`
	DisbursementDate    string       `json:"disbursement_date" binding:"required"`
	IsUnderConstruction bool         `json:"is_under_construction"`
	CoBorrower          string       `json:"co_borrower"`
	Notes               string       `json:"notes"`
}

type UpdateHomeLoanRequest struct {
	BankName          string `json:"bank_name"`
	LoanAccountNumber string `json:"loan_account_number"`
	PropertyAddress   string `json:"property_address"`
	LoanPurpose       string `json:"loan_purpose"`
	CoBorrower        string `json:"co_borrower"`
	Notes             string `json:"notes"`
}

type AddEMIPaymentRequest struct {
	Month    string `json:"month" binding:"required"` // "2026-04"
	PaidDate string `json:"paid_date" binding:"required"`
}

type AddPrepaymentRequest struct {
	Date   string         `json:"date" binding:"required"`
	Amount float64        `json:"amount" binding:"required"`
	Type   PrepaymentType `json:"type" binding:"required"`
	Notes  string         `json:"notes"`
}

type ChangeRateRequest struct {
	EffectiveDate string  `json:"effective_date" binding:"required"`
	NewRate       float64 `json:"new_rate" binding:"required"`
}

type AddDisbursementRequest struct {
	Date   string  `json:"date" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
	Notes  string  `json:"notes"`
}

type MarkConstructionCompleteRequest struct {
	CompletionDate string `json:"completion_date"` // optional
}

type AmortizationEntry struct {
	Month            int     `json:"month"`
	EMI              float64 `json:"emi"`
	PrincipalPortion float64 `json:"principal_portion"`
	InterestPortion  float64 `json:"interest_portion"`
	OutstandingAfter float64 `json:"outstanding_after"`
}
