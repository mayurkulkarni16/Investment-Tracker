package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

var policyTypes = map[string]bool{"term": true, "health": true, "life": true, "ulip": true, "vehicle": true, "home": true, "personal_accident": true, "other": true}
var premiumFrequencies = map[string]int{"monthly": 1, "quarterly": 3, "half_yearly": 6, "yearly": 12, "single": 0}
var policyStatuses = map[string]bool{"active": true, "lapsed": true, "matured": true, "surrendered": true, "expired": true}

// recommendedHealthCover is a baseline family-floater cover for Indian metros.
const recommendedHealthCover = 1_000_000

type InsuranceService struct {
	repo             *repository.InsuranceRepo
	ledgerRepo       *repository.LedgerRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
}

func NewInsuranceService(repo *repository.InsuranceRepo, ledgerRepo *repository.LedgerRepo, homeLoanRepo *repository.HomeLoanRepo, personalLoanRepo *repository.PersonalLoanRepo) *InsuranceService {
	return &InsuranceService{repo: repo, ledgerRepo: ledgerRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo}
}

func (s *InsuranceService) build(userID string, req models.InsurancePolicyRequest, p *models.InsurancePolicy) error {
	req.PolicyType = strings.ToLower(strings.TrimSpace(req.PolicyType))
	if !policyTypes[req.PolicyType] {
		return invalidf("policy_type must be one of term, health, life, ulip, vehicle, home, personal_accident, other")
	}
	if _, ok := premiumFrequencies[req.PremiumFrequency]; !ok {
		return invalidf("premium_frequency must be monthly, quarterly, half_yearly, yearly or single")
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if !policyStatuses[req.Status] {
		return invalidf("invalid status")
	}
	switch req.HealthCoverFor {
	case "", "self_family", "parents":
	default:
		return invalidf("health_cover_for must be self_family or parents")
	}
	start, err := parseDate(req.StartDate)
	if err != nil {
		return err
	}
	var end, premEnd *time.Time
	if req.EndDate != "" {
		t, err := parseDate(req.EndDate)
		if err != nil {
			return err
		}
		if !t.After(start) {
			return invalidf("end_date must be after start_date")
		}
		end = &t
	}
	if req.PremiumEndDate != "" {
		t, err := parseDate(req.PremiumEndDate)
		if err != nil {
			return err
		}
		premEnd = &t
	}
	p.UserID = userID
	p.PolicyType, p.Insurer, p.PolicyName, p.PolicyNumber = req.PolicyType, strings.TrimSpace(req.Insurer), req.PolicyName, req.PolicyNumber
	p.InsuredPersons, p.SumAssured, p.PremiumAmount, p.PremiumFrequency = req.InsuredPersons, req.SumAssured, req.PremiumAmount, req.PremiumFrequency
	p.HealthCoverFor, p.InsuredSenior = req.HealthCoverFor, req.InsuredSenior
	p.StartDate, p.EndDate, p.PremiumEndDate = start, end, premEnd
	p.CurrentValue, p.Nominee, p.Status, p.Notes = req.CurrentValue, req.Nominee, req.Status, req.Notes
	return nil
}

func (s *InsuranceService) Create(ctx context.Context, userID string, req models.InsurancePolicyRequest) (*models.InsurancePolicy, error) {
	p := &models.InsurancePolicy{}
	if err := s.build(userID, req, p); err != nil {
		return nil, err
	}
	p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
	id, err := s.repo.Insert(ctx, p)
	if err != nil {
		return nil, err
	}
	p.ID = id
	ComputePolicy(p, time.Now())
	return p, nil
}

func (s *InsuranceService) GetAll(ctx context.Context, userID string) ([]models.InsurancePolicy, error) {
	policies, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range policies {
		ComputePolicy(&policies[i], now)
	}
	return policies, nil
}

func (s *InsuranceService) GetByID(ctx context.Context, id string) (*models.InsurancePolicy, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	ComputePolicy(p, time.Now())
	return p, nil
}

func (s *InsuranceService) Update(ctx context.Context, id string, req models.InsurancePolicyRequest) (*models.InsurancePolicy, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if err := s.build(p.UserID, req, p); err != nil {
		return nil, err
	}
	p.UpdatedAt = time.Now()
	if err := s.repo.Replace(ctx, oid, p); err != nil {
		return nil, err
	}
	ComputePolicy(p, time.Now())
	return p, nil
}

func (s *InsuranceService) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, oid)
}

// PremiumDates returns the premium due dates of p falling in [from, to).
func PremiumDates(p *models.InsurancePolicy, from, to time.Time) []time.Time {
	step := premiumFrequencies[p.PremiumFrequency]
	if p.PremiumAmount <= 0 {
		return nil
	}
	if step == 0 {
		if !p.StartDate.Before(from) && p.StartDate.Before(to) {
			return []time.Time{p.StartDate}
		}
		return nil
	}
	stop := to
	if p.PremiumEndDate != nil && p.PremiumEndDate.Before(stop) {
		stop = p.PremiumEndDate.AddDate(0, 0, 1)
	}
	if p.EndDate != nil && p.EndDate.Before(stop) {
		stop = *p.EndDate
	}
	var out []time.Time
	for i := 0; i < 1200; i++ {
		d := p.StartDate.AddDate(0, i*step, 0)
		if !d.Before(stop) {
			break
		}
		if !d.Before(from) {
			out = append(out, d)
		}
	}
	return out
}

func ComputePolicy(p *models.InsurancePolicy, now time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if step := premiumFrequencies[p.PremiumFrequency]; step > 0 {
		p.AnnualPremium = round2(p.PremiumAmount * 12 / float64(step))
	}
	if p.Status == "active" {
		if next := PremiumDates(p, today, today.AddDate(2, 0, 0)); len(next) > 0 {
			p.NextPremiumDate = &next[0]
		}
	}
	if p.EndDate != nil {
		d := int(math.Ceil(p.EndDate.Sub(today).Hours() / 24))
		p.DaysToExpiry = &d
	}
	switch p.PolicyType {
	case "term", "life", "ulip":
		p.TaxSection = "80C"
	case "health":
		p.TaxSection = "80D"
	}
}

// PremiumsInRange sums premiums due in [from, to) for 80C (life, capped at 10%
// of sum assured) and 80D (health).
func PremiumsInRange(policies []models.InsurancePolicy, from, to time.Time) (sec80C, sec80D float64) {
	for i := range policies {
		p := &policies[i]
		if p.Status == "lapsed" {
			continue
		}
		total := float64(len(PremiumDates(p, from, to))) * p.PremiumAmount
		switch p.PolicyType {
		case "term", "life", "ulip":
			if p.SumAssured > 0 {
				total = math.Min(total, 0.10*p.SumAssured)
			}
			sec80C += total
		case "health":
			sec80D += total
		}
	}
	return round2(sec80C), round2(sec80D)
}

// annualLedgerIncome sums confirmed ledger income over the last 12 months.
func annualLedgerIncome(entries []models.LedgerEntry, now time.Time) float64 {
	from := now.AddDate(-1, 0, 0).Format("2006-01-02")
	total := 0.0
	for _, e := range entries {
		if e.Status == "confirmed" && e.Kind == "income" && e.Date >= from {
			total += float64(e.AmountPaise) / 100
		}
	}
	return total
}

func (s *InsuranceService) Summary(ctx context.Context, userID string, annualIncome float64) (*models.InsuranceSummary, error) {
	policies, err := s.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sum := &models.InsuranceSummary{UpcomingPremiums: []models.UpcomingPremium{}, Suggestions: []string{}}

	hasTerm := false
	for i := range policies {
		p := &policies[i]
		if p.Status != "active" {
			continue
		}
		switch p.PolicyType {
		case "term", "life", "ulip":
			sum.TotalLifeCover += p.SumAssured
			if p.PolicyType == "term" {
				hasTerm = true
			}
		case "health":
			sum.TotalHealthCover += p.SumAssured
		}
		sum.AnnualPremium += p.AnnualPremium
		sum.CashValue += p.CurrentValue
		for _, d := range PremiumDates(p, today, today.AddDate(0, 0, 60)) {
			sum.UpcomingPremiums = append(sum.UpcomingPremiums, models.UpcomingPremium{
				PolicyID: p.ID.Hex(), PolicyName: policyLabel(p), Insurer: p.Insurer,
				Date: d.Format("2006-01-02"), Amount: p.PremiumAmount, DaysLeft: int(d.Sub(today).Hours() / 24),
			})
		}
	}
	sort.Slice(sum.UpcomingPremiums, func(i, j int) bool { return sum.UpcomingPremiums[i].Date < sum.UpcomingPremiums[j].Date })

	fyStart := getCurrentFYStart(now)
	sum.Premium80CThisFY, sum.Premium80DThisFY = PremiumsInRange(policies, fyStart, fyStart.AddDate(1, 0, 0))

	switch {
	case annualIncome > 0:
		sum.AnnualIncome, sum.AnnualIncomeSource = annualIncome, "query"
	default:
		entries, _ := s.ledgerRepo.GetAll(ctx, userID)
		if inc := annualLedgerIncome(entries, now); inc > 0 {
			sum.AnnualIncome, sum.AnnualIncomeSource = round2(inc), "ledger"
		} else {
			sum.AnnualIncomeSource = "none"
		}
	}

	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	for _, l := range homeLoans {
		if l.Status == "active" {
			sum.OutstandingLoans += l.OutstandingPrincipal
		}
	}
	personalLoans, _ := s.personalLoanRepo.GetAll(ctx, userID)
	for _, l := range personalLoans {
		if l.Status == "active" {
			sum.OutstandingLoans += l.OutstandingPrincipal
		}
	}

	sum.RecommendedLifeCover = round2(10*sum.AnnualIncome + sum.OutstandingLoans)
	sum.LifeCoverGap = round2(math.Max(0, sum.RecommendedLifeCover-sum.TotalLifeCover))
	sum.RecommendedHealthCover = recommendedHealthCover
	sum.HealthCoverGap = math.Max(0, recommendedHealthCover-sum.TotalHealthCover)

	if sum.AnnualIncome == 0 {
		sum.Suggestions = append(sum.Suggestions, "Add your income in Income & Expenses (or enter it here) to check whether your life cover is adequate.")
	}
	if sum.LifeCoverGap > 0 && sum.RecommendedLifeCover > 0 {
		sum.Suggestions = append(sum.Suggestions, fmt.Sprintf("Life cover is ₹%s short of the recommended 10× annual income plus loans; a term plan is the cheapest way to close the gap.", formatLakhs(sum.LifeCoverGap)))
	}
	if !hasTerm && sum.TotalLifeCover > 0 {
		sum.Suggestions = append(sum.Suggestions, "You have no term plan. Endowment/ULIP policies give far less cover per rupee of premium.")
	}
	if sum.HealthCoverGap > 0 {
		sum.Suggestions = append(sum.Suggestions, fmt.Sprintf("Health cover is below ₹10L; consider a top-up/super top-up for ₹%s more.", formatLakhs(sum.HealthCoverGap)))
	}
	if sum.Premium80DThisFY > 25000 {
		sum.Suggestions = append(sum.Suggestions, "Health premiums exceed the ₹25,000 80D limit for self/family; premiums paid for senior-citizen parents qualify for a separate ₹50,000.")
	}
	sum.AnnualPremium = round2(sum.AnnualPremium)
	return sum, nil
}

func policyLabel(p *models.InsurancePolicy) string {
	if p.PolicyName != "" {
		return p.PolicyName
	}
	return p.Insurer + " " + strings.ReplaceAll(p.PolicyType, "_", " ")
}

func formatLakhs(v float64) string {
	switch {
	case v >= 1e7:
		return fmt.Sprintf("%.2f Cr", v/1e7)
	case v >= 1e5:
		return fmt.Sprintf("%.1f L", v/1e5)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
