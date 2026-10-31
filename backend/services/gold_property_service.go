package services

import (
	"context"
	"math"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var goldForms = map[string]bool{"physical": true, "digital": true, "sgb": true, "etf": true, "mutual_fund": true}
var propertyTypes = map[string]bool{"residential": true, "commercial": true, "land": true, "other": true}

// growthCAGR returns the annualised growth (%) from cost to value since start.
// Periods under a year return the simple (non-annualised) return.
func growthCAGR(cost, value float64, start, now time.Time) float64 {
	if cost <= 0 || value <= 0 || start.IsZero() {
		return 0
	}
	years := now.Sub(start).Hours() / 24 / 365.25
	if years < 1 {
		return round2((value/cost - 1) * 100)
	}
	return round2((math.Pow(value/cost, 1/years) - 1) * 100)
}

// ---------------------------------------------------------------------------
// Gold

type GoldService struct {
	repo         *repository.GoldRepo
	priceFetcher *StockPriceFetcher
}

func NewGoldService(repo *repository.GoldRepo, priceFetcher *StockPriceFetcher) *GoldService {
	return &GoldService{repo: repo, priceFetcher: priceFetcher}
}

func isGramBased(form string) bool { return form == "physical" || form == "digital" || form == "sgb" }

func (s *GoldService) build(userID string, req models.GoldHoldingRequest, g *models.GoldHolding) error {
	req.Form = strings.ToLower(strings.TrimSpace(req.Form))
	if !goldForms[req.Form] {
		return invalidf("form must be physical, digital, sgb, etf or mutual_fund")
	}
	purchase, err := parseDate(req.PurchaseDate)
	if err != nil {
		return err
	}
	purity := req.Purity
	if req.Form != "physical" || purity == 0 {
		purity = 24
	}
	if purity < 8 || purity > 24 {
		return invalidf("purity must be between 8 and 24 karat")
	}
	var maturity *time.Time
	if req.MaturityDate != "" {
		t, err := parseDate(req.MaturityDate)
		if err != nil {
			return err
		}
		maturity = &t
	} else if req.Form == "sgb" {
		t := purchase.AddDate(8, 0, 0) // SGBs mature after 8 years
		maturity = &t
	}
	rate := req.InterestRate
	if req.Form == "sgb" && rate == 0 {
		rate = 2.5
	}
	if req.Form != "sgb" {
		rate = 0
	}
	if g.CurrentPrice != req.CurrentPrice && req.CurrentPrice > 0 {
		now := time.Now()
		g.PriceUpdatedAt = &now
	}
	g.UserID, g.Name, g.Form, g.Quantity, g.Purity = userID, strings.TrimSpace(req.Name), req.Form, req.Quantity, purity
	g.PurchasePrice, g.PurchaseDate, g.CurrentPrice = req.PurchasePrice, purchase, req.CurrentPrice
	g.Symbol, g.InterestRate, g.MaturityDate, g.Notes = strings.ToUpper(strings.TrimSpace(req.Symbol)), rate, maturity, req.Notes
	return nil
}

func ComputeGold(g *models.GoldHolding, now time.Time) {
	value := g.Quantity * g.CurrentPrice
	if g.CurrentPrice <= 0 {
		value = g.PurchasePrice
	}
	g.CurrentValue = round2(value)
	g.GainLoss = round2(value - g.PurchasePrice)
	if g.PurchasePrice > 0 {
		g.GainLossPct = round2(g.GainLoss / g.PurchasePrice * 100)
	}
	g.CAGR = growthCAGR(g.PurchasePrice, value, g.PurchaseDate, now)
	if g.Form == "sgb" && g.InterestRate > 0 && g.PurchasePrice > 0 {
		end := now
		if g.MaturityDate != nil && g.MaturityDate.Before(end) {
			end = *g.MaturityDate
		}
		halfYears := 0
		for d := g.PurchaseDate.AddDate(0, 6, 0); !d.After(end); d = d.AddDate(0, 6, 0) {
			halfYears++
		}
		g.InterestEarned = round2(float64(halfYears) * g.PurchasePrice * g.InterestRate / 200)
	}
}

func (s *GoldService) Create(ctx context.Context, userID string, req models.GoldHoldingRequest) (*models.GoldHolding, error) {
	g := &models.GoldHolding{}
	if err := s.build(userID, req, g); err != nil {
		return nil, err
	}
	g.CreatedAt, g.UpdatedAt = time.Now(), time.Now()
	id, err := s.repo.Insert(ctx, g)
	if err != nil {
		return nil, err
	}
	g.ID = id
	ComputeGold(g, time.Now())
	return g, nil
}

func (s *GoldService) GetAll(ctx context.Context, userID string) ([]models.GoldHolding, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		ComputeGold(&items[i], now)
	}
	return items, nil
}

func (s *GoldService) GetByID(ctx context.Context, id string) (*models.GoldHolding, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	ComputeGold(g, time.Now())
	return g, nil
}

func (s *GoldService) Update(ctx context.Context, id string, req models.GoldHoldingRequest) (*models.GoldHolding, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if err := s.build(g.UserID, req, g); err != nil {
		return nil, err
	}
	g.UpdatedAt = time.Now()
	if err := s.repo.Replace(ctx, oid, g); err != nil {
		return nil, err
	}
	ComputeGold(g, time.Now())
	return g, nil
}

func (s *GoldService) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, oid)
}

// SetRate revalues gram-based holdings from today's 24K rate per gram.
func (s *GoldService) SetRate(ctx context.Context, userID string, rate24K float64) ([]models.GoldHolding, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		g := &items[i]
		if !isGramBased(g.Form) {
			continue
		}
		purity := g.Purity
		if purity == 0 {
			purity = 24
		}
		g.CurrentPrice = round2(rate24K * purity / 24)
		g.PriceUpdatedAt = &now
		g.UpdatedAt = now
		if err := s.repo.Replace(ctx, g.ID, g); err != nil {
			return nil, err
		}
	}
	return s.GetAll(ctx, userID)
}

// RefreshPrices fetches NSE prices for gold ETFs that have a symbol.
func (s *GoldService) RefreshPrices(ctx context.Context, userID string) ([]models.GoldHolding, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		g := &items[i]
		if g.Form != "etf" || g.Symbol == "" || s.priceFetcher == nil {
			continue
		}
		q, err := s.priceFetcher.FetchQuote(g.Symbol, "NSE")
		if err != nil || q.Price <= 0 {
			continue
		}
		g.CurrentPrice = q.Price
		g.PriceUpdatedAt = &now
		g.UpdatedAt = now
		if err := s.repo.Replace(ctx, g.ID, g); err != nil {
			return nil, err
		}
	}
	return s.GetAll(ctx, userID)
}

// ---------------------------------------------------------------------------
// Real estate

type PropertyService struct {
	repo         *repository.PropertyRepo
	homeLoanRepo *repository.HomeLoanRepo
}

func NewPropertyService(repo *repository.PropertyRepo, homeLoanRepo *repository.HomeLoanRepo) *PropertyService {
	return &PropertyService{repo: repo, homeLoanRepo: homeLoanRepo}
}

func (s *PropertyService) build(userID string, req models.PropertyRequest, p *models.Property) error {
	req.PropertyType = strings.ToLower(strings.TrimSpace(req.PropertyType))
	if !propertyTypes[req.PropertyType] {
		return invalidf("property_type must be residential, commercial, land or other")
	}
	purchase, err := parseDate(req.PurchaseDate)
	if err != nil {
		return err
	}
	if req.OwnershipPct == 0 {
		req.OwnershipPct = 100
	}
	if req.OwnershipPct < 0 || req.OwnershipPct > 100 {
		return invalidf("ownership_pct must be between 0 and 100")
	}
	if req.HomeLoanID != "" {
		if _, err := primitive.ObjectIDFromHex(req.HomeLoanID); err != nil {
			return invalidf("invalid home_loan_id")
		}
	}
	valueAsOf := time.Now()
	if req.ValueUpdatedAt != "" {
		if valueAsOf, err = parseDate(req.ValueUpdatedAt); err != nil {
			return err
		}
	} else if p.CurrentValue == req.CurrentValue && !p.ValueUpdatedAt.IsZero() {
		valueAsOf = p.ValueUpdatedAt
	}
	if req.CurrentValue == 0 {
		req.CurrentValue = req.PurchasePrice
	}
	if req.IsSelfOccupied {
		req.MonthlyRent = 0
	}
	p.UserID, p.Name, p.PropertyType, p.Location = userID, strings.TrimSpace(req.Name), req.PropertyType, req.Location
	p.PurchasePrice, p.PurchaseDate, p.CurrentValue, p.ValueUpdatedAt = req.PurchasePrice, purchase, req.CurrentValue, valueAsOf
	p.OwnershipPct, p.HomeLoanID, p.IsSelfOccupied = req.OwnershipPct, req.HomeLoanID, req.IsSelfOccupied
	p.MonthlyRent, p.AnnualCosts, p.Notes = req.MonthlyRent, req.AnnualCosts, req.Notes
	return nil
}

func ComputeProperty(p *models.Property, loans map[string]float64, now time.Time) {
	share := p.OwnershipPct / 100
	if share == 0 {
		share = 1
	}
	p.OwnedValue = round2(p.CurrentValue * share)
	ownedCost := p.PurchasePrice * share
	p.GainLoss = round2(p.OwnedValue - ownedCost)
	p.CAGR = growthCAGR(p.PurchasePrice, p.CurrentValue, p.PurchaseDate, now)
	if p.CurrentValue > 0 && !p.IsSelfOccupied {
		p.NetRentalYield = round2((p.MonthlyRent*12 - p.AnnualCosts) / p.CurrentValue * 100)
	}
	p.LoanOutstanding = round2(loans[p.HomeLoanID])
	p.Equity = round2(p.OwnedValue - p.LoanOutstanding)
}

func (s *PropertyService) loanMap(ctx context.Context, userID string) map[string]float64 {
	m := map[string]float64{}
	loans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	for _, l := range loans {
		if l.Status == models.LoanActive {
			m[l.ID.Hex()] = l.OutstandingPrincipal
		}
	}
	return m
}

func (s *PropertyService) Create(ctx context.Context, userID string, req models.PropertyRequest) (*models.Property, error) {
	p := &models.Property{}
	if err := s.build(userID, req, p); err != nil {
		return nil, err
	}
	p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
	id, err := s.repo.Insert(ctx, p)
	if err != nil {
		return nil, err
	}
	p.ID = id
	ComputeProperty(p, s.loanMap(ctx, userID), time.Now())
	return p, nil
}

func (s *PropertyService) GetAll(ctx context.Context, userID string) ([]models.Property, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	loans := s.loanMap(ctx, userID)
	now := time.Now()
	for i := range items {
		ComputeProperty(&items[i], loans, now)
	}
	return items, nil
}

func (s *PropertyService) GetByID(ctx context.Context, id string) (*models.Property, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	ComputeProperty(p, s.loanMap(ctx, p.UserID), time.Now())
	return p, nil
}

func (s *PropertyService) Update(ctx context.Context, id string, req models.PropertyRequest) (*models.Property, error) {
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
	ComputeProperty(p, s.loanMap(ctx, p.UserID), time.Now())
	return p, nil
}

func (s *PropertyService) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, oid)
}
