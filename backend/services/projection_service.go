package services

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

const monteCarloSimulations = 2000

type ProjectionService struct {
	mfRepo    *repository.MutualFundRepo
	fdRepo    *repository.FixedDepositRepo
	pfRepo    *repository.ProvidentFundRepo
	stockRepo *repository.StockRepo
	bondRepo  *repository.CorporateBondRepo
	npsRepo   *repository.NPSRepo
	sipRepo   *repository.SIPRepo
}

func NewProjectionService(
	mfRepo *repository.MutualFundRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	stockRepo *repository.StockRepo,
	bondRepo *repository.CorporateBondRepo,
	npsRepo *repository.NPSRepo,
	sipRepo *repository.SIPRepo,
) *ProjectionService {
	return &ProjectionService{
		mfRepo:    mfRepo,
		fdRepo:    fdRepo,
		pfRepo:    pfRepo,
		stockRepo: stockRepo,
		bondRepo:  bondRepo,
		npsRepo:   npsRepo,
		sipRepo:   sipRepo,
	}
}

func (s *ProjectionService) GetProjections(ctx context.Context, userID string, years int) (*models.ProjectionsResponse, error) {
	return s.GetProjectionsWithScenario(ctx, userID, years, "base")
}

func (s *ProjectionService) GetProjectionsWithScenario(ctx context.Context, userID string, years int, scenario string) (*models.ProjectionsResponse, error) {
	if years <= 0 || years > 30 {
		years = 5
	}

	// Scenario rate multiplier
	rateMultiplier := 1.0
	switch scenario {
	case "bull":
		rateMultiplier = 1.5
	case "bear":
		rateMultiplier = 0.4
	}

	totalMonths := years * 12
	now := time.Now()

	resp := &models.ProjectionsResponse{
		Investments: []models.InvestmentProjection{},
		Scenario:    scenario,
	}

	// Mutual Funds — use historical CAGR per fund, fallback 12% for equity, 8% for debt
	// Look up active SIPs to model recurring monthly contributions
	funds, err := s.mfRepo.GetAll(ctx, userID)
	activeSIPs, _ := s.sipRepo.GetAll(ctx, userID)
	sipByFundID := make(map[string]float64)
	for _, sip := range activeSIPs {
		if sip.Status == "active" && sip.FundID != "" {
			sipByFundID[sip.FundID] += sip.Amount
		}
	}
	if err == nil {
		for _, mf := range funds {
			if mf.CurrentValue <= 0 {
				continue
			}
			rate := estimateMFRate(&mf) * rateMultiplier
			vol := estimateMFVolatility(&mf)
			monthlySIP := sipByFundID[mf.ID.Hex()]
			proj := projectGrowthWithSIP(mf.FundName, "mutual_fund", mf.CurrentValue, rate, vol, totalMonths, now, monthlySIP)
			resp.Investments = append(resp.Investments, proj)
		}
	}

	// Fixed Deposits — use actual FD rate, project until maturity then flat
	fds, err := s.fdRepo.GetAll(ctx, userID)
	if err == nil {
		for _, fd := range fds {
			if fd.Status != "active" {
				continue
			}
			proj := projectFD(fd, totalMonths, now)
			resp.Investments = append(resp.Investments, proj)
		}
	}

	// Provident Fund — use current PF rate (default 8.25%), model ongoing monthly contributions
	pfs, err := s.pfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, pf := range pfs {
			if pf.CurrentBalance <= 0 {
				continue
			}
			rate := pf.InterestRate
			if rate <= 0 {
				rate = 8.25
			}
			rate *= rateMultiplier

			// Estimate monthly contribution from the latest FY's data
			monthlyContrib := 0.0
			if len(pf.FinancialYearEntries) > 0 {
				latestFY := pf.FinancialYearEntries[len(pf.FinancialYearEntries)-1]
				if len(latestFY.MonthlyContributions) > 0 {
					totalContrib := 0.0
					for _, mc := range latestFY.MonthlyContributions {
						totalContrib += mc.EmployeeContribution + mc.EmployerContribution
					}
					monthlyContrib = totalContrib / float64(len(latestFY.MonthlyContributions))
				}
			}

			proj := projectGrowthWithSIP(pf.EmployerName+" - "+string(pf.AccountType), "provident_fund", pf.CurrentBalance, rate, 1.0, totalMonths, now, monthlyContrib)
			resp.Investments = append(resp.Investments, proj)
		}
	}

	// Stocks — assume 12% average annual return
	stocks, err := s.stockRepo.GetAll(ctx, userID)
	if err == nil {
		for _, stock := range stocks {
			if stock.CurrentValue <= 0 {
				continue
			}
			rate := 12.0
			// If we have gain data, use actual CAGR
			if stock.TotalInvested > 0 && stock.CurrentValue > stock.TotalInvested {
				rate = estimateStockCAGR(&stock)
			}
			rate *= rateMultiplier
			vol := estimateStockVolatility(&stock)
			proj := projectGrowth(stock.StockName, "stock", stock.CurrentValue, rate, vol, totalMonths, now)
			resp.Investments = append(resp.Investments, proj)
		}
	}

	// Corporate Bonds — use coupon rate, project remaining returns
	bonds, err := s.bondRepo.GetAll(ctx, userID)
	if err == nil {
		for _, bond := range bonds {
			if bond.Status == "matured" {
				continue
			}
			proj := projectGrowth(bond.BondName, "corporate_bond", bond.RemainingPrincipal, bond.CouponRate*rateMultiplier, 3.0, totalMonths, now)
			resp.Investments = append(resp.Investments, proj)
		}
	}

	// NPS — blended return based on equity/debt allocation, model ongoing contributions
	npsAccounts, err := s.npsRepo.GetAll(ctx, userID)
	if err == nil {
		for _, nps := range npsAccounts {
			if nps.CurrentValue <= 0 || nps.Status != "active" {
				continue
			}
			// Blended return: equity 12%, corporate bond 9%, govt sec 7.5%, alternate 10%
			blendedRate := (nps.EquityPct*12.0 + nps.CorporateBondPct*9.0 + nps.GovtSecPct*7.5 + nps.AlternatePct*10.0) / 100.0
			if blendedRate <= 0 {
				blendedRate = 10.0 // fallback
			}
			blendedRate *= rateMultiplier
			// Blended volatility: equity 18%, debt 3%, govt 2%, alternate 8%
			blendedVol := (nps.EquityPct*18.0 + nps.CorporateBondPct*3.0 + nps.GovtSecPct*2.0 + nps.AlternatePct*8.0) / 100.0
			if blendedVol <= 0 {
				blendedVol = 8.0
			}

			// Estimate monthly contribution from contributions in the last 12 months
			monthlyContrib := 0.0
			twelveMonthsAgo := now.AddDate(-1, 0, 0)
			recentTotal := 0.0
			recentCount := 0
			for _, c := range nps.Contributions {
				if !c.Date.Before(twelveMonthsAgo) {
					recentTotal += c.Amount
					recentCount++
				}
			}
			if recentCount > 0 {
				monthlyContrib = recentTotal / 12.0
			}

			name := nps.AccountHolderName + " NPS (" + string(nps.AccountType) + ")"
			proj := projectGrowthWithSIP(name, "nps", nps.CurrentValue, blendedRate, blendedVol, totalMonths, now, monthlyContrib)
			resp.Investments = append(resp.Investments, proj)
		}
	}
	// Aggregate monthly projections
	resp.AggregateMonthly = aggregateProjections(resp.Investments, totalMonths, now)

	// Summary values
	resp.TotalCurrent = 0
	for _, inv := range resp.Investments {
		resp.TotalCurrent += inv.CurrentValue
	}
	if len(resp.AggregateMonthly) > 0 {
		for _, pt := range resp.AggregateMonthly {
			monthsFromNow := (pt.Year-now.Year())*12 + (pt.Month - int(now.Month()))
			if monthsFromNow == 12 {
				resp.TotalProjected1Y = pt.P50
				resp.Projected1YRange = &models.ProjectedRange{P10: pt.P10, P50: pt.P50, P90: pt.P90}
			}
			if monthsFromNow == 36 {
				resp.TotalProjected3Y = pt.P50
				resp.Projected3YRange = &models.ProjectedRange{P10: pt.P10, P50: pt.P50, P90: pt.P90}
			}
			if monthsFromNow == 60 {
				resp.TotalProjected5Y = pt.P50
				resp.Projected5YRange = &models.ProjectedRange{P10: pt.P10, P50: pt.P50, P90: pt.P90}
			}
		}
		// If projection period is shorter, use last available
		last := resp.AggregateMonthly[len(resp.AggregateMonthly)-1]
		if resp.TotalProjected5Y == 0 && years >= 5 {
			resp.TotalProjected5Y = last.P50
			resp.Projected5YRange = &models.ProjectedRange{P10: last.P10, P50: last.P50, P90: last.P90}
		}
		if resp.TotalProjected3Y == 0 && years >= 3 {
			resp.TotalProjected3Y = last.P50
			resp.Projected3YRange = &models.ProjectedRange{P10: last.P10, P50: last.P50, P90: last.P90}
		}
		if resp.TotalProjected1Y == 0 && years >= 1 {
			resp.TotalProjected1Y = last.P50
			resp.Projected1YRange = &models.ProjectedRange{P10: last.P10, P50: last.P50, P90: last.P90}
		}
	}

	return resp, nil
}

func projectGrowth(name, category string, currentVal, annualRate, annualVolatility float64, months int, now time.Time) models.InvestmentProjection {
	monthlyRate := annualRate / 100 / 12
	monthlyVol := annualVolatility / 100 / math.Sqrt(12)

	// Run Monte Carlo simulations
	allPaths := make([][]float64, monteCarloSimulations)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for s := 0; s < monteCarloSimulations; s++ {
		path := make([]float64, months+1)
		path[0] = currentVal
		val := currentVal
		for m := 1; m <= months; m++ {
			// Geometric Brownian Motion: drift + random shock
			shock := rng.NormFloat64() * monthlyVol
			val *= (1 + monthlyRate + shock)
			if val < 0 {
				val = 0
			}
			path[m] = val
		}
		allPaths[s] = path
	}

	// Extract percentiles at each month
	points := make([]models.ProjectionPoint, months+1)
	for m := 0; m <= months; m++ {
		vals := make([]float64, monteCarloSimulations)
		for s := 0; s < monteCarloSimulations; s++ {
			vals[s] = allPaths[s][m]
		}
		sort.Float64s(vals)
		t := now.AddDate(0, m, 0)
		points[m] = models.ProjectionPoint{
			Year:  t.Year(),
			Month: int(t.Month()),
			Value: math.Round(vals[len(vals)/2]),      // median (P50)
			P10:   math.Round(vals[len(vals)*10/100]), // pessimistic
			P50:   math.Round(vals[len(vals)*50/100]), // expected
			P90:   math.Round(vals[len(vals)*90/100]), // optimistic
		}
	}

	return models.InvestmentProjection{
		Name:           name,
		Category:       category,
		CurrentValue:   currentVal,
		AssumedRatePct: annualRate,
		VolatilityPct:  annualVolatility,
		Projections:    points,
	}
}

// projectGrowthWithSIP is like projectGrowth but adds a monthly recurring contribution (SIP, EPF, NPS).
// When monthlySIP is 0, it behaves identically to projectGrowth.
func projectGrowthWithSIP(name, category string, currentVal, annualRate, annualVolatility float64, months int, now time.Time, monthlySIP float64) models.InvestmentProjection {
	if monthlySIP <= 0 {
		return projectGrowth(name, category, currentVal, annualRate, annualVolatility, months, now)
	}

	monthlyRate := annualRate / 100 / 12
	monthlyVol := annualVolatility / 100 / math.Sqrt(12)

	allPaths := make([][]float64, monteCarloSimulations)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for s := 0; s < monteCarloSimulations; s++ {
		path := make([]float64, months+1)
		path[0] = currentVal
		val := currentVal
		for m := 1; m <= months; m++ {
			// Add monthly contribution first, then apply growth + shock
			val += monthlySIP
			shock := rng.NormFloat64() * monthlyVol
			val *= (1 + monthlyRate + shock)
			if val < 0 {
				val = 0
			}
			path[m] = val
		}
		allPaths[s] = path
	}

	points := make([]models.ProjectionPoint, months+1)
	for m := 0; m <= months; m++ {
		vals := make([]float64, monteCarloSimulations)
		for s := 0; s < monteCarloSimulations; s++ {
			vals[s] = allPaths[s][m]
		}
		sort.Float64s(vals)
		t := now.AddDate(0, m, 0)
		points[m] = models.ProjectionPoint{
			Year:  t.Year(),
			Month: int(t.Month()),
			Value: math.Round(vals[len(vals)/2]),
			P10:   math.Round(vals[len(vals)*10/100]),
			P50:   math.Round(vals[len(vals)*50/100]),
			P90:   math.Round(vals[len(vals)*90/100]),
		}
	}

	return models.InvestmentProjection{
		Name:           name,
		Category:       category,
		CurrentValue:   currentVal,
		AssumedRatePct: annualRate,
		VolatilityPct:  annualVolatility,
		Projections:    points,
	}
}
func projectFD(fd models.FixedDeposit, months int, now time.Time) models.InvestmentProjection {
	points := make([]models.ProjectionPoint, 0, months+1)
	monthlyRate := fd.InterestRate / 100 / 12
	var startVal float64
	if fd.InterestType == "cumulative" {
		startVal = fd.PrincipalAmount
	} else {
		startVal = fd.PrincipalAmount
	}

	points = append(points, models.ProjectionPoint{
		Year: now.Year(), Month: int(now.Month()), Value: math.Round(startVal),
		P10: math.Round(startVal), P50: math.Round(startVal), P90: math.Round(startVal),
	})

	val := startVal
	maturity := fd.MaturityDate
	for m := 1; m <= months; m++ {
		t := now.AddDate(0, m, 0)
		if fd.InterestType == "cumulative" && t.Before(maturity) {
			val *= (1 + monthlyRate)
		} else if fd.InterestType == "cumulative" && !t.Before(maturity) {
			val = fd.MaturityAmount
		}
		rounded := math.Round(val)
		// FDs are deterministic — P10/P50/P90 are all the same
		points = append(points, models.ProjectionPoint{
			Year: t.Year(), Month: int(t.Month()), Value: rounded,
			P10: rounded, P50: rounded, P90: rounded,
		})
	}

	name := fd.BankName
	if fd.FDNumber != "" {
		name += " - " + fd.FDNumber
	}

	return models.InvestmentProjection{
		Name:           name,
		Category:       "fixed_deposit",
		CurrentValue:   startVal,
		AssumedRatePct: fd.InterestRate,
		VolatilityPct:  0,
		Projections:    points,
	}
}

func estimateMFVolatility(mf *models.MutualFund) float64 {
	// Estimate annual volatility from transaction-implied returns
	// If we have enough transactions with varying values, compute std dev
	if len(mf.Transactions) >= 4 {
		returns := computeTransactionReturns(mf.Transactions, func(t models.MFTransaction) (time.Time, float64) {
			return t.Date, t.NAVAtPurchase
		})
		if vol := annualizedVolatility(returns); vol > 0 {
			return vol
		}
	}
	// Fallback by fund type
	switch mf.FundType {
	case "Equity", "ELSS":
		return 18.0
	case "Index":
		return 15.0
	case "Debt", "Liquid":
		return 3.0
	case "Hybrid":
		return 10.0
	default:
		return 12.0
	}
}

func estimateStockVolatility(stock *models.Stock) float64 {
	if len(stock.Transactions) >= 4 {
		returns := computeTransactionReturns(stock.Transactions, func(t models.StockTransaction) (time.Time, float64) {
			return t.Date, t.PricePerShare
		})
		if vol := annualizedVolatility(returns); vol > 0 {
			return vol
		}
	}
	return 22.0 // Default stock volatility
}

// computeTransactionReturns calculates period-over-period returns from transaction prices
func computeTransactionReturns[T any](txns []T, extract func(T) (time.Time, float64)) []float64 {
	type pricePoint struct {
		date  time.Time
		price float64
	}
	points := make([]pricePoint, 0, len(txns))
	for _, t := range txns {
		d, p := extract(t)
		if p > 0 {
			points = append(points, pricePoint{date: d, price: p})
		}
	}
	if len(points) < 2 {
		return nil
	}
	sort.Slice(points, func(i, j int) bool { return points[i].date.Before(points[j].date) })

	returns := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		ret := (points[i].price - points[i-1].price) / points[i-1].price
		returns = append(returns, ret)
	}
	return returns
}

// annualizedVolatility computes annualized volatility from periodic returns
func annualizedVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1)
	stdDev := math.Sqrt(variance)

	// Annualize: assume monthly-ish intervals — scale by sqrt(12)
	annual := stdDev * math.Sqrt(12) * 100
	if annual > 60 {
		annual = 60
	}
	return math.Round(annual*10) / 10
}

func estimateMFRate(mf *models.MutualFund) float64 {
	if mf.TotalInvested > 0 && mf.CurrentValue > 0 && len(mf.Transactions) > 0 {
		// Simple CAGR based on earliest transaction date
		earliest := mf.Transactions[0].Date
		for _, t := range mf.Transactions {
			if t.Date.Before(earliest) {
				earliest = t.Date
			}
		}
		years := time.Since(earliest).Hours() / 8766
		if years >= 0.5 {
			cagr := (math.Pow(mf.CurrentValue/mf.TotalInvested, 1/years) - 1) * 100
			if cagr > 0 && cagr < 50 {
				return math.Round(cagr*10) / 10
			}
		}
	}
	// Fallback by fund type
	switch mf.FundType {
	case "Equity", "ELSS", "Index":
		return 12.0
	case "Debt", "Liquid":
		return 7.0
	case "Hybrid":
		return 10.0
	default:
		return 10.0
	}
}

func estimateStockCAGR(stock *models.Stock) float64 {
	if stock.TotalInvested <= 0 || stock.CurrentValue <= 0 || len(stock.Transactions) == 0 {
		return 12.0
	}
	earliest := stock.Transactions[0].Date
	for _, t := range stock.Transactions {
		if t.Date.Before(earliest) {
			earliest = t.Date
		}
	}
	years := time.Since(earliest).Hours() / 8766
	if years >= 0.25 {
		cagr := (math.Pow(stock.CurrentValue/stock.TotalInvested, 1/years) - 1) * 100
		if cagr > 0 && cagr < 60 {
			return math.Round(cagr*10) / 10
		}
	}
	return 12.0
}

func aggregateProjections(investments []models.InvestmentProjection, months int, now time.Time) []models.ProjectionPoint {
	if len(investments) == 0 {
		return nil
	}

	points := make([]models.ProjectionPoint, months+1)
	for m := 0; m <= months; m++ {
		t := now.AddDate(0, m, 0)
		points[m] = models.ProjectionPoint{
			Year: t.Year(), Month: int(t.Month()), Value: 0, P10: 0, P50: 0, P90: 0,
		}
	}

	for _, inv := range investments {
		for i, pt := range inv.Projections {
			if i < len(points) {
				points[i].Value += pt.Value
				points[i].P10 += pt.P10
				points[i].P50 += pt.P50
				points[i].P90 += pt.P90
			}
		}
	}

	// Round aggregates
	for i := range points {
		points[i].Value = math.Round(points[i].Value)
		points[i].P10 = math.Round(points[i].P10)
		points[i].P50 = math.Round(points[i].P50)
		points[i].P90 = math.Round(points[i].P90)
	}

	return points
}
