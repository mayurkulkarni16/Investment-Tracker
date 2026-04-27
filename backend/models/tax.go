package models

// Tax Summary types (computed, no collection)

type Section80CBreakdown struct {
	EPFContribution   float64 `json:"epf_contribution"`
	PPFContribution   float64 `json:"ppf_contribution"` // if tracked
	ELSSInvestment    float64 `json:"elss_investment"`
	HomeLoanPrincipal float64 `json:"home_loan_principal"`
	LifeInsurance     float64 `json:"life_insurance"` // placeholder
	Total             float64 `json:"total"`
	Limit             float64 `json:"limit"` // 150000
	Deduction         float64 `json:"deduction"`
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
}

type CapitalGainsSummary struct {
	STCG     float64            `json:"stcg"`
	LTCG     float64            `json:"ltcg"`
	STCGTax  float64            `json:"stcg_tax"`
	LTCGTax  float64            `json:"ltcg_tax"`
	TotalTax float64            `json:"total_tax"`
	Entries  []CapitalGainEntry `json:"entries"`
	// LTCG exemption for equity MF/stocks: first 1.25L is exempt
	LTCGExemption float64 `json:"ltcg_exemption"`
}

type TaxSummary struct {
	FY                string                `json:"fy"` // 2025-26
	Section80C        Section80CBreakdown   `json:"section_80c"`
	Section80CCD      Section80CCDBreakdown `json:"section_80ccd"`
	Section24b        Section24bBreakdown   `json:"section_24b"`
	CapitalGains      CapitalGainsSummary   `json:"capital_gains"`
	InterestIncome    float64               `json:"interest_income"` // FD + bonds
	DividendIncome    float64               `json:"dividend_income"`
	TotalDeductions   float64               `json:"total_deductions"`
	TotalTaxableGains float64               `json:"total_taxable_gains"`
}
