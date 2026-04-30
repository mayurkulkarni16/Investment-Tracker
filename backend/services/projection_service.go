package services

import (
	"context"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type ProjectionService struct {
	mfRepo    *repository.MutualFundRepo
	fdRepo    *repository.FixedDepositRepo
	pfRepo    *repository.ProvidentFundRepo
	stockRepo *repository.StockRepo
	bondRepo  *repository.CorporateBondRepo
}

func NewProjectionService(
	mfRepo *repository.MutualFundRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	stockRepo *repository.StockRepo,
	bondRepo *repository.CorporateBondRepo,
) *ProjectionService {
	return &ProjectionService{
		mfRepo:    mfRepo,
		fdRepo:    fdRepo,
		pfRepo:    pfRepo,
		stockRepo: stockRepo,
		bondRepo:  bondRepo,
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
	funds, err := s.mfRepo.GetAll(ctx, userID)
	if err == nil {
		for _, mf := range funds {
			if mf.CurrentValue <= 0 {
				continue
			}
			rate := estimateMFRate(&mf) * rateMultiplier
			proj := projectGrowth(mf.FundName, "mutual_fund", mf.CurrentValue, rate, totalMonths, now)
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

	// Provident Fund — use current PF rate (default 8.25%)
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
			proj := projectGrowth(pf.EmployerName+" - "+string(pf.AccountType), "provident_fund", pf.CurrentBalance, rate, totalMonths, now)
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
			proj := projectGrowth(stock.StockName, "stock", stock.CurrentValue, rate, totalMonths, now)
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
			proj := projectGrowth(bond.BondName, "corporate_bond", bond.RemainingPrincipal, bond.CouponRate*rateMultiplier, totalMonths, now)
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
				resp.TotalProjected1Y = pt.Value
			}
			if monthsFromNow == 36 {
				resp.TotalProjected3Y = pt.Value
			}
			if monthsFromNow == 60 {
				resp.TotalProjected5Y = pt.Value
			}
		}
		// If projection period is shorter, use last available
		last := resp.AggregateMonthly[len(resp.AggregateMonthly)-1]
		if resp.TotalProjected5Y == 0 && years >= 5 {
			resp.TotalProjected5Y = last.Value
		}
		if resp.TotalProjected3Y == 0 && years >= 3 {
			resp.TotalProjected3Y = last.Value
		}
		if resp.TotalProjected1Y == 0 && years >= 1 {
			resp.TotalProjected1Y = last.Value
		}
	}

	return resp, nil
}

func projectGrowth(name, category string, currentVal, annualRate float64, months int, now time.Time) models.InvestmentProjection {
	monthlyRate := annualRate / 100 / 12
	points := make([]models.ProjectionPoint, 0, months+1)

	// Add current point
	points = append(points, models.ProjectionPoint{
		Year: now.Year(), Month: int(now.Month()), Value: math.Round(currentVal),
	})

	val := currentVal
	for m := 1; m <= months; m++ {
		val *= (1 + monthlyRate)
		t := now.AddDate(0, m, 0)
		points = append(points, models.ProjectionPoint{
			Year: t.Year(), Month: int(t.Month()), Value: math.Round(val),
		})
	}

	return models.InvestmentProjection{
		Name:           name,
		Category:       category,
		CurrentValue:   currentVal,
		AssumedRatePct: annualRate,
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
	})

	val := startVal
	maturity := fd.MaturityDate
	for m := 1; m <= months; m++ {
		t := now.AddDate(0, m, 0)
		if fd.InterestType == "cumulative" && t.Before(maturity) {
			val *= (1 + monthlyRate)
		} else if fd.InterestType == "cumulative" && !t.Before(maturity) {
			// After maturity, value stays at maturity amount
			val = fd.MaturityAmount
		}
		// For non-cumulative, principal stays same
		points = append(points, models.ProjectionPoint{
			Year: t.Year(), Month: int(t.Month()), Value: math.Round(val),
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
		Projections:    points,
	}
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
			Year: t.Year(), Month: int(t.Month()), Value: 0,
		}
	}

	for _, inv := range investments {
		for i, pt := range inv.Projections {
			if i < len(points) {
				points[i].Value += pt.Value
			}
		}
	}

	// Round aggregates
	for i := range points {
		points[i].Value = math.Round(points[i].Value)
	}

	return points
}
