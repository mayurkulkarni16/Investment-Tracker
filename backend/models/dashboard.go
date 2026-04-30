package models

type DashboardSummary struct {
	TotalInvested        float64            `json:"total_invested"`
	CurrentValue         float64            `json:"current_value"`
	TotalGains           float64            `json:"total_gains"`
	OverallReturnPct     float64            `json:"overall_return_percent"`
	ELSSTaxSaving        float64            `json:"elss_tax_saving"`
	AssetAllocation      map[string]float64 `json:"asset_allocation"`
	UpcomingPayouts      []UpcomingPayout   `json:"upcoming_payouts"`
	MutualFundSummary    InvestmentSummary  `json:"mutual_fund_summary"`
	CorporateBondSummary InvestmentSummary  `json:"corporate_bond_summary"`
	FixedDepositSummary  InvestmentSummary  `json:"fixed_deposit_summary"`
	ProvidentFundSummary InvestmentSummary  `json:"provident_fund_summary"`
	DataSources          map[string]string  `json:"data_sources,omitempty"`
}

type InvestmentSummary struct {
	TotalInvested float64 `json:"total_invested"`
	CurrentValue  float64 `json:"current_value"`
	Count         int     `json:"count"`
}

type UpcomingPayout struct {
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	Date       string  `json:"date"`
	Amount     float64 `json:"amount"`
	PayoutType string  `json:"payout_type"`
}
