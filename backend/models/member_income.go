package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Payslip is one month's salary. Projected payslips are expectations for
// months not yet paid; actual payslips replace them as they arrive.
type Payslip struct {
	Month     string `json:"month" bson:"month"` // YYYY-MM, within the FY
	Projected bool   `json:"projected" bson:"projected"`
	Employer  string `json:"employer" bson:"employer,omitempty"`

	// Fixed pay
	Basic            float64 `json:"basic" bson:"basic"`
	DA               float64 `json:"da" bson:"da"`
	HRA              float64 `json:"hra" bson:"hra"`
	SpecialAllowance float64 `json:"special_allowance" bson:"special_allowance"`
	LTA              float64 `json:"lta" bson:"lta"`
	OtherAllowances  float64 `json:"other_allowances" bson:"other_allowances"` // taxable allowances
	Perquisites      float64 `json:"perquisites" bson:"perquisites"`

	// Variable pay
	Bonus          float64 `json:"bonus" bson:"bonus"`           // performance / joining / retention
	Incentives     float64 `json:"incentives" bson:"incentives"` // sales incentives, commission
	ESOPPerquisite float64 `json:"esop_perquisite" bson:"esop_perquisite"`
	OtherVariable  float64 `json:"other_variable" bson:"other_variable"`

	// Contributions & deductions on the payslip
	EmployeePF      float64 `json:"employee_pf" bson:"employee_pf"` // EPF + VPF (80C)
	EmployerNPS     float64 `json:"employer_nps" bson:"employer_nps"`
	ProfessionalTax float64 `json:"professional_tax" bson:"professional_tax"`
	TDS             float64 `json:"tds" bson:"tds"`

	RentPaid float64 `json:"rent_paid" bson:"rent_paid"` // rent paid by the employee this month (HRA / 80GG)
}

func (p *Payslip) Fixed() float64 {
	return p.Basic + p.DA + p.HRA + p.SpecialAllowance + p.LTA + p.OtherAllowances + p.Perquisites
}

func (p *Payslip) Variable() float64 {
	return p.Bonus + p.Incentives + p.ESOPPerquisite + p.OtherVariable
}

// Gross is taxable gross salary for the month, including employer NPS
// (a perquisite that is then deducted under 80CCD(2)).
func (p *Payslip) Gross() float64 { return p.Fixed() + p.Variable() + p.EmployerNPS }

// OtherIncome covers non-salary income not tracked by investment modules.
type OtherIncome struct {
	Pension          float64 `json:"pension" bson:"pension"`               // uncommuted pension (annual)
	FamilyPension    float64 `json:"family_pension" bson:"family_pension"` // annual
	BusinessScheme   string  `json:"business_scheme" bson:"business_scheme,omitempty"`
	BusinessReceipts float64 `json:"business_receipts" bson:"business_receipts"` // gross receipts / turnover
	BusinessProfit   float64 `json:"business_profit" bson:"business_profit"`     // net profit when scheme = actual
	SavingsInterest  float64 `json:"savings_interest" bson:"savings_interest"`
	OtherInterest    float64 `json:"other_interest" bson:"other_interest"` // deposits not tracked in the app
	Dividends        float64 `json:"dividends" bson:"dividends"`           // not tracked in the app
	RentalIncome     float64 `json:"rental_income" bson:"rental_income"`   // annual rent, properties not tracked in the app
	MunicipalTax     float64 `json:"municipal_tax" bson:"municipal_tax"`
	OtherIncome      float64 `json:"other_income" bson:"other_income"` // freelance (non-business), gifts > ₹50k, etc.
	STCGEquity       float64 `json:"stcg_equity" bson:"stcg_equity"`   // gains outside the app
	LTCGEquity       float64 `json:"ltcg_equity" bson:"ltcg_equity"`
	OtherGainsSlab   float64 `json:"other_gains_slab" bson:"other_gains_slab"`
}

// ManualDeductions are claims not derived from tracked investments.
type ManualDeductions struct {
	Sec80COther      float64 `json:"sec_80c_other" bson:"sec_80c_other"` // tuition fees, stamp duty, SSY/NSC/LIC not tracked…
	Sec80DSelf       float64 `json:"sec_80d_self" bson:"sec_80d_self"`   // policies not tracked + preventive check-up
	Sec80DParents    float64 `json:"sec_80d_parents" bson:"sec_80d_parents"`
	ParentsSenior    bool    `json:"parents_senior" bson:"parents_senior"`
	Sec80E           float64 `json:"sec_80e" bson:"sec_80e"`
	Sec80G           float64 `json:"sec_80g" bson:"sec_80g"`
	NPSOther         float64 `json:"nps_other" bson:"nps_other"` // own Tier-1 NPS not tracked
	LTAExempt        float64 `json:"lta_exempt" bson:"lta_exempt"`
	HomeLoanInterest float64 `json:"home_loan_interest" bson:"home_loan_interest"` // self-occupied, loans not tracked
	OtherTDS         float64 `json:"other_tds" bson:"other_tds"`                   // TDS on FD, rent etc. (Form 26AS)
	AdvanceTax       float64 `json:"advance_tax" bson:"advance_tax"`               // advance + self-assessment tax paid
}

// MemberIncome is one family member's income for one financial year.
type MemberIncome struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"user_id" bson:"user_id"`
	ProfileID string             `json:"profile_id" bson:"profile_id"`
	FY        string             `json:"fy" bson:"fy"` // 2025-26

	Payslips []Payslip `json:"payslips" bson:"payslips"`
	// ProjectRemaining fills months without a payslip with the latest
	// payslip's fixed pay (no variable pay) to estimate the full year.
	ProjectRemaining bool             `json:"project_remaining" bson:"project_remaining"`
	Other            OtherIncome      `json:"other" bson:"other"`
	Deductions       ManualDeductions `json:"deductions" bson:"deductions"`
	Notes            string           `json:"notes" bson:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type MemberIncomeRequest struct {
	Payslips         []Payslip        `json:"payslips"`
	ProjectRemaining bool             `json:"project_remaining"`
	Other            OtherIncome      `json:"other"`
	Deductions       ManualDeductions `json:"deductions"`
	Notes            string           `json:"notes"`
}
