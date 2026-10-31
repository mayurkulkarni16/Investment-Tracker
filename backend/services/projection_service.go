package services

import (
	"context"
	"hash/fnv"
	"math"
	"math/rand"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

const projectionSimulations = 2000

type ProjectionService struct {
	mfRepo            *repository.MutualFundRepo
	fdRepo            *repository.FixedDepositRepo
	pfRepo            *repository.ProvidentFundRepo
	stockRepo         *repository.StockRepo
	bondRepo          *repository.CorporateBondRepo
	npsRepo           *repository.NPSRepo
	sipRepo           *repository.SIPRepo
	homeLoanRepo      *repository.HomeLoanRepo
	personalLoanRepo  *repository.PersonalLoanRepo
	vehicleRentalRepo *repository.VehicleRentalRepo
}

func NewProjectionService(
	mfRepo *repository.MutualFundRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	stockRepo *repository.StockRepo,
	bondRepo *repository.CorporateBondRepo,
	npsRepo *repository.NPSRepo,
	sipRepo *repository.SIPRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	vehicleRentalRepo *repository.VehicleRentalRepo,
) *ProjectionService {
	return &ProjectionService{
		mfRepo: mfRepo, fdRepo: fdRepo, pfRepo: pfRepo, stockRepo: stockRepo, bondRepo: bondRepo,
		npsRepo: npsRepo, sipRepo: sipRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		vehicleRentalRepo: vehicleRentalRepo,
	}
}

type ProjectionOptions struct {
	Years    int
	Scenario string // base, bull, bear
	// InflationRate (annual %) > 0 expresses all projected values in today's money.
	InflationRate float64
}

// Scenarios shift the expected return of market-linked holdings only; contractual
// returns (FD, PF, bond coupons, rental agreements) are unaffected.
var scenarioShifts = map[string]struct{ equity, debt float64 }{
	"base": {0, 0},
	"bull": {3, 0.5},
	"bear": {-4, -1},
}

// holdingSpec is the input to the projection engine for one holding. Market-linked
// holdings are simulated; scheduled holdings carry a precomputed nominal path.
type holdingSpec struct {
	name, category string
	current        float64
	rate, vol      float64 // annual %, rate is the median (geometric) return
	beta           float64 // loading on the common market factor (0..1)
	monthly        float64 // contribution added at the start of each month
	contribMonths  int     // months with contributions; < 0 means for the whole horizon
	scheduled      []float64
	note           string
}

func (s *ProjectionService) GetProjections(ctx context.Context, userID string, opts ProjectionOptions) (*models.ProjectionsResponse, error) {
	if opts.Years <= 0 || opts.Years > 30 {
		opts.Years = 5
	}
	if _, ok := scenarioShifts[opts.Scenario]; !ok {
		opts.Scenario = "base"
	}
	if opts.InflationRate < 0 || opts.InflationRate > 20 {
		opts.InflationRate = 0
	}
	months := opts.Years * 12
	now := time.Now()

	specs := s.collectHoldings(ctx, userID, opts.Scenario, months, now)
	liabilities := s.collectLiabilities(ctx, userID, months)

	seed := projectionSeed(userID)
	investments, aggregate := runProjection(specs, months, projectionSimulations, seed, now)

	resp := &models.ProjectionsResponse{
		Investments:   investments,
		Scenario:      opts.Scenario,
		InflationRate: opts.InflationRate,
		Simulations:   projectionSimulations,
		Liabilities:   []models.LiabilityProjection{},
	}
	if len(specs) == 0 {
		resp.Investments = []models.InvestmentProjection{}
	}

	// Liabilities and net worth (assets are stochastic, liabilities deterministic).
	liabTotals := make([]float64, months+1)
	for _, l := range liabilities {
		points := make([]models.ProjectionPoint, months+1)
		for m := 0; m <= months; m++ {
			liabTotals[m] += l.path[m]
			points[m] = flatPoint(now, m, l.path[m])
		}
		resp.Liabilities = append(resp.Liabilities, models.LiabilityProjection{
			Name: l.name, Category: l.category, CurrentValue: l.path[0],
			InterestRate: l.rate, MonthlyEMI: l.emi, Projections: points,
		})
	}
	resp.LiabilitiesMonthly = make([]models.ProjectionPoint, months+1)
	resp.NetWorthMonthly = make([]models.ProjectionPoint, months+1)
	for m := 0; m <= months; m++ {
		resp.LiabilitiesMonthly[m] = flatPoint(now, m, liabTotals[m])
		nw := flatPoint(now, m, 0)
		if aggregate != nil {
			a := aggregate[m]
			nw = models.ProjectionPoint{Year: a.Year, Month: a.Month,
				Value: a.Value - liabTotals[m], P10: a.P10 - liabTotals[m], P50: a.P50 - liabTotals[m], P90: a.P90 - liabTotals[m]}
		} else {
			nw.Value, nw.P10, nw.P50, nw.P90 = -liabTotals[m], -liabTotals[m], -liabTotals[m], -liabTotals[m]
		}
		resp.NetWorthMonthly[m] = nw
	}
	resp.AggregateMonthly = aggregate

	if opts.InflationRate > 0 {
		deflate := func(points []models.ProjectionPoint) {
			for m := range points {
				f := math.Pow(1+opts.InflationRate/100, float64(m)/12)
				points[m].Value = math.Round(points[m].Value / f)
				points[m].P10 = math.Round(points[m].P10 / f)
				points[m].P50 = math.Round(points[m].P50 / f)
				points[m].P90 = math.Round(points[m].P90 / f)
			}
		}
		for i := range resp.Investments {
			deflate(resp.Investments[i].Projections)
		}
		for i := range resp.Liabilities {
			deflate(resp.Liabilities[i].Projections)
		}
		deflate(resp.AggregateMonthly)
		deflate(resp.LiabilitiesMonthly)
		deflate(resp.NetWorthMonthly)
	}

	for _, inv := range resp.Investments {
		resp.TotalCurrent += inv.CurrentValue
	}
	resp.TotalLiabilities = liabTotals[0]
	resp.CurrentNetWorth = resp.TotalCurrent - resp.TotalLiabilities

	if len(resp.AggregateMonthly) > 0 {
		at := func(m int) (float64, *models.ProjectedRange) {
			pt := resp.AggregateMonthly[m]
			return pt.P50, &models.ProjectedRange{P10: pt.P10, P50: pt.P50, P90: pt.P90}
		}
		if months >= 12 {
			resp.TotalProjected1Y, resp.Projected1YRange = at(12)
		}
		if months >= 36 {
			resp.TotalProjected3Y, resp.Projected3YRange = at(36)
		}
		if months >= 60 {
			resp.TotalProjected5Y, resp.Projected5YRange = at(60)
		}
		resp.TotalProjectedEnd, resp.ProjectedEndRange = at(months)
	}
	return resp, nil
}

func (s *ProjectionService) collectHoldings(ctx context.Context, userID, scenario string, months int, now time.Time) []holdingSpec {
	shift := scenarioShifts[scenario]
	shifted := func(rate, equityShare float64) float64 {
		return round1(rate + equityShare*shift.equity + (1-equityShare)*shift.debt)
	}
	var specs []holdingSpec

	// Mutual funds, with recurring contributions from active SIPs.
	type sipPlan struct {
		monthly float64
		months  int
	}
	sipByFund := map[string]sipPlan{}
	if sips, err := s.sipRepo.GetAll(ctx, userID); err == nil {
		for _, sip := range sips {
			if sip.Status != "active" || sip.FundID == "" {
				continue
			}
			n := -1
			if sip.EndDate != nil {
				if n = monthsBetween(now, *sip.EndDate); n <= 0 {
					continue
				}
			}
			p := sipByFund[sip.FundID]
			p.monthly += monthlyEquivalent(sip.Amount, sip.Frequency)
			if p.months >= 0 && (n < 0 || n > p.months) {
				p.months = n
			}
			sipByFund[sip.FundID] = p
		}
	}
	if funds, err := s.mfRepo.GetAll(ctx, userID); err == nil {
		for i := range funds {
			mf := &funds[i]
			if mf.CurrentValue <= 0 {
				continue
			}
			prof := MutualFundProfile(mf.FundType)
			rate := prof.ExpectedReturn
			note := "Category long-term assumption"
			if first, ok := earliestMFTransaction(mf); ok {
				rate = BlendedReturn(ComputeMutualFundXIRR(mf), yearsSince(first, now), prof)
				note = "Your XIRR blended with category long-term return"
			}
			sip := sipByFund[mf.ID.Hex()]
			contribMonths := sip.months
			if sip.monthly == 0 {
				contribMonths = 0
			}
			specs = append(specs, holdingSpec{
				name: mf.FundName, category: "mutual_fund", current: mf.CurrentValue,
				rate: shifted(rate, prof.EquityShare), vol: prof.Volatility, beta: marketBeta(prof.EquityShare),
				monthly: sip.monthly, contribMonths: contribMonths, note: note,
			})
		}
	}

	// Stocks
	if stocks, err := s.stockRepo.GetAll(ctx, userID); err == nil {
		for i := range stocks {
			st := &stocks[i]
			if st.CurrentValue <= 0 {
				continue
			}
			rate := StockProfile.ExpectedReturn
			note := "Long-term equity assumption"
			if first, ok := earliestStockTransaction(st); ok {
				rate = BlendedReturn(ComputeStockXIRR(st), yearsSince(first, now), StockProfile)
				note = "Your XIRR blended with long-term equity return"
			}
			name := st.StockName
			if name == "" {
				name = st.Symbol
			}
			specs = append(specs, holdingSpec{
				name: name, category: "stock", current: st.CurrentValue,
				rate: shifted(rate, 1), vol: StockProfile.Volatility, beta: 0.8, note: note,
			})
		}
	}

	// NPS: allocation-weighted profile, contributions at last year's average pace.
	if accounts, err := s.npsRepo.GetAll(ctx, userID); err == nil {
		yearAgo := now.AddDate(-1, 0, 0)
		for i := range accounts {
			nps := &accounts[i]
			if nps.CurrentValue <= 0 || nps.Status != "active" {
				continue
			}
			prof := NPSProfile(nps)
			recent := 0.0
			for _, c := range nps.Contributions {
				if !c.Date.Before(yearAgo) {
					recent += c.Amount
				}
			}
			specs = append(specs, holdingSpec{
				name: nps.AccountHolderName + " NPS (" + string(nps.AccountType) + ")", category: "nps",
				current: nps.CurrentValue, rate: shifted(prof.ExpectedReturn, prof.EquityShare), vol: prof.Volatility,
				beta: marketBeta(prof.EquityShare), monthly: recent / 12, contribMonths: -1,
				note: "Scheme allocation-weighted long-term return",
			})
		}
	}

	// Provident fund: declared rate, monthly contributions continue.
	if pfs, err := s.pfRepo.GetAll(ctx, userID); err == nil {
		for _, pf := range pfs {
			if pf.CurrentBalance <= 0 {
				continue
			}
			rate := pf.InterestRate
			if rate <= 0 {
				rate = 8.25
			}
			contrib := 0.0
			if n := len(pf.FinancialYearEntries); n > 0 {
				latest := pf.FinancialYearEntries[n-1]
				if len(latest.MonthlyContributions) > 0 {
					total := 0.0
					for _, mc := range latest.MonthlyContributions {
						total += mc.EmployeeContribution + mc.EmployerContribution
					}
					contrib = total / float64(len(latest.MonthlyContributions))
				}
			}
			specs = append(specs, holdingSpec{
				name: pf.EmployerName + " - " + string(pf.AccountType), category: "provident_fund",
				current: pf.CurrentBalance, rate: rate, monthly: contrib, contribMonths: -1,
				scheduled: compoundPath(pf.CurrentBalance, rate, contrib, months),
				note:      "Declared rate; contributions assumed to continue",
			})
		}
	}

	// Fixed deposits: accrued value to maturity, then held as cash.
	if fds, err := s.fdRepo.GetAll(ctx, userID); err == nil {
		for i := range fds {
			fd := &fds[i]
			NormalizeFixedDeposit(fd, now)
			if fd.Status != models.FDActive {
				continue
			}
			path := make([]float64, months+1)
			for m := 0; m <= months; m++ {
				path[m] = fdValueAt(fd, now, addMonthsClamped(now, m))
			}
			name := fd.BankName
			if fd.FDNumber != "" {
				name += " - " + fd.FDNumber
			}
			specs = append(specs, holdingSpec{
				name: name, category: "fixed_deposit", current: path[0], rate: fd.InterestRate, scheduled: path,
				note: "Contract rate; proceeds held as cash after maturity",
			})
		}
	}

	// Corporate bonds: principal outstanding plus coupons received (not reinvested).
	if bonds, err := s.bondRepo.GetAll(ctx, userID); err == nil {
		for i := range bonds {
			b := &bonds[i]
			remaining := b.InvestmentAmount
			for _, r := range b.PrincipalRepayments {
				if r.Status == models.PayoutReceived || !r.ScheduledDate.After(now) {
					remaining -= r.Amount
				}
			}
			if remaining <= 0 {
				continue
			}
			path := make([]float64, months+1)
			for m := 0; m <= months; m++ {
				t := addMonthsClamped(now, m)
				v := remaining
				for _, p := range b.InterestPayouts {
					if p.Status == models.PayoutPending && p.ScheduledDate.After(now) && !p.ScheduledDate.After(t) {
						v += p.Amount
					}
				}
				path[m] = v
			}
			specs = append(specs, holdingSpec{
				name: b.BondName, category: "corporate_bond", current: remaining, rate: b.CouponRate, scheduled: path,
				note: "Scheduled coupons and principal, held as cash",
			})
		}
	}

	// Vehicle rentals: book value plus promised payouts, assuming every payout is made on time.
	if rentals, err := s.vehicleRentalRepo.GetAll(ctx, userID); err == nil {
		for i := range rentals {
			v := &rentals[i]
			ComputeVehicleRental(v, now)
			if v.Status == models.VehicleRentalCompleted || v.CurrentValue <= 0 && v.NextPayoutDate == nil {
				continue
			}
			path := make([]float64, months+1)
			for m := 0; m <= months; m++ {
				t := addMonthsClamped(now, m)
				val := vehicleRentalBookValue(v, t)
				for _, p := range v.Payouts {
					if p.Status == models.PayoutPending && p.ScheduledDate.After(now) && !p.ScheduledDate.After(t) {
						val += p.Amount
					}
				}
				path[m] = val
			}
			name := v.Provider
			if v.VehicleModel != "" {
				name += " - " + v.VehicleModel
			}
			specs = append(specs, holdingSpec{
				name: name, category: "vehicle_rental", current: path[0], rate: promisedRentalReturn(v, now),
				scheduled: path, note: "Assumes every remaining payout is paid on time",
			})
		}
	}

	return specs
}

type liabilitySpec struct {
	name, category string
	rate, emi      float64
	path           []float64
}

func (s *ProjectionService) collectLiabilities(ctx context.Context, userID string, months int) []liabilitySpec {
	var out []liabilitySpec
	if loans, err := s.homeLoanRepo.GetAll(ctx, userID); err == nil {
		for _, l := range loans {
			if l.Status != models.LoanActive || l.OutstandingPrincipal <= 0 {
				continue
			}
			out = append(out, liabilitySpec{
				name: l.BankName + " Home Loan", category: "home_loan", rate: l.InterestRate, emi: l.EMIAmount,
				path: amortizationPath(l.OutstandingPrincipal, l.InterestRate, l.EMIAmount, months),
			})
		}
	}
	if loans, err := s.personalLoanRepo.GetAll(ctx, userID); err == nil {
		for _, l := range loans {
			if l.Status != models.LoanActive || l.OutstandingPrincipal <= 0 {
				continue
			}
			out = append(out, liabilitySpec{
				name: l.LenderName + " Personal Loan", category: "personal_loan", rate: l.InterestRate, emi: l.EMIAmount,
				path: amortizationPath(l.OutstandingPrincipal, l.InterestRate, l.EMIAmount, months),
			})
		}
	}
	return out
}

// runProjection simulates all market-linked holdings jointly so that portfolio
// percentiles reflect diversification, and adds scheduled holdings on top.
// Each month: v = (v + contribution) * exp(mu + sigma*z), with mu = ln(1+rate)/12
// (so the median path grows at exactly `rate`) and z = beta*M + sqrt(1-beta^2)*e,
// where M is a market shock shared by all holdings in the same simulation.
// Memory is O(months * sims) regardless of the number of holdings.
func runProjection(specs []holdingSpec, months, sims int, seed int64, now time.Time) ([]models.InvestmentProjection, []models.ProjectionPoint) {
	if len(specs) == 0 {
		return nil, nil
	}
	rng := rand.New(rand.NewSource(seed))
	market := make([]float32, sims*months)
	for i := range market {
		market[i] = float32(rng.NormFloat64())
	}

	totals := make([]float64, (months+1)*sims) // [m*sims + s]
	values := make([]float64, (months+1)*sims)
	scratch := make([]float64, sims)
	out := make([]models.InvestmentProjection, 0, len(specs))

	for _, h := range specs {
		proj := models.InvestmentProjection{
			Name: h.name, Category: h.category, CurrentValue: math.Round(h.current),
			AssumedRatePct: h.rate, VolatilityPct: h.vol, MonthlyContribution: math.Round(h.monthly),
			Note: h.note, Projections: make([]models.ProjectionPoint, months+1),
		}

		if h.scheduled != nil {
			proj.Basis = models.ProjectionScheduled
			proj.VolatilityPct = 0
			for m := 0; m <= months; m++ {
				v := h.scheduled[m]
				row := totals[m*sims : (m+1)*sims]
				for s := range row {
					row[s] += v
				}
				proj.Projections[m] = flatPoint(now, m, v)
			}
			out = append(out, proj)
			continue
		}

		proj.Basis = models.ProjectionSimulated
		mu := math.Log(1+h.rate/100) / 12
		sigma := h.vol / 100 / math.Sqrt(12)
		idio := math.Sqrt(math.Max(0, 1-h.beta*h.beta))
		for s := 0; s < sims; s++ {
			v := h.current
			values[s] = v
			totals[s] += v
			for m := 1; m <= months; m++ {
				if h.contribMonths < 0 || m <= h.contribMonths {
					v += h.monthly
				}
				z := h.beta*float64(market[s*months+m-1]) + idio*rng.NormFloat64()
				v *= math.Exp(mu + sigma*z)
				values[m*sims+s] = v
				totals[m*sims+s] += v
			}
		}
		for m := 0; m <= months; m++ {
			p10, p50, p90 := percentiles(values[m*sims:(m+1)*sims], scratch)
			proj.Projections[m] = point(now, m, p10, p50, p90)
		}
		out = append(out, proj)
	}

	aggregate := make([]models.ProjectionPoint, months+1)
	for m := 0; m <= months; m++ {
		p10, p50, p90 := percentiles(totals[m*sims:(m+1)*sims], scratch)
		aggregate[m] = point(now, m, p10, p50, p90)
	}
	return out, aggregate
}

func marketBeta(equityShare float64) float64 {
	return 0.9*equityShare + 0.2*(1-equityShare)
}

func point(now time.Time, m int, p10, p50, p90 float64) models.ProjectionPoint {
	t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, m, 0)
	return models.ProjectionPoint{
		Year: t.Year(), Month: int(t.Month()),
		Value: math.Round(p50), P10: math.Round(p10), P50: math.Round(p50), P90: math.Round(p90),
	}
}

func flatPoint(now time.Time, m int, v float64) models.ProjectionPoint {
	return point(now, m, v, v, v)
}

// percentiles returns the 10th, 50th and 90th percentiles of xs (unchanged)
// using quickselect on scratch.
func percentiles(xs, scratch []float64) (float64, float64, float64) {
	a := scratch[:len(xs)]
	copy(a, xs)
	n := len(a)
	k10, k50, k90 := n*10/100, n/2, n*90/100
	p50 := selectKth(a, k50)
	p10 := p50
	if k10 < k50 {
		p10 = selectKth(a[:k50], k10)
	}
	p90 := p50
	if k90 > k50 {
		p90 = selectKth(a[k50+1:], k90-k50-1)
	}
	return p10, p50, p90
}

// selectKth partially orders a so that a[k] is the k-th smallest element, with
// a[:k] <= a[k] <= a[k+1:], and returns it.
func selectKth(a []float64, k int) float64 {
	lo, hi := 0, len(a)-1
	for lo < hi {
		mid := lo + (hi-lo)/2
		x, y, z := a[lo], a[mid], a[hi]
		pivot := math.Max(math.Min(x, y), math.Min(math.Max(x, y), z))
		i, j := lo, hi
		for i <= j {
			for a[i] < pivot {
				i++
			}
			for a[j] > pivot {
				j--
			}
			if i <= j {
				a[i], a[j] = a[j], a[i]
				i++
				j--
			}
		}
		switch {
		case k <= j:
			hi = j
		case k >= i:
			lo = i
		default:
			return a[k]
		}
	}
	return a[k]
}

// compoundPath grows a balance at an annual rate with a monthly contribution.
func compoundPath(start, annualRate, monthly float64, months int) []float64 {
	path := make([]float64, months+1)
	path[0] = start
	g := math.Pow(1+annualRate/100, 1.0/12)
	v := start
	for m := 1; m <= months; m++ {
		v = (v + monthly) * g
		path[m] = v
	}
	return path
}

// amortizationPath projects a loan's outstanding principal paying a fixed EMI.
func amortizationPath(outstanding, annualRate, emi float64, months int) []float64 {
	path := make([]float64, months+1)
	path[0] = outstanding
	r := annualRate / 100 / 12
	b := outstanding
	for m := 1; m <= months; m++ {
		if emi > 0 {
			b = b*(1+r) - emi
			if b < 0 {
				b = 0
			}
		}
		path[m] = b
	}
	return path
}

// fdValueAt is the FD's value at time t: accrued value (quarterly compounding)
// for cumulative FDs, or principal plus interest paid out after `now` for
// payout FDs. After maturity the proceeds are assumed to be held as cash.
func fdValueAt(fd *models.FixedDeposit, now, t time.Time) float64 {
	if fd.InterestType == models.InterestCumulative {
		if !t.Before(fd.MaturityDate) && fd.MaturityAmount > 0 {
			return fd.MaturityAmount
		}
		years := t.Sub(fd.StartDate).Hours() / 8766
		if years < 0 {
			years = 0
		}
		v := fd.PrincipalAmount * math.Pow(1+fd.InterestRate/400, 4*years)
		if fd.MaturityAmount > 0 && v > fd.MaturityAmount {
			v = fd.MaturityAmount
		}
		return v
	}
	v := fd.PrincipalAmount
	for _, p := range fd.InterestPayouts {
		if p.Status == models.PayoutPending && p.ScheduledDate.After(now) && !p.ScheduledDate.After(t) {
			v += p.Amount
		}
	}
	return v
}

// promisedRentalReturn is the annualised return implied by the remaining
// promised payouts and buyback, measured from today's book value.
func promisedRentalReturn(v *models.VehicleRental, now time.Time) float64 {
	if v.CurrentValue <= 0 {
		return 0
	}
	flows := []Cashflow{{Date: now, Amount: -v.CurrentValue}}
	for _, p := range v.Payouts {
		if p.Status == models.PayoutPending && p.ScheduledDate.After(now) {
			flows = append(flows, Cashflow{Date: p.ScheduledDate, Amount: p.Amount})
		}
	}
	if v.BuybackAmount > 0 && v.BuybackStatus == models.PayoutPending {
		flows = append(flows, Cashflow{Date: v.AgreementEndDate, Amount: v.BuybackAmount})
	}
	x, err := CalculateXIRR(flows)
	if err != nil || math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	return round1(x * 100)
}

func monthlyEquivalent(amount float64, frequency string) float64 {
	switch frequency {
	case "weekly":
		return amount * 52 / 12
	case "fortnightly":
		return amount * 26 / 12
	case "daily":
		return amount * 21
	case "quarterly":
		return amount / 3
	case "half_yearly", "biannually":
		return amount / 6
	case "yearly", "annually":
		return amount / 12
	default:
		return amount
	}
}

func monthsBetween(from, to time.Time) int {
	return (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
}

func yearsSince(t, now time.Time) float64 {
	return now.Sub(t).Hours() / 8766
}

func earliestMFTransaction(mf *models.MutualFund) (time.Time, bool) {
	var first time.Time
	for _, t := range mf.Transactions {
		if first.IsZero() || t.Date.Before(first) {
			first = t.Date
		}
	}
	return first, !first.IsZero()
}

func earliestStockTransaction(st *models.Stock) (time.Time, bool) {
	var first time.Time
	for _, t := range st.Transactions {
		if first.IsZero() || t.Date.Before(first) {
			first = t.Date
		}
	}
	return first, !first.IsZero()
}

// projectionSeed makes results stable across page loads for the same user.
func projectionSeed(userID string) int64 {
	h := fnv.New64a()
	h.Write([]byte(userID))
	return int64(h.Sum64() & math.MaxInt64)
}
