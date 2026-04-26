package models

type ProjectionPoint struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Value float64 `json:"value"`
}

type InvestmentProjection struct {
	Name           string            `json:"name"`
	Category       string            `json:"category"` // mutual_fund, fixed_deposit, provident_fund, stock
	CurrentValue   float64           `json:"current_value"`
	AssumedRatePct float64           `json:"assumed_rate_pct"`
	Projections    []ProjectionPoint `json:"projections"`
}

type ProjectionsResponse struct {
	Investments      []InvestmentProjection `json:"investments"`
	AggregateMonthly []ProjectionPoint      `json:"aggregate_monthly"`
	TotalCurrent     float64                `json:"total_current"`
	TotalProjected1Y float64                `json:"total_projected_1y"`
	TotalProjected3Y float64                `json:"total_projected_3y"`
	TotalProjected5Y float64                `json:"total_projected_5y"`
}
