package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type InsightSeverity string

const (
	SeverityCritical InsightSeverity = "critical"
	SeverityWarning  InsightSeverity = "warning"
	SeverityInfo     InsightSeverity = "info"
	SeverityPositive InsightSeverity = "positive"
)

type InsightCategory string

const (
	CatRisk        InsightCategory = "risk"
	CatPerformance InsightCategory = "performance"
	CatTax         InsightCategory = "tax"
	CatDiversify   InsightCategory = "diversification"
	CatLoan        InsightCategory = "loan"
	CatOpportunity InsightCategory = "opportunity"
	CatGoal        InsightCategory = "goal"
)

type Insight struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Severity    InsightSeverity `json:"severity"`
	Category    InsightCategory `json:"category"`
	Action      string          `json:"action,omitempty"`
}

type InsightsResponse struct {
	Insights      []Insight `json:"insights"`
	CriticalCount int       `json:"critical_count"`
	WarningCount  int       `json:"warning_count"`
	InfoCount     int       `json:"info_count"`
	PositiveCount int       `json:"positive_count"`
}

// Thresholds used by the rules.
const (
	minHoldingYearsForXIRR   = 1.0
	underperformXIRR         = 4.0
	highCostLoanRate         = 12.0
	elevatedLoanRate         = 9.0
	alternativeConcentration = 10.0 // % of assets in unregulated/alternative schemes
	taxSaverFDTenureMonths   = 60   // 5-year tax-saver FDs are locked in
	oldRegimeTopSlab         = 0.312
)

type InsightsService struct {
	mfService         *MutualFundService
	stockService      *StockService
	bondService       *CorporateBondService
	fdService         *FixedDepositService
	pfService         *ProvidentFundService
	homeLoanRepo      *repository.HomeLoanRepo
	personalLoanRepo  *repository.PersonalLoanRepo
	npsService        *NPSService
	goalService       *GoalService
	taxService        *TaxService
	vehicleRentalRepo *repository.VehicleRentalRepo
	familyTax         *FamilyTaxService
}

// SetFamilyTax enables member-specific tax insights.
func (s *InsightsService) SetFamilyTax(f *FamilyTaxService) { s.familyTax = f }

func NewInsightsService(
	mfService *MutualFundService,
	stockService *StockService,
	bondService *CorporateBondService,
	fdService *FixedDepositService,
	pfService *ProvidentFundService,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	npsService *NPSService,
	goalService *GoalService,
	taxService *TaxService,
	vehicleRentalRepo *repository.VehicleRentalRepo,
) *InsightsService {
	return &InsightsService{
		mfService: mfService, stockService: stockService, bondService: bondService,
		fdService: fdService, pfService: pfService, homeLoanRepo: homeLoanRepo,
		personalLoanRepo: personalLoanRepo, npsService: npsService, goalService: goalService,
		taxService: taxService, vehicleRentalRepo: vehicleRentalRepo,
	}
}

func (s *InsightsService) GetInsights(ctx context.Context, userID string) (*InsightsResponse, error) {
	now := time.Now()
	insights := []Insight{}
	// IDs are derived from the rule and the entity so they stay stable across requests.
	add := func(id string, in Insight) {
		in.ID = id
		insights = append(insights, in)
	}

	funds, _ := s.mfService.GetAll(ctx, userID)
	stocks, _ := s.stockService.GetAll(ctx, userID)
	bonds, _ := s.bondService.GetAll(ctx, userID)
	fds, _ := s.fdService.GetAll(ctx, userID)
	pfs, _ := s.pfService.GetAll(ctx, userID)
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	personalLoans, _ := s.personalLoanRepo.GetAll(ctx, userID)
	npsAccounts, _ := s.npsService.GetAll(ctx, userID)
	goals, _ := s.goalService.GetAll(ctx, userID)
	var rentals []models.VehicleRental
	if s.vehicleRentalRepo != nil {
		rentals, _ = s.vehicleRentalRepo.GetAll(ctx, userID)
		for i := range rentals {
			ComputeVehicleRental(&rentals[i], now)
		}
	}

	// Totals
	var totalEquity, totalDebt, totalAlternative, totalAssets, totalLiabilities float64
	for _, f := range funds {
		share := MutualFundProfile(f.FundType).EquityShare
		totalEquity += f.CurrentValue * share
		totalDebt += f.CurrentValue * (1 - share)
		totalAssets += f.CurrentValue
	}
	for _, st := range stocks {
		totalEquity += st.CurrentValue
		totalAssets += st.CurrentValue
	}
	for _, b := range bonds {
		totalDebt += b.RemainingPrincipal
		totalAssets += b.RemainingPrincipal
	}
	for i := range fds {
		if v := FixedDepositCurrentValue(&fds[i], now); v > 0 {
			totalDebt += v
			totalAssets += v
		}
	}
	for _, pf := range pfs {
		totalDebt += pf.CurrentBalance
		totalAssets += pf.CurrentBalance
	}
	for _, nps := range npsAccounts {
		share := NPSProfile(&nps).EquityShare
		totalAssets += nps.CurrentValue
		totalEquity += nps.CurrentValue * share
		totalDebt += nps.CurrentValue * (1 - share)
	}
	for _, v := range rentals {
		totalAlternative += v.CurrentValue
		totalAssets += v.CurrentValue
	}
	for _, hl := range homeLoans {
		if hl.Status == models.LoanActive {
			totalLiabilities += hl.OutstandingPrincipal
		}
	}
	for _, pl := range personalLoans {
		if pl.Status == models.LoanActive {
			totalLiabilities += pl.OutstandingPrincipal
		}
	}

	// ── Asset allocation ──
	if marketAssets := totalEquity + totalDebt; marketAssets > 0 {
		eqPct := totalEquity / marketAssets * 100
		if eqPct > 80 {
			add("allocation-high-equity", Insight{
				Title:       "High Equity Concentration",
				Description: fmt.Sprintf("%.0f%% of your market-linked portfolio is in equity. Consider adding debt instruments for stability.", eqPct),
				Severity:    SeverityWarning, Category: CatRisk,
				Action: "Consider moving 10-20% into FDs, bonds, or debt mutual funds, depending on your time horizon.",
			})
		} else if eqPct < 20 && totalEquity > 0 {
			add("allocation-low-equity", Insight{
				Title:       "Low Equity Exposure",
				Description: fmt.Sprintf("Only %.0f%% of your market-linked portfolio is in equity. You may be missing long-term growth.", eqPct),
				Severity:    SeverityInfo, Category: CatDiversify,
				Action: "For goals 7+ years away, consider diversified equity or index funds.",
			})
		}
	}

	// ── Vehicle rentals / alternative schemes ──
	for _, v := range rentals {
		if v.OverdueCount > 0 {
			add("rental-overdue:"+v.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Rental Payouts Overdue: %s", v.Provider),
				Description: fmt.Sprintf("%d payout(s) totalling %s have not been received. Missed payouts are often the first sign of trouble in fixed-return schemes.", v.OverdueCount, formatINR(v.OverdueAmount)),
				Severity:    SeverityCritical, Category: CatRisk,
				Action: "Follow up in writing, keep the agreement and payment records, and avoid adding more money until payouts resume.",
			})
		}
	}
	if totalAssets > 0 && totalAlternative > 0 {
		if pct := totalAlternative / totalAssets * 100; pct > alternativeConcentration {
			add("alternative-concentration", Insight{
				Title:       "High Exposure to Unregulated Schemes",
				Description: fmt.Sprintf("%.0f%% of your assets (%s) are in vehicle rental schemes, which aren't regulated by SEBI or RBI and depend on a single operator.", pct, formatINR(totalAlternative)),
				Severity:    SeverityWarning, Category: CatRisk,
				Action: fmt.Sprintf("Consider keeping such schemes under %.0f%% of your portfolio.", alternativeConcentration),
			})
		}
	}

	// ── Emergency fund (cash-like funds + FDs that aren't tax-saver lock-ins) ──
	liquidAssets := 0.0
	for i := range fds {
		if fds[i].TenureMonths < taxSaverFDTenureMonths {
			liquidAssets += FixedDepositCurrentValue(&fds[i], now)
		}
	}
	for _, f := range funds {
		if IsCashLikeFund(f.FundType) {
			liquidAssets += f.CurrentValue
		}
	}
	monthlyEMI := 0.0
	for _, hl := range homeLoans {
		if hl.Status == models.LoanActive {
			monthlyEMI += hl.EMIAmount
		}
	}
	for _, pl := range personalLoans {
		if pl.Status == models.LoanActive {
			monthlyEMI += pl.EMIAmount
		}
	}
	if monthlyEMI > 0 {
		coverage := liquidAssets / monthlyEMI
		if coverage < 6 {
			sev := SeverityWarning
			if coverage < 3 {
				sev = SeverityCritical
			}
			add("emergency-fund", Insight{
				Title: "Emergency Fund Below 6 Months of EMIs",
				Description: fmt.Sprintf("Liquid assets (%s in liquid/overnight funds and non-tax-saver FDs) cover %.1f months of EMIs (%s/month).",
					formatINR(liquidAssets), coverage, formatINR(monthlyEMI)),
				Severity: sev, Category: CatRisk,
				Action: "Build an emergency fund of at least 6 months of EMIs plus living expenses in liquid funds or sweep FDs.",
			})
		}
	}

	// ── Underperforming funds (only once held long enough for XIRR to mean something) ──
	for i := range funds {
		f := &funds[i]
		first, ok := earliestMFTransaction(f)
		if !ok || f.TotalInvested <= 10000 || yearsSince(first, now) < minHoldingYearsForXIRR {
			continue
		}
		if f.XIRR < underperformXIRR && f.XIRR > -50 {
			add("mf-underperforming:"+f.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Underperforming Fund: %s", f.FundName),
				Description: fmt.Sprintf("XIRR of %.1f%% over %.1f years is below FD rates. Invested: %s.", f.XIRR, yearsSince(first, now), formatINR(f.TotalInvested)),
				Severity:    SeverityWarning, Category: CatPerformance,
				Action: "Compare it with its category index over the same period before deciding to switch.",
			})
		}
	}

	// ── Stocks with significant losses ──
	for _, st := range stocks {
		if st.TotalInvested > 5000 && st.GainLossPercent < -20 {
			add("stock-loss:"+st.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Significant Loss: %s", st.Symbol),
				Description: fmt.Sprintf("%s is down %.1f%%. Loss: %s.", st.Symbol, st.GainLossPercent, formatINR(-st.GainLoss)),
				Severity:    SeverityWarning, Category: CatPerformance,
				Action: "Re-check whether the reason you bought it still holds. Booking the loss can offset capital gains (tax-loss harvesting).",
			})
		}
	}

	// ── FD maturing soon ──
	thirtyDays := now.AddDate(0, 0, 30)
	for _, fd := range fds {
		if fd.Status == "active" && !fd.MaturityDate.Before(now) && fd.MaturityDate.Before(thirtyDays) {
			add("fd-maturing:"+fd.ID.Hex(), Insight{
				Title:       fmt.Sprintf("FD Maturing Soon: %s", fd.BankName),
				Description: fmt.Sprintf("FD of %s at %s matures on %s.", formatINR(fd.PrincipalAmount), fd.BankName, fd.MaturityDate.Format("02 Jan 2006")),
				Severity:    SeverityInfo, Category: CatOpportunity,
				Action: "Plan reinvestment — compare current FD rates or consider debt mutual funds.",
			})
		}
	}

	// ── Leverage ──
	if totalAssets > 0 {
		loanRatio := totalLiabilities / totalAssets * 100
		if loanRatio > 50 {
			add("leverage-high", Insight{
				Title:       "High Leverage",
				Description: fmt.Sprintf("Your liabilities are %.0f%% of assets. Total loans: %s vs assets: %s.", loanRatio, formatINR(totalLiabilities), formatINR(totalAssets)),
				Severity:    SeverityCritical, Category: CatLoan,
				Action: "Prioritize loan repayment. Consider prepaying high-interest loans first.",
			})
		} else if loanRatio > 30 {
			add("leverage-moderate", Insight{
				Title:       "Moderate Leverage",
				Description: fmt.Sprintf("Liabilities are %.0f%% of assets. Stay watchful.", loanRatio),
				Severity:    SeverityWarning, Category: CatLoan,
			})
		}
	}

	// ── Loan cost vs returns ──
	bestFDRate := 0.0
	for _, fd := range fds {
		if fd.Status == "active" && fd.InterestRate > bestFDRate {
			bestFDRate = fd.InterestRate
		}
	}
	for _, hl := range homeLoans {
		if hl.Status != models.LoanActive || bestFDRate <= 0 || hl.InterestRate <= bestFDRate+1 {
			continue
		}
		add("homeloan-vs-fd:"+hl.ID.Hex(), Insight{
			Title:       fmt.Sprintf("Loan Rate Higher Than FD Returns: %s", hl.BankName),
			Description: fmt.Sprintf("Home loan at %.2f%% while your best FD earns %.2f%% (and FD interest is taxable).", hl.InterestRate, bestFDRate),
			Severity:    SeverityWarning, Category: CatLoan,
			Action: "Prepaying usually beats a new FD. Under the old regime, factor in the Section 24(b) interest deduction (up to ₹2L) before deciding.",
		})
	}
	avgMFReturn := avgXIRR(funds)
	for _, pl := range personalLoans {
		if pl.Status != models.LoanActive || pl.InterestRate < elevatedLoanRate {
			continue
		}
		sev := SeverityWarning
		if pl.InterestRate >= highCostLoanRate {
			sev = SeverityCritical
		}
		desc := fmt.Sprintf("Personal loan at %.1f%%. Few investments reliably beat that after tax.", pl.InterestRate)
		if avgMFReturn > 0 {
			desc = fmt.Sprintf("Personal loan at %.1f%% vs your average mutual fund return of %.1f%%.", pl.InterestRate, avgMFReturn)
		}
		add("personal-loan-cost:"+pl.ID.Hex(), Insight{
			Title:       fmt.Sprintf("High-Cost Loan: %s", pl.LenderName),
			Description: desc,
			Severity:    sev, Category: CatLoan,
			Action: "Prioritize prepaying this loan before making new investments.",
		})
	}

	// ── Tax: member-specific regime / deduction / placement suggestions when
	// income data exists, else generic 80C room and NPS 80CCD(1B) tips ──
	familyTaxUsed := false
	if s.familyTax != nil {
		if rep, err := s.familyTax.Report(ctx, userID, ""); err == nil {
			hasIncome := false
			for _, m := range rep.Members {
				hasIncome = hasIncome || m.HasIncomeRecord
			}
			if hasIncome {
				familyTaxUsed = true
				shown := 0
				for _, sg := range rep.Suggestions {
					if sg.EstSaving < 1000 || shown >= 4 {
						continue
					}
					shown++
					add("family-tax:"+sg.ID, Insight{
						Title:       sg.Title,
						Description: fmt.Sprintf("FY %s · %s", rep.FY, sg.Detail),
						Severity:    SeverityInfo, Category: CatTax,
						Action: fmt.Sprintf("Estimated saving %s. See Tax Center → Family for details.", formatINR(sg.EstSaving)),
					})
				}
			}
		}
	}
	if s.taxService != nil && totalAssets > 0 && !familyTaxUsed {
		if tax, err := s.taxService.GetTaxSummary(ctx, userID, ""); err == nil {
			c := tax.Section80C
			if room := c.Limit - c.Total; room > 0 {
				add("tax-80c-room", Insight{
					Title: "Section 80C Room Left This Year",
					Description: fmt.Sprintf("FY %s: %s of the %s limit used (EPF %s, ELSS %s, home-loan principal %s). %s remaining.",
						tax.FY, formatINR(c.Total), formatINR(c.Limit), formatINR(c.EPFContribution), formatINR(c.ELSSInvestment), formatINR(c.HomeLoanPrincipal), formatINR(room)),
					Severity: SeverityInfo, Category: CatTax,
					Action: fmt.Sprintf("Only relevant under the old tax regime: investing %s more (ELSS, PPF, etc.) saves up to %s in the 30%% slab. 80C isn't available under the new regime.",
						formatINR(room), formatINR(room*oldRegimeTopSlab)),
				})
			} else if gap := 50000 - tax.Section80CCD.NPS80CCD1B; gap > 0 {
				add("tax-nps-80ccd1b", Insight{
					Title:       "NPS Additional Deduction Available",
					Description: fmt.Sprintf("Your 80C limit is used up. Section 80CCD(1B) allows %s more for NPS Tier 1 contributions this year.", formatINR(gap)),
					Severity:    SeverityInfo, Category: CatTax,
					Action: "Only relevant under the old tax regime. NPS is locked in until 60, so weigh liquidity before contributing.",
				})
			}
		}
	}

	// ── No provident fund (only once there is a portfolio to speak of) ──
	if len(pfs) == 0 && totalAssets > 0 {
		add("no-provident-fund", Insight{
			Title:       "No Provident Fund",
			Description: "You don't have any EPF/PPF records. PF provides guaranteed, tax-efficient returns.",
			Severity:    SeverityInfo, Category: CatOpportunity,
			Action: "If you're salaried, add your EPF account; otherwise consider a PPF account for safe long-term savings.",
		})
	}

	// ── Single stock concentration ──
	totalStockValue := 0.0
	for _, st := range stocks {
		totalStockValue += st.CurrentValue
	}
	if len(stocks) > 1 && totalStockValue > 0 {
		for _, st := range stocks {
			if share := st.CurrentValue / totalStockValue; share > 0.40 {
				add("stock-concentration:"+st.ID.Hex(), Insight{
					Title:       fmt.Sprintf("Stock Concentration: %s", st.Symbol),
					Description: fmt.Sprintf("%s is %.0f%% of your stock portfolio. High single-stock risk.", st.Symbol, share*100),
					Severity:    SeverityWarning, Category: CatRisk,
					Action: "Consider diversifying — reduce the position or add other stocks/ETFs.",
				})
			}
		}
	}

	// ── Goals off track (current value + actual contributions vs inflation-adjusted target) ──
	for _, g := range goals {
		if g.Status != "active" || g.OnTrack || g.TargetAmount <= 0 || !g.TargetDate.After(now) {
			continue
		}
		desc := fmt.Sprintf("At the current pace (%s/month, %.1f%% return) you'll reach %s of the %s needed by %s.",
			formatINR(g.MonthlyContribution), g.EffectiveReturnRate, formatINR(g.ProjectedValue), formatINR(g.InflatedTarget), g.TargetDate.Format("Jan 2006"))
		if len(g.LinkedInvestments) == 0 {
			desc = "No investments are linked to this goal yet, so nothing is counting towards it."
		}
		add("goal-behind:"+g.ID.Hex(), Insight{
			Title:       fmt.Sprintf("Goal Behind Schedule: %s", g.Name),
			Description: desc,
			Severity:    SeverityWarning, Category: CatGoal,
			Action: fmt.Sprintf("Invest about %s/month more, link more investments, or move the target date.", formatINR(g.AdditionalMonthlyNeeded)),
		})
	}

	// ── Stale prices ──
	sevenDaysAgo := now.AddDate(0, 0, -7)
	for _, st := range stocks {
		if st.PriceLastUpdated != nil && st.PriceLastUpdated.Before(sevenDaysAgo) && st.TotalQuantity > 0 {
			add("stale-price:"+st.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Stale Price: %s", st.Symbol),
				Description: fmt.Sprintf("Stock price last updated %s. Values may be inaccurate.", st.PriceLastUpdated.Format("02 Jan 2006")),
				Severity:    SeverityInfo, Category: CatPerformance,
				Action: "Refresh stock prices to get accurate portfolio valuation.",
			})
		}
	}
	for _, f := range funds {
		if f.NAVLastUpdated != nil && f.NAVLastUpdated.Before(sevenDaysAgo) && f.TotalUnits > 0 {
			add("stale-nav:"+f.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Stale NAV: %s", f.FundName),
				Description: fmt.Sprintf("NAV last updated %s.", f.NAVLastUpdated.Format("02 Jan 2006")),
				Severity:    SeverityInfo, Category: CatPerformance,
				Action: "Refresh NAV for accurate valuation.",
			})
		}
	}

	// ── Strong performers ──
	for i := range funds {
		f := &funds[i]
		first, ok := earliestMFTransaction(f)
		if ok && f.TotalInvested > 10000 && f.XIRR > 15 && yearsSince(first, now) >= minHoldingYearsForXIRR {
			add("mf-top:"+f.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Top Performer: %s", f.FundName),
				Description: fmt.Sprintf("XIRR of %.1f%% — significantly above average.", f.XIRR),
				Severity:    SeverityPositive, Category: CatPerformance,
			})
		}
	}
	for _, st := range stocks {
		if st.TotalInvested > 5000 && st.GainLossPercent > 50 {
			add("stock-star:"+st.ID.Hex(), Insight{
				Title:       fmt.Sprintf("Star Stock: %s", st.Symbol),
				Description: fmt.Sprintf("Up %.1f%% (Gain: %s).", st.GainLossPercent, formatINR(st.GainLoss)),
				Severity:    SeverityPositive, Category: CatPerformance,
				Action: "If it has grown into a large share of your portfolio, consider rebalancing.",
			})
		}
	}

	severityOrder := map[InsightSeverity]int{
		SeverityCritical: 0, SeverityWarning: 1, SeverityInfo: 2, SeverityPositive: 3,
	}
	sort.SliceStable(insights, func(i, j int) bool {
		return severityOrder[insights[i].Severity] < severityOrder[insights[j].Severity]
	})

	resp := &InsightsResponse{Insights: insights}
	for _, ins := range insights {
		switch ins.Severity {
		case SeverityCritical:
			resp.CriticalCount++
		case SeverityWarning:
			resp.WarningCount++
		case SeverityInfo:
			resp.InfoCount++
		case SeverityPositive:
			resp.PositiveCount++
		}
	}
	return resp, nil
}

func formatINR(v float64) string {
	if v < 0 {
		return fmt.Sprintf("-₹%s", formatINRPositive(-v))
	}
	return fmt.Sprintf("₹%s", formatINRPositive(v))
}

func formatINRPositive(v float64) string {
	if v >= 10000000 {
		return fmt.Sprintf("%.2f Cr", v/10000000)
	}
	if v >= 100000 {
		return fmt.Sprintf("%.2f L", v/100000)
	}
	return fmt.Sprintf("%.0f", math.Round(v))
}

func avgXIRR(funds []models.MutualFund) float64 {
	totalW := 0.0
	sumWX := 0.0
	for _, f := range funds {
		if f.TotalInvested > 0 {
			totalW += f.TotalInvested
			sumWX += f.XIRR * f.TotalInvested
		}
	}
	if totalW == 0 {
		return 0
	}
	return sumWX / totalW
}
