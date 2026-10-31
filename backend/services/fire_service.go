package services

import (
	"context"
	"math"
	"math/rand"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

const (
	defaultPreReturn     = 10.0
	defaultPostReturn    = 7.0
	defaultInflation     = 6.0
	fireSimulations      = 1000
	preRetirementStdDev  = 0.14
	postRetirementStdDev = 0.08
)

type FIREService struct {
	netWorth   *NetWorthService
	ledgerRepo *repository.LedgerRepo
	sipRepo    *repository.SIPRepo
}

func NewFIREService(netWorth *NetWorthService, ledgerRepo *repository.LedgerRepo, sipRepo *repository.SIPRepo) *FIREService {
	return &FIREService{netWorth: netWorth, ledgerRepo: ledgerRepo, sipRepo: sipRepo}
}

// financialCorpus sums assets that can fund retirement (excludes real estate).
func financialCorpus(items []models.AssetItem) float64 {
	total := 0.0
	for _, it := range items {
		if !it.IsLiability && it.Category != "Real Estate" {
			total += it.Value
		}
	}
	return round2(total)
}

func (s *FIREService) Defaults(ctx context.Context, userID string) (*models.FIREDefaults, error) {
	d := &models.FIREDefaults{
		MonthlyExpensesSource: "none", MonthlySavingsSource: "none",
		PreRetirementReturn: defaultPreReturn, PostRetirementReturn: defaultPostReturn, Inflation: defaultInflation,
	}
	entries, err := s.ledgerRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	sum := BuildLedgerSummary(entries, nil, 6, time.Now())
	if sum.AvgMonthlyExpense > 0 {
		d.MonthlyExpenses, d.MonthlyExpensesSource = round2(sum.AvgMonthlyExpense), "ledger"
	}
	if saving := sum.AvgMonthlyIncome - sum.AvgMonthlyExpense; sum.AvgMonthlyIncome > 0 && saving > 0 {
		d.MonthlySavings, d.MonthlySavingsSource = round2(saving), "ledger"
	} else if sips, _ := s.sipRepo.GetAll(ctx, userID); len(sips) > 0 {
		total := 0.0
		for _, sip := range sips {
			if sip.Status != "active" {
				continue
			}
			switch sip.Frequency {
			case "weekly":
				total += sip.Amount * 52 / 12
			case "quarterly":
				total += sip.Amount / 3
			default:
				total += sip.Amount
			}
		}
		if total > 0 {
			d.MonthlySavings, d.MonthlySavingsSource = round2(total), "sips"
		}
	}
	d.CurrentCorpus = financialCorpus(s.netWorth.ListItems(ctx, userID))
	return d, nil
}

func (s *FIREService) Plan(req models.FIREPlanRequest) (*models.FIREPlan, error) {
	return BuildFIREPlan(req, time.Now().Year())
}

// annuityCorpus is the corpus needed at the start of retirement to fund an
// annual expense (paid at the start of each year, growing with inflation) for
// the given number of years.
func annuityCorpus(annualExpense float64, years int, ret, inflation float64) float64 {
	total, factor := 0.0, 1.0
	q := (1 + inflation) / (1 + ret)
	for k := 0; k < years; k++ {
		total += annualExpense * factor
		factor *= q
	}
	return total
}

type firePlanParams struct {
	req                    models.FIREPlanRequest
	pre, post, inf, stepUp float64
	expenseToday           float64 // annual, already scaled by retirement_expense_pct
}

func (p firePlanParams) contribution(year int, monthly float64) float64 {
	return monthly * 12 * math.Pow(1+p.stepUp, float64(year))
}

// accumulate grows the corpus until retireAge with the given monthly savings.
// Contributions arrive through the year, so they earn roughly half a year.
func (p firePlanParams) accumulate(retireAge int, monthly float64) float64 {
	c := p.req.CurrentCorpus
	for y := 0; y < retireAge-p.req.CurrentAge; y++ {
		c = c*(1+p.pre) + p.contribution(y, monthly)*(1+p.pre/2)
	}
	return c
}

// lastsUntil reports the age at which a corpus runs out when retiring at
// retireAge (0 if it lasts until life expectancy).
func (p firePlanParams) depletesAt(corpus float64, retireAge int) int {
	w := p.expenseToday * math.Pow(1+p.inf, float64(retireAge-p.req.CurrentAge))
	for age := retireAge; age < p.req.LifeExpectancy; age++ {
		if corpus < w-0.5 {
			return age
		}
		corpus = (corpus - w) * (1 + p.post)
		w *= 1 + p.inf
	}
	return 0
}

func BuildFIREPlan(req models.FIREPlanRequest, startYear int) (*models.FIREPlan, error) {
	if req.RetirementAge < req.CurrentAge {
		return nil, invalidf("retirement_age must be at least current_age")
	}
	if req.LifeExpectancy <= req.RetirementAge {
		return nil, invalidf("life_expectancy must be greater than retirement_age")
	}
	if req.PreRetirementReturn == 0 {
		req.PreRetirementReturn = defaultPreReturn
	}
	if req.PostRetirementReturn == 0 {
		req.PostRetirementReturn = defaultPostReturn
	}
	if req.Inflation == 0 {
		req.Inflation = defaultInflation
	}
	if req.RetirementExpensePct == 0 {
		req.RetirementExpensePct = 100
	}
	p := firePlanParams{
		req: req, pre: req.PreRetirementReturn / 100, post: req.PostRetirementReturn / 100,
		inf: req.Inflation / 100, stepUp: req.AnnualStepUpPct / 100,
		expenseToday: req.MonthlyExpenses * 12 * req.RetirementExpensePct / 100,
	}
	years := req.RetirementAge - req.CurrentAge
	retireYears := req.LifeExpectancy - req.RetirementAge

	plan := &models.FIREPlan{Input: req, YearsToRetirement: years}
	plan.AnnualExpenseAtRetire = round2(p.expenseToday * math.Pow(1+p.inf, float64(years)))
	plan.TargetCorpus = round2(annuityCorpus(plan.AnnualExpenseAtRetire, retireYears, p.post, p.inf))
	plan.SWRCorpus = round2(plan.AnnualExpenseAtRetire * 25)
	projected := p.accumulate(req.RetirementAge, req.MonthlySavings)
	plan.ProjectedCorpus = round2(projected)
	plan.Gap = round2(plan.TargetCorpus - projected)
	plan.OnTrack = plan.Gap <= 0

	// Corpus is linear in the monthly saving amount: projected = base + m*perRupee.
	base := p.accumulate(req.RetirementAge, 0)
	perRupee := p.accumulate(req.RetirementAge, 1) - base
	if perRupee > 0 {
		plan.RequiredMonthlySavings = round2(math.Max(0, (plan.TargetCorpus-base)/perRupee))
	} else if base < plan.TargetCorpus {
		plan.RequiredMonthlySavings = -1 // retiring now: savings can't help
	}

	for age := req.CurrentAge; age < req.LifeExpectancy; age++ {
		if p.depletesAt(p.accumulate(age, req.MonthlySavings), age) == 0 {
			plan.EarliestRetirementAge = age
			break
		}
	}
	plan.CorpusDepletesAtAge = p.depletesAt(projected, req.RetirementAge)

	plan.FINumber = round2(annuityCorpus(p.expenseToday, req.LifeExpectancy-req.CurrentAge, p.post, p.inf))
	if plan.FINumber > 0 {
		plan.ProgressPct = round2(math.Min(100, req.CurrentCorpus/plan.FINumber*100))
	}

	// Year-by-year table.
	c := req.CurrentCorpus
	w := plan.AnnualExpenseAtRetire
	for age := req.CurrentAge; age < req.LifeExpectancy; age++ {
		y := age - req.CurrentAge
		row := models.FIREYear{Age: age, Year: startYear + y}
		expenseNow := p.expenseToday * math.Pow(1+p.inf, float64(y))
		row.RequiredCorpus = round2(annuityCorpus(expenseNow, req.LifeExpectancy-age, p.post, p.inf))
		if age < req.RetirementAge {
			row.Phase = "accumulation"
			row.Contribution = round2(p.contribution(y, req.MonthlySavings))
			c = c*(1+p.pre) + row.Contribution*(1+p.pre/2)
		} else {
			row.Phase = "retirement"
			draw := math.Min(w, math.Max(c, 0))
			row.Withdrawal = round2(draw)
			c = (c - draw) * (1 + p.post)
			w *= 1 + p.inf
		}
		row.EndCorpus = round2(c)
		plan.Years = append(plan.Years, row)
	}

	plan.SuccessProbabilityPct = monteCarloSuccess(p, rand.New(rand.NewSource(42)))
	return plan, nil
}

// monteCarloSuccess runs randomised annual returns and returns the share of
// paths where the money lasts until life expectancy.
func monteCarloSuccess(p firePlanParams, rng *rand.Rand) float64 {
	success := 0
	for i := 0; i < fireSimulations; i++ {
		c := p.req.CurrentCorpus
		for y := 0; y < p.req.RetirementAge-p.req.CurrentAge; y++ {
			r := p.pre + rng.NormFloat64()*preRetirementStdDev
			c = c*(1+r) + p.contribution(y, p.req.MonthlySavings)*(1+r/2)
			if c < 0 {
				c = 0
			}
		}
		w := p.expenseToday * math.Pow(1+p.inf, float64(p.req.RetirementAge-p.req.CurrentAge))
		ok := true
		for age := p.req.RetirementAge; age < p.req.LifeExpectancy; age++ {
			if c < w-0.5 {
				ok = false
				break
			}
			r := p.post + rng.NormFloat64()*postRetirementStdDev
			c = (c - w) * (1 + r)
			w *= 1 + p.inf
		}
		if ok {
			success++
		}
	}
	return round2(float64(success) / fireSimulations * 100)
}

// ---------------------------------------------------------------------------
// Household finance overview (dashboard money row and mobile home screen).

type FinanceService struct {
	netWorth     *NetWorthService
	snapshotRepo *repository.NetWorthRepo
	bankRepo     *repository.BankAccountRepo
	budgets      *BudgetService
	insurance    *InsuranceService
	ledgerRepo   *repository.LedgerRepo
}

func NewFinanceService(netWorth *NetWorthService, snapshotRepo *repository.NetWorthRepo, bankRepo *repository.BankAccountRepo, budgets *BudgetService, insurance *InsuranceService, ledgerRepo *repository.LedgerRepo) *FinanceService {
	return &FinanceService{netWorth: netWorth, snapshotRepo: snapshotRepo, bankRepo: bankRepo, budgets: budgets, insurance: insurance, ledgerRepo: ledgerRepo}
}

func (s *FinanceService) Overview(ctx context.Context, userID string) (*models.FinanceOverview, error) {
	now := time.Now()
	o := &models.FinanceOverview{Budgets: []models.BudgetStatus{}, UpcomingPremiums: []models.UpcomingPremium{}}

	items := s.netWorth.ListItems(ctx, userID)
	for _, it := range items {
		if it.IsLiability {
			o.TotalLiabilities += it.Value
			continue
		}
		o.TotalAssets += it.Value
		switch it.Category {
		case "Gold":
			o.GoldValue += it.Value
		case "Real Estate":
			o.RealEstateValue += it.Value
		}
	}
	o.TotalAssets, o.TotalLiabilities = round2(o.TotalAssets), round2(o.TotalLiabilities)
	o.NetWorth = round2(o.TotalAssets - o.TotalLiabilities)
	o.GoldValue, o.RealEstateValue = round2(o.GoldValue), round2(o.RealEstateValue)
	lastMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0).Format("2006-01")
	if snap, err := s.snapshotRepo.GetByMonth(ctx, userID, lastMonth); err == nil && snap != nil {
		change := round2(o.NetWorth - snap.NetWorth)
		o.NetWorthChangeMonth = &change
	}

	accounts, err := s.bankRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	bank, cash, emergency, stale := LiquidTotals(accounts)
	o.BankBalance, o.CashInHand, o.LiquidBalance, o.EmergencyFund, o.StaleBalances = round2(bank), round2(cash), round2(bank+cash), round2(emergency), stale

	summary, err := s.budgets.Summary(ctx, userID, 6)
	if err != nil {
		return nil, err
	}
	cm := summary.CurrentMonth
	o.MonthIncome, o.MonthExpense, o.MonthSavingsRatePct = cm.Income, cm.Expense, cm.SavingsRatePct
	o.AvgMonthlyExpense, o.AvgSavingsRatePct = summary.AvgMonthlyExpense, summary.AvgSavingsRatePct
	o.PendingReview = summary.PendingReviewCount
	if summary.Budgets != nil {
		o.Budgets = summary.Budgets
	}
	for _, b := range o.Budgets {
		switch b.Status {
		case "over":
			o.BudgetsOver++
		case "warning":
			o.BudgetsWarning++
		}
	}
	if o.AvgMonthlyExpense > 0 {
		o.EmergencyMonths = round2(o.EmergencyFund / o.AvgMonthlyExpense)
		o.FINumber = round2(o.AvgMonthlyExpense * 12 * 25)
		o.FIProgressPct = round2(math.Min(100, financialCorpus(items)/o.FINumber*100))
	}

	ins, err := s.insurance.Summary(ctx, userID, 0)
	if err != nil {
		return nil, err
	}
	o.LifeCover, o.HealthCover = ins.TotalLifeCover, ins.TotalHealthCover
	for _, p := range ins.UpcomingPremiums {
		if p.DaysLeft <= 30 {
			o.UpcomingPremiums = append(o.UpcomingPremiums, p)
		}
	}
	return o, nil
}
