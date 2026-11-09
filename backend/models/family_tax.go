package models

import (
	"time"

	"investment-tracker/taxengine"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tax line kinds derived from tracked assets.
const (
	LineInterest        = "interest"     // FD / bond interest (slab)
	LineDividend        = "dividend"     // slab
	LineSTCGEquity      = "stcg_equity"  // 111A
	LineLTCGEquity      = "ltcg_equity"  // 112A
	LineGainSlab        = "gain_slab"    // debt MF etc. at slab
	LineRent            = "rent"         // house property (gross annual rent)
	LineOtherRental     = "other_rental" // alternate investment payouts, e.g. EV leasing (other sources)
	Line80CEPF          = "80c_epf"      // EPF / VPF employee share
	Line80CPPF          = "80c_ppf"      // PPF deposits
	Line80CELSS         = "80c_elss"     // ELSS purchases
	Line80CPrincipal    = "80c_home_principal"
	Line80CLife         = "80c_life_insurance"
	Line80DSelf         = "80d_self"
	Line80DParents      = "80d_parents"
	LineNPSOwn          = "nps_own"      // Tier-1 own contributions: 80CCD(1)/(1B)
	LineNPSEmployer     = "nps_employer" // 80CCD(2)
	Line24bSelfOccupied = "24b_self_occupied"
	Line24bPreEMI       = "24b_pre_emi" // pre-construction interest, treated as self-occupied
	Line24bLetOut       = "24b_let_out"
)

// TaxLine is one tax-relevant amount from a tracked asset for a FY, after
// splitting by ownership share and applying clubbing.
type TaxLine struct {
	AssetType string  `json:"asset_type"`
	AssetID   string  `json:"asset_id"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Amount    float64 `json:"amount"`
	Senior    bool    `json:"senior,omitempty"`     // 80D: a covered person is 60+
	ProfileID string  `json:"profile_id,omitempty"` // taxpayer / claimant
	OwnerID   string  `json:"owner_id,omitempty"`   // legal owner when clubbed to someone else
	Clubbed   bool    `json:"clubbed,omitempty"`
	// ClubbingIgnored marks income that clubbing rules would move to another
	// member but is kept with the owner because the household turned them off.
	ClubbingIgnored bool    `json:"clubbing_ignored,omitempty"`
	SharePct        float64 `json:"share_pct,omitempty"`
}

type IncomeBreakdown struct {
	SalaryFixed     float64 `json:"salary_fixed"`
	SalaryVariable  float64 `json:"salary_variable"`
	EmployerNPS     float64 `json:"employer_nps"`
	ProjectedMonths int     `json:"projected_months"` // months estimated rather than actual
	ActualMonths    int     `json:"actual_months"`
	Pension         float64 `json:"pension"`
	Business        float64 `json:"business"`
	Interest        float64 `json:"interest"`
	Dividends       float64 `json:"dividends"`
	Rent            float64 `json:"rent"`
	OtherRental     float64 `json:"other_rental"`
	OtherIncome     float64 `json:"other_income"`
	STCGEquity      float64 `json:"stcg_equity"`
	LTCGEquity      float64 `json:"ltcg_equity"`
	GainsAtSlab     float64 `json:"gains_at_slab"`
	ClubbedIncome   float64 `json:"clubbed_income"` // others' income added to this member
	Total           float64 `json:"total"`
}

type MemberTaxReport struct {
	ProfileID       string               `json:"profile_id"`
	Name            string               `json:"name"`
	Relationship    string               `json:"relationship"`
	Color           string               `json:"color"`
	Occupation      string               `json:"occupation"`
	Age             int                  `json:"age"`
	Resident        bool                 `json:"resident"`
	HasIncomeRecord bool                 `json:"has_income_record"`
	PreferredRegime string               `json:"preferred_regime"`
	Income          IncomeBreakdown      `json:"income"`
	Comparison      taxengine.Comparison `json:"comparison"`
	ChosenRegime    taxengine.Regime     `json:"chosen_regime"` // preferred if set, else recommended
	ChosenTax       float64              `json:"chosen_tax"`
	ExtraTaxVsBest  float64              `json:"extra_tax_vs_best"` // cost of the preferred regime vs the recommended one
	TaxPaid         float64              `json:"tax_paid"`          // TDS + advance tax
	BalancePayable  float64              `json:"balance_payable"`   // negative = refund
	Lines           []TaxLine            `json:"lines"`
	Warnings        []string             `json:"warnings"`
}

type TaxSuggestion struct {
	ID         string  `json:"id"`
	ProfileID  string  `json:"profile_id"`
	MemberName string  `json:"member_name"`
	Category   string  `json:"category"` // regime, deduction, placement, clubbing, data, capital_gains
	Title      string  `json:"title"`
	Detail     string  `json:"detail"`
	EstSaving  float64 `json:"est_saving"`
}

// TaxSettings are household-wide tax modelling choices (one per user).
// Defaults (zero values) follow the law: clubbing is applied.
type TaxSettings struct {
	ID     primitive.ObjectID `json:"-" bson:"_id,omitempty"`
	UserID string             `json:"-" bson:"user_id"`
	// IgnoreSpouseClubbing taxes income from money given to a spouse in the
	// spouse's hands instead of the giver's (Sec 64(1)(iv) not applied).
	IgnoreSpouseClubbing bool `json:"ignore_spouse_clubbing" bson:"ignore_spouse_clubbing"`
	// IgnoreMinorClubbing taxes a minor child's income in the child's own
	// hands instead of the parent's (Sec 64(1A) not applied).
	IgnoreMinorClubbing bool      `json:"ignore_minor_clubbing" bson:"ignore_minor_clubbing"`
	UpdatedAt           time.Time `json:"updated_at" bson:"updated_at"`
}

type FamilyTaxReport struct {
	FY       string      `json:"fy"`
	Settings TaxSettings `json:"settings"`
	// ClubbingIgnored is income kept with spouses/minors only because clubbing is turned off.
	ClubbingIgnored   float64           `json:"clubbing_ignored"`
	Act               string            `json:"act"`
	SupportedFYs      []string          `json:"supported_fys"`
	Members           []MemberTaxReport `json:"members"`
	TotalTaxBest      float64           `json:"total_tax_best"`      // every member on the recommended regime
	TotalTaxPreferred float64           `json:"total_tax_preferred"` // members' declared regimes
	RegimeSavings     float64           `json:"regime_savings"`      // preferred − best
	UnassignedItems   int               `json:"unassigned_items"`
	DefaultMemberName string            `json:"default_member_name"` // unassigned items are attributed here
	Suggestions       []TaxSuggestion   `json:"suggestions"`
	Disclaimer        string            `json:"disclaimer"`
}

// Placement instruments understood by the optimizer.
var PlacementInstruments = map[string]string{
	"fd":        "Fixed / recurring deposit",
	"debt_mf":   "Debt mutual fund",
	"bond":      "Corporate / government bond",
	"scss":      "Senior Citizens Savings Scheme",
	"equity_mf": "Equity mutual fund / stocks",
	"elss":      "ELSS (tax-saver fund)",
	"ppf":       "Public Provident Fund",
	"ssy":       "Sukanya Samriddhi Yojana",
	"nps":       "NPS Tier-1",
	"savings":   "Savings account",
}

type PlacementRequest struct {
	FY                string  `json:"fy" form:"fy"`
	Amount            float64 `json:"amount" form:"amount" binding:"required,gt=0"`
	Instrument        string  `json:"instrument" form:"instrument" binding:"required"`
	ExpectedReturnPct float64 `json:"expected_return_pct" form:"expected_return_pct" binding:"min=0,max=50"`
	FundedBy          string  `json:"funded_by" form:"funded_by"` // whose money; "" = each candidate's own money
}

type PlacementOption struct {
	ProfileID       string   `json:"profile_id"`
	Name            string   `json:"name"`
	Eligible        bool     `json:"eligible"`
	TaxpayerID      string   `json:"taxpayer_id"` // who is taxed on the income (after clubbing)
	TaxpayerName    string   `json:"taxpayer_name"`
	AnnualIncome    float64  `json:"annual_income"`
	IncrementalTax  float64  `json:"incremental_tax"`  // household tax on the income
	DeductionSaving float64  `json:"deduction_saving"` // tax saved by 80C/80CCD/80D
	NetTaxImpact    float64  `json:"net_tax_impact"`   // incremental tax − deduction saving
	PostTaxIncome   float64  `json:"post_tax_income"`  // annual income − net tax impact
	PostTaxYieldPct float64  `json:"post_tax_yield_pct"`
	Notes           []string `json:"notes"`
}

type PlacementResult struct {
	FY          string            `json:"fy"`
	Request     PlacementRequest  `json:"request"`
	Options     []PlacementOption `json:"options"` // eligible first, best first
	BestProfile string            `json:"best_profile_id"`
	Notes       []string          `json:"notes"`
}
