package models

type ProjectionPoint struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Value float64 `json:"value"`
	P10   float64 `json:"p10"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
}

type InvestmentProjection struct {
	Name           string            `json:"name"`
	Category       string            `json:"category"` // mutual_fund, fixed_deposit, provident_fund, stock
	CurrentValue   float64           `json:"current_value"`
	AssumedRatePct float64           `json:"assumed_rate_pct"`
	VolatilityPct  float64           `json:"volatility_pct"`
	Projections    []ProjectionPoint `json:"projections"`
}

type ProjectedRange struct {
	P10 float64 `json:"p10"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
}

type ProjectionsResponse struct {
	Investments      []InvestmentProjection `json:"investments"`
	AggregateMonthly []ProjectionPoint      `json:"aggregate_monthly"`
	TotalCurrent     float64                `json:"total_current"`
	TotalProjected1Y float64                `json:"total_projected_1y"`
	TotalProjected3Y float64                `json:"total_projected_3y"`
	TotalProjected5Y float64                `json:"total_projected_5y"`
	Projected1YRange *ProjectedRange        `json:"projected_1y_range,omitempty"`
	Projected3YRange *ProjectedRange        `json:"projected_3y_range,omitempty"`
	Projected5YRange *ProjectedRange        `json:"projected_5y_range,omitempty"`
	Scenario         string                 `json:"scenario"`
}
