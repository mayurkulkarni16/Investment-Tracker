package services

import (
	"context"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type GoalService struct {
	repo      *repository.GoalRepo
	mfRepo    *repository.MutualFundRepo
	stockRepo *repository.StockRepo
	fdRepo    *repository.FixedDepositRepo
	pfRepo    *repository.ProvidentFundRepo
	npsRepo   *repository.NPSRepo
	bondRepo  *repository.CorporateBondRepo
}

func NewGoalService(
	repo *repository.GoalRepo,
	mfRepo *repository.MutualFundRepo,
	stockRepo *repository.StockRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	npsRepo *repository.NPSRepo,
	bondRepo *repository.CorporateBondRepo,
) *GoalService {
	return &GoalService{repo: repo, mfRepo: mfRepo, stockRepo: stockRepo, fdRepo: fdRepo, pfRepo: pfRepo, npsRepo: npsRepo, bondRepo: bondRepo}
}

func (s *GoalService) Create(ctx context.Context, req models.CreateGoalRequest) (*models.Goal, error) {
	targetDate, err := parseDate(req.TargetDate)
	if err != nil {
		return nil, err
	}

	assumedReturn := req.AssumedReturnRate
	if assumedReturn == 0 {
		assumedReturn = 12
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
		Name:              req.Name,
		Category:          req.Category,
		Icon:              icon,
		Priority:          priority,
		TargetAmount:      req.TargetAmount,
		TargetDate:        targetDate,
		AssumedReturnRate: assumedReturn,
		LinkedInvestments: []models.LinkedInvestment{},
		Status:            "active",
		Notes:             req.Notes,
	}

	if err := s.repo.Create(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal)
	return goal, nil
}

func (s *GoalService) GetAll(ctx context.Context) ([]models.Goal, error) {
	goals, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range goals {
		s.computeDerived(ctx, &goals[i])
	}
	return goals, nil
}

func (s *GoalService) GetByID(ctx context.Context, id string) (*models.Goal, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	goal, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal)
	return goal, nil
}

func (s *GoalService) Update(ctx context.Context, id string, req models.UpdateGoalRequest) (*models.Goal, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	goal, err := s.repo.GetByID(ctx, objID)
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
	if req.AssumedReturnRate > 0 {
		goal.AssumedReturnRate = req.AssumedReturnRate
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
	s.computeDerived(ctx, goal)
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
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	goal, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	link := models.LinkedInvestment{
		InvestmentType: req.InvestmentType,
		InvestmentID:   req.InvestmentID,
		InvestmentName: req.InvestmentName,
		AllocatedPct:   req.AllocatedPct,
	}

	goal.LinkedInvestments = append(goal.LinkedInvestments, link)

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal)
	return goal, nil
}

func (s *GoalService) BatchLinkInvestments(ctx context.Context, id string, req models.BatchLinkInvestmentRequest) (*models.Goal, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	goal, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	// Build set of already-linked IDs to avoid duplicates
	linked := make(map[string]bool)
	for _, li := range goal.LinkedInvestments {
		linked[li.InvestmentID] = true
	}

	for _, inv := range req.Investments {
		if linked[inv.InvestmentID] {
			continue
		}
		goal.LinkedInvestments = append(goal.LinkedInvestments, models.LinkedInvestment{
			InvestmentType: inv.InvestmentType,
			InvestmentID:   inv.InvestmentID,
			InvestmentName: inv.InvestmentName,
			AllocatedPct:   inv.AllocatedPct,
		})
		linked[inv.InvestmentID] = true
	}

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, err
	}
	s.computeDerived(ctx, goal)
	return goal, nil
}

func (s *GoalService) UnlinkInvestment(ctx context.Context, id string, investmentID string) (*models.Goal, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	goal, err := s.repo.GetByID(ctx, objID)
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
	s.computeDerived(ctx, goal)
	return goal, nil
}

func (s *GoalService) computeDerived(ctx context.Context, goal *models.Goal) {
	currentValue := 0.0

	for _, li := range goal.LinkedInvestments {
		val := s.getInvestmentValue(ctx, li)
		currentValue += val * li.AllocatedPct / 100
	}

	goal.CurrentValue = math.Round(currentValue*100) / 100
	if goal.TargetAmount > 0 {
		goal.ProgressPct = math.Round(currentValue/goal.TargetAmount*10000) / 100
	}

	now := time.Now()
	months := int(goal.TargetDate.Sub(now).Hours() / 24 / 30)
	if months < 0 {
		months = 0
	}
	goal.MonthsRemaining = months

	// Monthly SIP needed: FV = P * ((1+r)^n - 1)/r * (1+r)
	remaining := goal.TargetAmount - currentValue
	if remaining > 0 && months > 0 {
		monthlyRate := goal.AssumedReturnRate / 100 / 12
		if monthlyRate > 0 {
			n := float64(months)
			factor := (math.Pow(1+monthlyRate, n) - 1) / monthlyRate * (1 + monthlyRate)
			goal.MonthlyNeeded = math.Round(remaining/factor*100) / 100
		} else {
			goal.MonthlyNeeded = math.Round(remaining/float64(months)*100) / 100
		}
	}

	goal.OnTrack = goal.ProgressPct >= float64(100-goal.MonthsRemaining*100/max(1, goal.MonthsRemaining+len(goal.LinkedInvestments)))
	if goal.ProgressPct >= 100 {
		goal.OnTrack = true
	}

	// Projection points: project monthly growth for up to target date or 60 months
	maxMonths := months
	if maxMonths <= 0 {
		maxMonths = 12
	}
	if maxMonths > 60 {
		maxMonths = 60
	}
	monthlyRate := goal.AssumedReturnRate / 100 / 12
	projValue := currentValue
	goal.ProjectionPoints = make([]models.GoalProjection, 0, maxMonths+1)
	goal.ProjectionPoints = append(goal.ProjectionPoints, models.GoalProjection{Month: 0, Value: math.Round(projValue)})
	projectedDateFound := false
	for m := 1; m <= maxMonths; m++ {
		projValue = projValue*(1+monthlyRate) + goal.MonthlyNeeded
		goal.ProjectionPoints = append(goal.ProjectionPoints, models.GoalProjection{Month: m, Value: math.Round(projValue)})
		if !projectedDateFound && projValue >= goal.TargetAmount {
			pd := now.AddDate(0, m, 0)
			goal.ProjectedDate = &pd
			projectedDateFound = true
		}
	}

	// Shortfall: projected value at target date vs target
	if months > 0 {
		fv := currentValue
		for m := 0; m < months; m++ {
			fv = fv*(1+monthlyRate) + goal.MonthlyNeeded
		}
		goal.Shortfall = math.Round((goal.TargetAmount-fv)*100) / 100
		if goal.Shortfall < 0 {
			goal.Shortfall = 0
		}
	}
}

func (s *GoalService) getInvestmentValue(ctx context.Context, li models.LinkedInvestment) float64 {
	switch li.InvestmentType {
	case "mutual_fund":
		id, err := parseObjectID(li.InvestmentID)
		if err != nil {
			return 0
		}
		mf, err := s.mfRepo.GetByID(ctx, id)
		if err != nil {
			return 0
		}
		return mf.CurrentValue
	case "stock":
		id, err := parseObjectID(li.InvestmentID)
		if err != nil {
			return 0
		}
		stock, err := s.stockRepo.GetByID(ctx, id)
		if err != nil {
			return 0
		}
		return stock.CurrentValue
	case "fixed_deposit", "fd":
		id, err := parseObjectID(li.InvestmentID)
		if err != nil {
			return 0
		}
		fd, err := s.fdRepo.GetByID(ctx, id)
		if err != nil {
			return 0
		}
		return fd.MaturityAmount
	case "provident_fund", "pf":
		id, err := parseObjectID(li.InvestmentID)
		if err != nil {
			return 0
		}
		pf, err := s.pfRepo.GetByID(ctx, id)
		if err != nil {
			return 0
		}
		return pf.CurrentBalance
	case "nps":
		id, err := parseObjectID(li.InvestmentID)
		if err != nil {
			return 0
		}
		nps, err := s.npsRepo.GetByID(ctx, id)
		if err != nil {
			return 0
		}
		return nps.CurrentValue
	case "corporate_bond", "bond":
		id, err := parseObjectID(li.InvestmentID)
		if err != nil {
			return 0
		}
		bond, err := s.bondRepo.GetByID(ctx, id)
		if err != nil {
			return 0
		}
		return bond.RemainingPrincipal
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
