package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PFAccountType string

const (
	PFTypeEPF PFAccountType = "EPF"
	PFTypeVPF PFAccountType = "VPF"
	PFTypePPF PFAccountType = "PPF"
)

type MonthlyContribution struct {
	Month                string  `json:"month" bson:"month"`
	EmployeeContribution float64 `json:"employee_contribution" bson:"employee_contribution"`
	EmployerContribution float64 `json:"employer_contribution" bson:"employer_contribution"`
	Total                float64 `json:"total" bson:"total"`
}

type FinancialYearEntry struct {
	FinancialYear        string                `json:"financial_year" bson:"financial_year"`
	MonthlyContributions []MonthlyContribution `json:"monthly_contributions" bson:"monthly_contributions"`
	OpeningBalance       float64               `json:"opening_balance" bson:"opening_balance"`
	InterestEarned       float64               `json:"interest_earned" bson:"interest_earned"`
	ClosingBalance       float64               `json:"closing_balance" bson:"closing_balance"`
}

type ProvidentFund struct {
	ID                        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID                    string               `json:"user_id" bson:"user_id"`
	AccountType               PFAccountType        `json:"account_type" bson:"account_type"`
	AccountNumber             string               `json:"account_number" bson:"account_number"`
	EmployerName              string               `json:"employer_name" bson:"employer_name"`
	InterestRate              float64              `json:"interest_rate" bson:"interest_rate"`
	FinancialYearEntries      []FinancialYearEntry `json:"financial_year_entries" bson:"financial_year_entries"`
	CurrentBalance            float64              `json:"current_balance" bson:"current_balance"`
	TotalEmployeeContribution float64              `json:"total_employee_contribution" bson:"total_employee_contribution"`
	TotalEmployerContribution float64              `json:"total_employer_contribution" bson:"total_employer_contribution"`
	TotalInterestEarned       float64              `json:"total_interest_earned" bson:"total_interest_earned"`
	XIRR                      float64              `json:"xirr" bson:"-"`
	Notes                     string               `json:"notes" bson:"notes"`
	CreatedAt                 time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt                 time.Time            `json:"updated_at" bson:"updated_at"`
}

type CreateProvidentFundRequest struct {
	AccountType   PFAccountType `json:"account_type" binding:"required"`
	AccountNumber string        `json:"account_number" binding:"required"`
	EmployerName  string        `json:"employer_name"`
	InterestRate  float64       `json:"interest_rate" binding:"required"`
	Notes         string        `json:"notes"`
}

type AddMonthlyContributionRequest struct {
	FinancialYear        string  `json:"financial_year" binding:"required"`
	Month                string  `json:"month" binding:"required"`
	EmployeeContribution float64 `json:"employee_contribution" binding:"required"`
	EmployerContribution float64 `json:"employer_contribution"`
}

type ImportPFContribution struct {
	Month                string  `json:"month"`
	EmployeeContribution float64 `json:"employee_contribution"`
	EmployerContribution float64 `json:"employer_contribution"`
}

type ImportPFYear struct {
	FinancialYear  string                 `json:"financial_year"`
	OpeningBalance float64                `json:"opening_balance"`
	Contributions  []ImportPFContribution `json:"contributions"`
	InterestEarned float64                `json:"interest_earned"`
	ClosingBalance float64                `json:"closing_balance"`
}

type ImportPFRequest struct {
	AccountNumber string         `json:"account_number"`
	EmployerName  string         `json:"employer_name"`
	Years         []ImportPFYear `json:"years"`
}

type ImportPFResult struct {
	YearsAdded         int      `json:"years_added"`
	ContributionsAdded int      `json:"contributions_added"`
	Errors             []string `json:"errors,omitempty"`
}
