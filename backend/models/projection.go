package models

type ProjectionPoint struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Value float64 `json:"value"`
	P10   float64 `json:"p10"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
}

// Projection bases: "simulated" holdings are market-linked and use Monte Carlo;
// "scheduled" holdings follow contractual cashflows (FD, PF, bonds, rentals).
const (
	ProjectionSimulated = "simulated"
	ProjectionScheduled = "scheduled"
)

type InvestmentProjection struct {
	Name                string            `json:"name"`
	Category            string            `json:"category"` // mutual_fund, stock, nps, fixed_deposit, provident_fund, corporate_bond, vehicle_rental
	Basis               string            `json:"basis"`
	CurrentValue        float64           `json:"current_value"`
	AssumedRatePct      float64           `json:"assumed_rate_pct"`
	VolatilityPct       float64           `json:"volatility_pct"`
	MonthlyContribution float64           `json:"monthly_contribution"`
	Note                string            `json:"note,omitempty"`
	Projections         []ProjectionPoint `json:"projections"`
}

type LiabilityProjection struct {
	Name         string            `json:"name"`
	Category     string            `json:"category"` // home_loan, personal_loan
	CurrentValue float64           `json:"current_value"`
	InterestRate float64           `json:"interest_rate"`
	MonthlyEMI   float64           `json:"monthly_emi"`
	Projections  []ProjectionPoint `json:"projections"`
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
	// Values at the end of the requested horizon.
	TotalProjectedEnd float64         `json:"total_projected_end"`
	ProjectedEndRange *ProjectedRange `json:"projected_end_range,omitempty"`
	Scenario          string          `json:"scenario"`

	// Liabilities are amortised deterministically; net worth = assets - liabilities.
	Liabilities        []LiabilityProjection `json:"liabilities"`
	LiabilitiesMonthly []ProjectionPoint     `json:"liabilities_monthly"`
	NetWorthMonthly    []ProjectionPoint     `json:"net_worth_monthly"`
	TotalLiabilities   float64               `json:"total_liabilities"`
	CurrentNetWorth    float64               `json:"current_net_worth"`

	// InflationRate > 0 means every value is expressed in today's money.
	InflationRate float64 `json:"inflation_rate"`
	Simulations   int     `json:"simulations"`
}
