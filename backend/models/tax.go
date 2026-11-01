package models

// Tax Summary types (computed, no collection)

type Section80CBreakdown struct {
	EPFContribution   float64 `json:"epf_contribution"`
	PPFContribution   float64 `json:"ppf_contribution"` // if tracked
	ELSSInvestment    float64 `json:"elss_investment"`
	HomeLoanPrincipal float64 `json:"home_loan_principal"`
	LifeInsurance     float64 `json:"life_insurance"` // premiums of life/term/ULIP policies due this FY
	Total             float64 `json:"total"`
	Limit             float64 `json:"limit"` // 150000
	Deduction         float64 `json:"deduction"`
}

// Section80DBreakdown covers health insurance premiums (self, spouse, children
// and, separately, parents).
type Section80DBreakdown struct {
	HealthInsurance float64 `json:"health_insurance"`
	Parents         float64 `json:"parents"`
	Limit           float64 `json:"limit"` // 25000 (50000 if a senior citizen)
	ParentsLimit    float64 `json:"parents_limit"`
	Deduction       float64 `json:"deduction"`
}

type Section80CCDBreakdown struct {
	NPS80CCD1  float64 `json:"nps_80ccd1"`  // within 80C limit
	NPS80CCD1B float64 `json:"nps_80ccd1b"` // additional 50K
	NPS80CCD2  float64 `json:"nps_80ccd2"`  // employer (14% of basic)
}

type Section24bBreakdown struct {
	HomeLoanInterest float64 `json:"home_loan_interest"`
	PreEMIInterest   float64 `json:"pre_emi_interest"`
	Total            float64 `json:"total"`
	Limit            float64 `json:"limit"` // 200000 for self-occupied
	Deduction        float64 `json:"deduction"`
}

type CapitalGainEntry struct {
	InvestmentType string  `json:"investment_type"` // mutual_fund, stock
	InvestmentID   string  `json:"investment_id"`
	InvestmentName string  `json:"investment_name"`
	BuyDate        string  `json:"buy_date"`
	SellDate       string  `json:"sell_date"`
	BuyAmount      float64 `json:"buy_amount"`
	SellAmount     float64 `json:"sell_amount"`
	Gain           float64 `json:"gain"`
	HoldingDays    int     `json:"holding_days"`
	IsLongTerm     bool    `json:"is_long_term"`
	TaxRate        float64 `json:"tax_rate"` // percentage
	TaxLiability   float64 `json:"tax_liability"`
	Grandfathered  bool    `json:"grandfathered,omitempty"` // pre-31-Jan-2018 equity
	IsEquity       bool    `json:"is_equity"`               // false = taxed at slab (debt MF)
}

type CapitalGainsSummary struct {
	STCG           float64            `json:"stcg"`       // equity (111A)
	LTCG           float64            `json:"ltcg"`       // equity (112A)
	SlabGains      float64            `json:"slab_gains"` // debt MF etc., taxed at the member's slab rate
	STCGTax        float64            `json:"stcg_tax"`
	LTCGTax        float64            `json:"ltcg_tax"`
	TotalTax       float64            `json:"total_tax"`
	Entries        []CapitalGainEntry `json:"entries"`
	LTCGExemption  float64            `json:"ltcg_exemption"`
	HarvestingTips []string           `json:"harvesting_tips,omitempty"`
}

type TaxSummary struct {
	FY                string                `json:"fy"` // 2025-26
	ProfileID         string                `json:"profile_id,omitempty"`
	Section80C        Section80CBreakdown   `json:"section_80c"`
	Section80CCD      Section80CCDBreakdown `json:"section_80ccd"`
	Section80D        Section80DBreakdown   `json:"section_80d"`
	Section24b        Section24bBreakdown   `json:"section_24b"`
	CapitalGains      CapitalGainsSummary   `json:"capital_gains"`
	InterestIncome    float64               `json:"interest_income"` // FD + bonds
	DividendIncome    float64               `json:"dividend_income"`
	RentalIncome      float64               `json:"rental_income"`       // let-out property (gross)
	OtherRentalIncome float64               `json:"other_rental_income"` // vehicle leasing payouts
	TotalDeductions   float64               `json:"total_deductions"`
	TotalTaxableGains float64               `json:"total_taxable_gains"`
}
