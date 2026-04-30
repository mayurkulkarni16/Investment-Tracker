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

type InsightsService struct {
	mfService        *MutualFundService
	stockService     *StockService
	bondService      *CorporateBondService
	fdService        *FixedDepositService
	pfService        *ProvidentFundService
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	npsService       *NPSService
	goalService      *GoalService
}

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
) *InsightsService {
	return &InsightsService{
		mfService: mfService, stockService: stockService, bondService: bondService,
		fdService: fdService, pfService: pfService, homeLoanRepo: homeLoanRepo,
		personalLoanRepo: personalLoanRepo, npsService: npsService, goalService: goalService,
	}
}

func (s *InsightsService) GetInsights(ctx context.Context, userID string) (*InsightsResponse, error) {
	var insights []Insight

	funds, _ := s.mfService.GetAll(ctx, userID)
	stocks, _ := s.stockService.GetAll(ctx, userID)
	bonds, _ := s.bondService.GetAll(ctx, userID)
	fds, _ := s.fdService.GetAll(ctx, userID)
	pfs, _ := s.pfService.GetAll(ctx, userID)
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	personalLoans, _ := s.personalLoanRepo.GetAll(ctx, userID)
	npsAccounts, _ := s.npsService.GetAll(ctx, userID)
	goals, _ := s.goalService.GetAll(ctx, userID)

	// Compute totals
	var totalEquity, totalDebt, totalAssets, totalLiabilities float64
	for _, f := range funds {
		switch f.FundType {
		case "Equity", "ELSS", "Index":
			totalEquity += f.CurrentValue
		case "Debt", "Liquid":
			totalDebt += f.CurrentValue
		case "Hybrid":
			totalEquity += f.CurrentValue * 0.65
			totalDebt += f.CurrentValue * 0.35
		default:
			totalEquity += f.CurrentValue
		}
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
	for _, fd := range fds {
		if fd.Status == "active" {
			totalDebt += fd.PrincipalAmount
			totalAssets += fd.PrincipalAmount
		}
	}
	for _, pf := range pfs {
		totalDebt += pf.CurrentBalance
		totalAssets += pf.CurrentBalance
	}
	for _, nps := range npsAccounts {
		totalAssets += nps.CurrentValue
		totalEquity += nps.CurrentValue * nps.EquityPct / 100
		totalDebt += nps.CurrentValue * (100 - nps.EquityPct) / 100
	}
	for _, hl := range homeLoans {
		totalLiabilities += hl.OutstandingPrincipal
	}
	for _, pl := range personalLoans {
		totalLiabilities += pl.OutstandingPrincipal
	}

	id := 0
	nextID := func() string {
		id++
		return fmt.Sprintf("INS-%03d", id)
	}

	// ── Rule 1: High equity concentration ──
	if totalAssets > 0 {
		eqPct := totalEquity / totalAssets * 100
		if eqPct > 80 {
			insights = append(insights, Insight{
				ID: nextID(), Title: "High Equity Concentration",
				Description: fmt.Sprintf("%.0f%% of your portfolio is in equity. Consider adding debt instruments for stability.", eqPct),
				Severity:    SeverityWarning, Category: CatRisk,
				Action: "Consider moving 10-20% into FDs, bonds, or debt mutual funds.",
			})
		} else if eqPct < 20 && totalEquity > 0 {
			insights = append(insights, Insight{
				ID: nextID(), Title: "Low Equity Exposure",
				Description: fmt.Sprintf("Only %.0f%% in equity. You may be missing growth potential.", eqPct),
				Severity:    SeverityInfo, Category: CatDiversify,
				Action: "Consider allocating more to equity mutual funds or index funds for long-term growth.",
			})
		}
	}

	// ── Rule 2: Emergency fund check ──
	liquidAssets := 0.0
	for _, fd := range fds {
		if fd.Status == "active" {
			liquidAssets += fd.PrincipalAmount
		}
	}
	for _, f := range funds {
		if f.FundType == "Liquid" {
			liquidAssets += f.CurrentValue
		}
	}
	monthlyEMI := 0.0
	for _, hl := range homeLoans {
		if hl.Status == "active" {
			monthlyEMI += hl.EMIAmount
		}
	}
	for _, pl := range personalLoans {
		if pl.Status == "active" {
			monthlyEMI += pl.EMIAmount
		}
	}
	if monthlyEMI > 0 && liquidAssets < monthlyEMI*6 {
		insights = append(insights, Insight{
			ID: nextID(), Title: "Insufficient Emergency Fund",
			Description: fmt.Sprintf("Your liquid assets (%s) cover less than 6 months of EMI obligations (%s/month).",
				formatINR(liquidAssets), formatINR(monthlyEMI)),
			Severity: SeverityCritical, Category: CatRisk,
			Action: "Build up an emergency fund of at least 6 months of expenses in liquid funds or FDs.",
		})
	}

	// ── Rule 3: Underperforming mutual funds ──
	for _, f := range funds {
		if f.TotalInvested > 10000 && f.XIRR < 4 && f.XIRR > -50 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Underperforming Fund: %s", f.FundName),
				Description: fmt.Sprintf("XIRR of %.1f%% is below FD rates. Invested: %s.", f.XIRR, formatINR(f.TotalInvested)),
				Severity:    SeverityWarning, Category: CatPerformance,
				Action: "Review if this fund aligns with your goals. Consider switching to a better-performing fund.",
			})
		}
	}

	// ── Rule 4: Stocks with significant losses ──
	for _, st := range stocks {
		if st.TotalInvested > 5000 && st.GainLossPercent < -20 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Significant Loss: %s", st.Symbol),
				Description: fmt.Sprintf("%s is down %.1f%%. Loss: %s.", st.Symbol, st.GainLossPercent, formatINR(-st.GainLoss)),
				Severity:    SeverityWarning, Category: CatPerformance,
				Action: "Evaluate if fundamentals have changed. Consider tax-loss harvesting or averaging down.",
			})
		}
	}

	// ── Rule 5: FD maturing soon ──
	thirtyDays := time.Now().AddDate(0, 0, 30)
	for _, fd := range fds {
		if fd.Status == "active" && fd.MaturityDate.Before(thirtyDays) {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("FD Maturing Soon: %s", fd.BankName),
				Description: fmt.Sprintf("FD of %s at %s matures on %s.", formatINR(fd.PrincipalAmount), fd.BankName, fd.MaturityDate.Format("02 Jan 2006")),
				Severity:    SeverityInfo, Category: CatOpportunity,
				Action: "Plan reinvestment — compare current FD rates or consider debt mutual funds.",
			})
		}
	}

	// ── Rule 6: Bond interest pending ──
	for _, bond := range bonds {
		if bond.Status != "active" {
			continue
		}
		for _, p := range bond.InterestPayouts {
			if p.Status == "pending" && p.ScheduledDate.Before(time.Now()) {
				insights = append(insights, Insight{
					ID: nextID(), Title: fmt.Sprintf("Overdue Interest: %s", bond.BondName),
					Description: fmt.Sprintf("Interest payout of %s was due on %s but not received.", formatINR(p.Amount), p.ScheduledDate.Format("02 Jan 2006")),
					Severity:    SeverityCritical, Category: CatRisk,
					Action: "Follow up with the issuer or your broker immediately.",
				})
				break
			}
		}
	}

	// ── Rule 7: High loan-to-asset ratio ──
	if totalAssets > 0 {
		loanRatio := totalLiabilities / totalAssets * 100
		if loanRatio > 50 {
			insights = append(insights, Insight{
				ID: nextID(), Title: "High Leverage",
				Description: fmt.Sprintf("Your liabilities are %.0f%% of assets. Total loans: %s vs assets: %s.", loanRatio, formatINR(totalLiabilities), formatINR(totalAssets)),
				Severity:    SeverityCritical, Category: CatLoan,
				Action: "Prioritize loan repayment. Consider prepaying high-interest loans first.",
			})
		} else if loanRatio > 30 {
			insights = append(insights, Insight{
				ID: nextID(), Title: "Moderate Leverage",
				Description: fmt.Sprintf("Liabilities are %.0f%% of assets. Stay watchful.", loanRatio),
				Severity:    SeverityWarning, Category: CatLoan,
			})
		}
	}

	// ── Rule 8: Loan vs investment rate comparison ──
	for _, hl := range homeLoans {
		if hl.Status != "active" {
			continue
		}
		// Compare loan rate vs best FD rate
		bestFDRate := 0.0
		for _, fd := range fds {
			if fd.Status == "active" && fd.InterestRate > bestFDRate {
				bestFDRate = fd.InterestRate
			}
		}
		if bestFDRate > 0 && hl.InterestRate > bestFDRate+1 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Loan Rate Higher Than FD Returns: %s", hl.BankName),
				Description: fmt.Sprintf("Home loan at %.2f%% while your best FD earns %.2f%%. You're paying more in interest than you earn.", hl.InterestRate, bestFDRate),
				Severity:    SeverityWarning, Category: CatLoan,
				Action: "Consider prepaying this loan instead of opening new FDs — the effective return is higher.",
			})
		}
	}
	for _, pl := range personalLoans {
		if pl.Status != "active" {
			continue
		}
		avgMFReturn := avgXIRR(funds)
		if avgMFReturn > 0 && pl.InterestRate > avgMFReturn*1.5 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("High-Cost Loan: %s", pl.LenderName),
				Description: fmt.Sprintf("Personal loan at %.1f%% is significantly higher than your average MF return of %.1f%%.", pl.InterestRate, avgMFReturn),
				Severity:    SeverityCritical, Category: CatLoan,
				Action: "Prioritize prepaying this loan before making new investments.",
			})
		}
	}

	// ── Rule 9: ELSS tax saving opportunity ──
	elssInvested := 0.0
	for _, f := range funds {
		if f.IsELSS {
			elssInvested += f.TotalInvested
		}
	}
	if elssInvested < 150000 {
		gap := 150000 - elssInvested
		insights = append(insights, Insight{
			ID: nextID(), Title: "ELSS Tax Saving Opportunity",
			Description: fmt.Sprintf("You've invested %s in ELSS out of ₹1.5L limit under 80C. Potential tax saving of %s.", formatINR(elssInvested), formatINR(gap*0.3)),
			Severity:    SeverityInfo, Category: CatTax,
			Action: fmt.Sprintf("Invest %s more in ELSS funds to maximize Section 80C deduction.", formatINR(gap)),
		})
	}

	// ── Rule 10: NPS tax saving opportunity ──
	nps80ccd1b := 0.0
	for _, nps := range npsAccounts {
		nps80ccd1b += nps.Section80CCD1B
	}
	if nps80ccd1b < 50000 {
		gap := 50000 - nps80ccd1b
		insights = append(insights, Insight{
			ID: nextID(), Title: "NPS Additional Tax Benefit Available",
			Description: fmt.Sprintf("You can claim %s more under Section 80CCD(1B) — additional ₹50K deduction beyond 80C.", formatINR(gap)),
			Severity:    SeverityInfo, Category: CatTax,
			Action: "Contribute to NPS Tier 1 to get additional tax deduction of up to ₹50,000.",
		})
	}

	// ── Rule 11: No provident fund ──
	if len(pfs) == 0 {
		insights = append(insights, Insight{
			ID: nextID(), Title: "No Provident Fund",
			Description: "You don't have any EPF/PPF records. PF provides guaranteed returns with tax benefits.",
			Severity:    SeverityInfo, Category: CatOpportunity,
			Action: "Consider opening a PPF account for safe, tax-free long-term returns.",
		})
	}

	// ── Rule 12: Single stock concentration ──
	totalStockValue := 0.0
	for _, st := range stocks {
		totalStockValue += st.CurrentValue
	}
	for _, st := range stocks {
		if totalStockValue > 0 && st.CurrentValue/totalStockValue > 0.40 && len(stocks) > 1 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Stock Concentration: %s", st.Symbol),
				Description: fmt.Sprintf("%s is %.0f%% of your stock portfolio. High single-stock risk.", st.Symbol, st.CurrentValue/totalStockValue*100),
				Severity:    SeverityWarning, Category: CatRisk,
				Action: "Consider diversifying — reduce position or add other stocks/ETFs.",
			})
		}
	}

	// ── Rule 13: Goals off track ──
	for _, g := range goals {
		if !g.TargetDate.IsZero() && g.TargetDate.After(time.Now()) && g.TargetAmount > 0 {
			// Recompute progress
			currentVal := g.CurrentValue
			remaining := g.TargetAmount - currentVal
			monthsLeft := int(time.Until(g.TargetDate).Hours() / (24 * 30))
			if monthsLeft > 0 && remaining > 0 {
				monthlyNeeded := remaining / float64(monthsLeft)
				if monthlyNeeded > g.TargetAmount/float64(monthsLeft)*1.5 {
					insights = append(insights, Insight{
						ID: nextID(), Title: fmt.Sprintf("Goal Behind Schedule: %s", g.Name),
						Description: fmt.Sprintf("Need %s/month for next %d months. Currently at %.0f%% of target.",
							formatINR(monthlyNeeded), monthsLeft, currentVal/g.TargetAmount*100),
						Severity: SeverityWarning, Category: CatGoal,
						Action: "Increase SIP amounts or review linked investments for this goal.",
					})
				}
			}
		}
	}

	// ── Rule 14: Stale prices ──
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	for _, st := range stocks {
		if st.PriceLastUpdated != nil && st.PriceLastUpdated.Before(sevenDaysAgo) && st.TotalQuantity > 0 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Stale Price: %s", st.Symbol),
				Description: fmt.Sprintf("Stock price last updated %s. Values may be inaccurate.", st.PriceLastUpdated.Format("02 Jan 2006")),
				Severity:    SeverityInfo, Category: CatPerformance,
				Action: "Refresh stock prices to get accurate portfolio valuation.",
			})
		}
	}
	for _, f := range funds {
		if f.NAVLastUpdated != nil && f.NAVLastUpdated.Before(sevenDaysAgo) && f.TotalUnits > 0 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Stale NAV: %s", f.FundName),
				Description: fmt.Sprintf("NAV last updated %s.", f.NAVLastUpdated.Format("02 Jan 2006")),
				Severity:    SeverityInfo, Category: CatPerformance,
				Action: "Refresh NAV for accurate valuation.",
			})
		}
	}

	// ── Rule 15: Strong performers ──
	for _, f := range funds {
		if f.TotalInvested > 10000 && f.XIRR > 15 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Top Performer: %s", f.FundName),
				Description: fmt.Sprintf("XIRR of %.1f%% — significantly above average.", f.XIRR),
				Severity:    SeverityPositive, Category: CatPerformance,
			})
		}
	}
	for _, st := range stocks {
		if st.TotalInvested > 5000 && st.GainLossPercent > 50 {
			insights = append(insights, Insight{
				ID: nextID(), Title: fmt.Sprintf("Star Stock: %s", st.Symbol),
				Description: fmt.Sprintf("Up %.1f%% (Gain: %s). Consider booking partial profits.", st.GainLossPercent, formatINR(st.GainLoss)),
				Severity:    SeverityPositive, Category: CatPerformance,
				Action: "Consider booking partial profits to lock in gains.",
			})
		}
	}

	// Sort: critical first, then warning, info, positive
	severityOrder := map[InsightSeverity]int{
		SeverityCritical: 0, SeverityWarning: 1, SeverityInfo: 2, SeverityPositive: 3,
	}
	sort.Slice(insights, func(i, j int) bool {
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
