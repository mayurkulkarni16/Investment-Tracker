package models

// FIREDefaults are values derived from the user's data to prefill the planner.
type FIREDefaults struct {
	MonthlyExpenses       float64 `json:"monthly_expenses"`
	MonthlyExpensesSource string  `json:"monthly_expenses_source"` // ledger, none
	MonthlySavings        float64 `json:"monthly_savings"`
	MonthlySavingsSource  string  `json:"monthly_savings_source"` // ledger, sips, none
	CurrentCorpus         float64 `json:"current_corpus"`         // financial assets (excl. real estate, home equity)
	PreRetirementReturn   float64 `json:"pre_retirement_return"`  // %
	PostRetirementReturn  float64 `json:"post_retirement_return"`
	Inflation             float64 `json:"inflation"`
}

type FIREPlanRequest struct {
	CurrentAge           int     `json:"current_age" binding:"required,min=15,max=90"`
	RetirementAge        int     `json:"retirement_age" binding:"required,min=20,max=95"`
	LifeExpectancy       int     `json:"life_expectancy" binding:"required,min=40,max=110"`
	MonthlyExpenses      float64 `json:"monthly_expenses" binding:"required,gt=0"` // today's rupees
	CurrentCorpus        float64 `json:"current_corpus" binding:"min=0"`
	MonthlySavings       float64 `json:"monthly_savings" binding:"min=0"`
	AnnualStepUpPct      float64 `json:"annual_step_up_pct" binding:"min=0,max=50"`
	PreRetirementReturn  float64 `json:"pre_retirement_return" binding:"min=0,max=30"`
	PostRetirementReturn float64 `json:"post_retirement_return" binding:"min=0,max=30"`
	Inflation            float64 `json:"inflation" binding:"min=0,max=20"`
	RetirementExpensePct float64 `json:"retirement_expense_pct" binding:"min=0,max=200"` // % of today's expenses needed after retirement (default 100)
}

type FIREYear struct {
	Age            int     `json:"age"`
	Year           int     `json:"year"`
	Phase          string  `json:"phase"` // accumulation, retirement
	Contribution   float64 `json:"contribution"`
	Withdrawal     float64 `json:"withdrawal"`
	EndCorpus      float64 `json:"end_corpus"`
	RequiredCorpus float64 `json:"required_corpus"` // corpus needed at this age to retire now
}

type FIREPlan struct {
	Input                  FIREPlanRequest `json:"input"`
	YearsToRetirement      int             `json:"years_to_retirement"`
	AnnualExpenseAtRetire  float64         `json:"annual_expense_at_retirement"`
	TargetCorpus           float64         `json:"target_corpus"` // annuity-based, lasts to life expectancy
	SWRCorpus              float64         `json:"swr_corpus"`    // 25x rule (4% withdrawal) for reference
	ProjectedCorpus        float64         `json:"projected_corpus"`
	Gap                    float64         `json:"gap"` // target - projected (positive = shortfall)
	OnTrack                bool            `json:"on_track"`
	RequiredMonthlySavings float64         `json:"required_monthly_savings"` // with the same step-up
	EarliestRetirementAge  int             `json:"earliest_retirement_age"`  // 0 if not reachable before life expectancy
	CorpusDepletesAtAge    int             `json:"corpus_depletes_at_age"`   // 0 if it lasts
	SuccessProbabilityPct  float64         `json:"success_probability_pct"`  // Monte Carlo
	FINumber               float64         `json:"fi_number"`                // corpus needed if retiring today
	ProgressPct            float64         `json:"progress_pct"`             // current corpus / FI number
	Years                  []FIREYear      `json:"years"`
}

// FinanceOverview is a compact household-finance snapshot for the dashboard and mobile home screen.
type FinanceOverview struct {
	NetWorth            float64           `json:"net_worth"`
	TotalAssets         float64           `json:"total_assets"`
	TotalLiabilities    float64           `json:"total_liabilities"`
	NetWorthChangeMonth *float64          `json:"net_worth_change_month,omitempty"` // vs last month's snapshot
	BankBalance         float64           `json:"bank_balance"`
	CashInHand          float64           `json:"cash_in_hand"`
	LiquidBalance       float64           `json:"liquid_balance"` // bank + cash + wallets
	EmergencyFund       float64           `json:"emergency_fund"` // accounts flagged as emergency fund (or all liquid if none flagged)
	EmergencyMonths     float64           `json:"emergency_months"`
	StaleBalances       int               `json:"stale_balances"` // accounts not updated in 30 days
	MonthIncome         float64           `json:"month_income"`
	MonthExpense        float64           `json:"month_expense"`
	MonthSavingsRatePct float64           `json:"month_savings_rate_pct"`
	AvgMonthlyExpense   float64           `json:"avg_monthly_expense"`
	AvgSavingsRatePct   float64           `json:"avg_savings_rate_pct"`
	PendingReview       int               `json:"pending_review"`
	BudgetsOver         int               `json:"budgets_over"`
	BudgetsWarning      int               `json:"budgets_warning"`
	Budgets             []BudgetStatus    `json:"budgets"`
	GoldValue           float64           `json:"gold_value"`
	RealEstateValue     float64           `json:"real_estate_value"`
	LifeCover           float64           `json:"life_cover"`
	HealthCover         float64           `json:"health_cover"`
	UpcomingPremiums    []UpcomingPremium `json:"upcoming_premiums"`
	FINumber            float64           `json:"fi_number"` // 25x annual expenses
	FIProgressPct       float64           `json:"fi_progress_pct"`
}

// BrokerImportResult reports what a broker CSV import did (or would do on a dry run).
type BrokerImportResult struct {
	Broker        string            `json:"broker"`
	Kind          string            `json:"kind"` // holdings, tradebook
	DryRun        bool              `json:"dry_run"`
	Rows          int               `json:"rows"`
	StocksCreated int               `json:"stocks_created"`
	TradesAdded   int               `json:"trades_added"`
	Skipped       int               `json:"skipped"`
	Items         []BrokerImportRow `json:"items"`
	Errors        []string          `json:"errors,omitempty"`
}

type BrokerImportRow struct {
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Action   string  `json:"action"`         // create, add_trade, skip_duplicate, skip_exists, conflict, error
	Type     string  `json:"type,omitempty"` // buy, sell
	Date     string  `json:"date,omitempty"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Message  string  `json:"message,omitempty"`
}
