package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

// ErrGoalAllocation is returned when linking would allocate more than 100% of
// an investment across goals.
var ErrGoalAllocation = errors.New("allocation exceeds 100% of the investment across goals")

// defaultGoalReturn is used when a goal has no explicit rate and no linked investments.
const defaultGoalReturn = 10.0

type GoalService struct {
	repo      *repository.GoalRepo
	mfRepo    *repository.MutualFundRepo
	stockRepo *repository.StockRepo
	fdRepo    *repository.FixedDepositRepo
	pfRepo    *repository.ProvidentFundRepo
	npsRepo   *repository.NPSRepo
	bondRepo  *repository.CorporateBondRepo
	sipRepo   *repository.SIPRepo
}

func NewGoalService(
	repo *repository.GoalRepo,
	mfRepo *repository.MutualFundRepo,
	stockRepo *repository.StockRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	npsRepo *repository.NPSRepo,
	bondRepo *repository.CorporateBondRepo,
	sipRepo *repository.SIPRepo,
) *GoalService {
	return &GoalService{repo: repo, mfRepo: mfRepo, stockRepo: stockRepo, fdRepo: fdRepo, pfRepo: pfRepo, npsRepo: npsRepo, bondRepo: bondRepo, sipRepo: sipRepo}
}

func (s *GoalService) Create(ctx context.Context, userID string, req models.CreateGoalRequest) (*models.Goal, error) {
	targetDate, err := parseDate(req.TargetDate)
	if err != nil {
		return nil, err
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	icon := req.Icon
	if icon == "" {
		icon = "🎯"
	}

	goal := &models.Goal{
		UserID:            userID,
		Name:              req.Name,
		Category:          req.Category,
		Icon:              icon,
		Priority:          priority,
		TargetAmount:      req.TargetAmount,
		TargetDate:        targetDate,
		AssumedReturnRate: req.AssumedReturnRate,
		InflationRate:     req.InflationRate,
		LinkedInvestments: []models.LinkedInvestment{},
		Status:            "active",
		Notes:             req.Notes,
	}

	if err := s.repo.Create(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal, nil)
	return goal, nil
}

func (s *GoalService) GetAll(ctx context.Context, userID string) ([]models.Goal, error) {
	goals, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	sips := s.sipContributions(ctx, userID)
	for i := range goals {
		s.computeDerived(ctx, &goals[i], sips)
	}
	return goals, nil
}

func (s *GoalService) GetByID(ctx context.Context, id string) (*models.Goal, error) {
	goal, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal, nil)
	return goal, nil
}

func (s *GoalService) Update(ctx context.Context, id string, req models.UpdateGoalRequest) (*models.Goal, error) {
	goal, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		goal.Name = req.Name
	}
	if req.Category != "" {
		goal.Category = req.Category
	}
	if req.Icon != "" {
		goal.Icon = req.Icon
	}
	if req.Priority != "" {
		goal.Priority = req.Priority
	}
	if req.TargetAmount > 0 {
		goal.TargetAmount = req.TargetAmount
	}
	if req.TargetDate != "" {
		td, err := parseDate(req.TargetDate)
		if err != nil {
			return nil, err
		}
		goal.TargetDate = td
	}
	if req.AssumedReturnRate != nil {
		goal.AssumedReturnRate = *req.AssumedReturnRate
	}
	if req.InflationRate != nil {
		goal.InflationRate = *req.InflationRate
	}
	if req.Status != "" {
		goal.Status = req.Status
	}
	if req.Notes != "" {
		goal.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal, nil)
	return goal, nil
}

func (s *GoalService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *GoalService) LinkInvestment(ctx context.Context, id string, req models.LinkInvestmentRequest) (*models.Goal, error) {
	return s.BatchLinkInvestments(ctx, id, models.BatchLinkInvestmentRequest{Investments: []models.LinkInvestmentRequest{req}})
}

// BatchLinkInvestments links investments to a goal. Re-linking an investment
// already on this goal updates its allocation. The total allocation of any
// investment across all goals may not exceed 100%.
func (s *GoalService) BatchLinkInvestments(ctx context.Context, id string, req models.BatchLinkInvestmentRequest) (*models.Goal, error) {
	goal, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}

	allocatedElsewhere := map[string]float64{}
	if others, err := s.repo.GetAll(ctx, goal.UserID); err == nil {
		for _, g := range others {
			if g.ID == goal.ID || g.Status == "abandoned" || g.Status == "cancelled" {
				continue
			}
			for _, li := range g.LinkedInvestments {
				allocatedElsewhere[li.InvestmentID] += li.AllocatedPct
			}
		}
	}

	for _, inv := range req.Investments {
		if inv.AllocatedPct <= 0 || inv.AllocatedPct > 100 {
			return nil, fmt.Errorf("%w: allocation for %s must be between 0 and 100%%", ErrGoalAllocation, inv.InvestmentName)
		}
		if used := allocatedElsewhere[inv.InvestmentID]; used+inv.AllocatedPct > 100.0001 {
			return nil, fmt.Errorf("%w: %.0f%% of %s is already allocated to other goals; at most %.0f%% is available",
				ErrGoalAllocation, used, inv.InvestmentName, math.Max(0, 100-used))
		}
		link := models.LinkedInvestment{
			InvestmentType: inv.InvestmentType,
			InvestmentID:   inv.InvestmentID,
			InvestmentName: inv.InvestmentName,
			AllocatedPct:   inv.AllocatedPct,
		}
		replaced := false
		for i := range goal.LinkedInvestments {
			if goal.LinkedInvestments[i].InvestmentID == inv.InvestmentID {
				goal.LinkedInvestments[i] = link
				replaced = true
				break
			}
		}
		if !replaced {
			goal.LinkedInvestments = append(goal.LinkedInvestments, link)
		}
	}

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal, nil)
	return goal, nil
}

func (s *GoalService) UnlinkInvestment(ctx context.Context, id string, investmentID string) (*models.Goal, error) {
	goal, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}

	filtered := []models.LinkedInvestment{}
	for _, li := range goal.LinkedInvestments {
		if li.InvestmentID != investmentID {
			filtered = append(filtered, li)
		}
	}
	goal.LinkedInvestments = filtered

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal, nil)
	return goal, nil
}

func (s *GoalService) load(ctx context.Context, id string) (*models.Goal, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, objID)
}

// sipContributions maps fund ID -> current monthly SIP amount for active SIPs.
func (s *GoalService) sipContributions(ctx context.Context, userID string) map[string]float64 {
	out := map[string]float64{}
	if s.sipRepo == nil {
		return out
	}
	sips, err := s.sipRepo.GetAll(ctx, userID)
	if err != nil {
		return out
	}
	now := time.Now()
	for _, sip := range sips {
		if sip.Status != "active" || sip.FundID == "" || (sip.EndDate != nil && sip.EndDate.Before(now)) {
			continue
		}
		out[sip.FundID] += monthlyEquivalent(sip.Amount, sip.Frequency)
	}
	return out
}

// goalHolding is the current state of one linked investment.
type goalHolding struct {
	value, monthly, rate float64
}

func (s *GoalService) holding(ctx context.Context, li models.LinkedInvestment, sips map[string]float64, now time.Time) (goalHolding, bool) {
	id, err := parseObjectID(li.InvestmentID)
	if err != nil {
		return goalHolding{}, false
	}
	switch li.InvestmentType {
	case "mutual_fund":
		mf, err := s.mfRepo.GetByID(ctx, id)
		if err != nil {
			return goalHolding{}, false
		}
		return goalHolding{value: mf.CurrentValue, monthly: sips[li.InvestmentID], rate: MutualFundProfile(mf.FundType).ExpectedReturn}, true
	case "stock":
		st, err := s.stockRepo.GetByID(ctx, id)
		if err != nil {
			return goalHolding{}, false
		}
		return goalHolding{value: st.CurrentValue, rate: StockProfile.ExpectedReturn}, true
	case "fixed_deposit", "fd":
		fd, err := s.fdRepo.GetByID(ctx, id)
		if err != nil {
			return goalHolding{}, false
		}
		NormalizeFixedDeposit(fd, now)
		return goalHolding{value: FixedDepositCurrentValue(fd, now), rate: fd.InterestRate}, true
	case "provident_fund", "pf":
		pf, err := s.pfRepo.GetByID(ctx, id)
		if err != nil {
			return goalHolding{}, false
		}
		rate := pf.InterestRate
		if rate <= 0 {
			rate = 8.25
		}
		monthly := 0.0
		if n := len(pf.FinancialYearEntries); n > 0 {
			latest := pf.FinancialYearEntries[n-1]
			if len(latest.MonthlyContributions) > 0 {
				total := 0.0
				for _, mc := range latest.MonthlyContributions {
					total += mc.EmployeeContribution + mc.EmployerContribution
				}
				monthly = total / float64(len(latest.MonthlyContributions))
			}
		}
		return goalHolding{value: pf.CurrentBalance, monthly: monthly, rate: rate}, true
	case "nps":
		nps, err := s.npsRepo.GetByID(ctx, id)
		if err != nil {
			return goalHolding{}, false
		}
		recent := 0.0
		yearAgo := now.AddDate(-1, 0, 0)
		for _, c := range nps.Contributions {
			if !c.Date.Before(yearAgo) {
				recent += c.Amount
			}
		}
		return goalHolding{value: nps.CurrentValue, monthly: recent / 12, rate: NPSProfile(nps).ExpectedReturn}, true
	case "corporate_bond", "bond":
		bond, err := s.bondRepo.GetByID(ctx, id)
		if err != nil {
			return goalHolding{}, false
		}
		NormalizeCorporateBond(bond)
		return goalHolding{value: bond.RemainingPrincipal, rate: bond.CouponRate}, true
	}
	return goalHolding{}, false
}

// computeDerived fills in progress and projections. The goal is "on track" when
// the current linked value plus ongoing contributions (active SIPs, PF/NPS
// contributions), grown at the effective return, reaches the inflation-adjusted
// target by the target date.
func (s *GoalService) computeDerived(ctx context.Context, goal *models.Goal, sips map[string]float64) {
	now := time.Now()
	if sips == nil {
		sips = s.sipContributions(ctx, goal.UserID)
	}

	var current, monthly, rateNum, rateDen float64
	for _, li := range goal.LinkedInvestments {
		h, ok := s.holding(ctx, li, sips, now)
		if !ok {
			continue
		}
		share := li.AllocatedPct / 100
		current += h.value * share
		monthly += h.monthly * share
		w := (h.value + h.monthly*12) * share
		rateNum += h.rate * w
		rateDen += w
	}

	rate := goal.AssumedReturnRate
	if rate <= 0 {
		rate = defaultGoalReturn
		if rateDen > 0 {
			rate = rateNum / rateDen
		}
	}

	months := monthsBetween(now, goal.TargetDate)
	if goal.TargetDate.Day() < now.Day() {
		months--
	}
	if months < 0 {
		months = 0
	}

	plan := planGoal(goalInputs{
		target: goal.TargetAmount, inflation: goal.InflationRate, rate: rate,
		current: current, monthly: monthly, months: months,
	})

	goal.CurrentValue = round2(current)
	goal.MonthlyContribution = round2(monthly)
	goal.EffectiveReturnRate = round1(rate)
	goal.MonthsRemaining = months
	goal.InflatedTarget = math.Round(plan.inflatedTarget)
	goal.ProgressPct = plan.progressPct
	goal.ProjectedValue = math.Round(plan.projectedValue)
	goal.MonthlyNeeded = math.Round(plan.monthlyNeeded)
	goal.AdditionalMonthlyNeeded = math.Round(plan.additionalNeeded)
	goal.Shortfall = math.Round(plan.shortfall)
	goal.OnTrack = plan.onTrack
	goal.ProjectedDate = nil
	if plan.reachMonth >= 0 {
		d := now.AddDate(0, plan.reachMonth, 0)
		goal.ProjectedDate = &d
	}
	goal.ProjectionPoints = plan.points
}

type goalInputs struct {
	target, inflation, rate float64
	current, monthly        float64
	months                  int
}

type goalPlan struct {
	inflatedTarget, progressPct, projectedValue float64
	monthlyNeeded, additionalNeeded, shortfall  float64
	onTrack                                     bool
	reachMonth                                  int // -1 if not reached within 50 years
	points                                      []models.GoalProjection
}

// planGoal is the pure goal-planning math (contributions at the start of each
// month, monthly geometric growth).
func planGoal(in goalInputs) goalPlan {
	r := math.Pow(1+in.rate/100, 1.0/12) - 1
	inflAt := func(m int) float64 { return in.target * math.Pow(1+in.inflation/100, float64(m)/12) }
	grow := func(m int) float64 { return math.Pow(1+r, float64(m)) }
	annuityDue := func(m int) float64 {
		if r == 0 {
			return float64(m)
		}
		return (grow(m) - 1) / r * (1 + r)
	}
	trajectory := func(m int) float64 { return in.current*grow(m) + in.monthly*annuityDue(m) }

	p := goalPlan{inflatedTarget: inflAt(in.months), reachMonth: -1}
	if p.inflatedTarget > 0 {
		p.progressPct = math.Round(in.current/p.inflatedTarget*10000) / 100
	}
	p.projectedValue = trajectory(in.months)

	if in.months > 0 {
		need := (p.inflatedTarget - in.current*grow(in.months)) / annuityDue(in.months)
		p.monthlyNeeded = math.Max(0, need)
		p.additionalNeeded = math.Max(0, p.monthlyNeeded-in.monthly)
	}
	p.shortfall = math.Max(0, p.inflatedTarget-p.projectedValue)
	p.onTrack = in.current >= p.inflatedTarget || p.projectedValue >= p.inflatedTarget*0.995

	for m := 0; m <= 600; m++ {
		if trajectory(m) >= inflAt(m) {
			p.reachMonth = m
			break
		}
	}

	horizon := in.months
	if horizon < 12 {
		horizon = 12
	}
	if horizon > 360 {
		horizon = 360
	}
	p.points = make([]models.GoalProjection, 0, horizon+1)
	for m := 0; m <= horizon; m++ {
		p.points = append(p.points, models.GoalProjection{Month: m, Value: math.Round(trajectory(m))})
	}
	return p
}
