package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
)

var accountTypes = map[string]bool{"savings": true, "current": true, "salary": true, "cash": true, "wallet": true, "other": true}
var lastFourPattern = regexp.MustCompile(`^([0-9]{4})?$`)

const maxBalanceHistory = 36

// staleAfterDays is when a manually maintained balance is flagged as out of date.
const staleAfterDays = 30

type AccountService struct {
	repo       *repository.BankAccountRepo
	ledgerRepo *repository.LedgerRepo
}

func NewAccountService(repo *repository.BankAccountRepo, ledgerRepo *repository.LedgerRepo) *AccountService {
	return &AccountService{repo: repo, ledgerRepo: ledgerRepo}
}

func validateAccount(req *models.BankAccountRequest) error {
	req.AccountType = strings.ToLower(strings.TrimSpace(req.AccountType))
	if !accountTypes[req.AccountType] {
		return invalidf("account_type must be savings, current, salary, cash, wallet or other")
	}
	req.LastFour = strings.TrimSpace(req.LastFour)
	if !lastFourPattern.MatchString(req.LastFour) {
		return invalidf("last_four must be empty or exactly four digits")
	}
	if req.Balance < 0 && req.AccountType != "current" {
		return invalidf("balance cannot be negative")
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Status != "active" && req.Status != "closed" {
		return invalidf("status must be active or closed")
	}
	return nil
}

func optionalDate(s string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return fallback, nil
	}
	return parseDate(s)
}

func (s *AccountService) Create(ctx context.Context, userID string, req models.BankAccountRequest) (*models.BankAccount, error) {
	if err := validateAccount(&req); err != nil {
		return nil, err
	}
	now := time.Now()
	asOf, err := optionalDate(req.BalanceAsOf, now)
	if err != nil {
		return nil, err
	}
	acc := &models.BankAccount{
		UserID: userID, Name: strings.TrimSpace(req.Name), Institution: req.Institution, AccountType: req.AccountType,
		LastFour: req.LastFour, Balance: req.Balance, BalanceAsOf: asOf, InterestRate: req.InterestRate,
		IsEmergencyFund: req.IsEmergencyFund, Status: req.Status, Notes: req.Notes,
		BalanceHistory: []models.BalancePoint{{Date: asOf, Balance: req.Balance, Source: "manual"}},
		CreatedAt:      now, UpdatedAt: now,
	}
	id, err := s.repo.Insert(ctx, acc)
	if err != nil {
		return nil, err
	}
	acc.ID = id
	s.decorate(acc, nil, now)
	return acc, nil
}

func (s *AccountService) GetAll(ctx context.Context, userID string) ([]models.BankAccount, error) {
	accounts, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	entries, err := s.ledgerRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load account transactions: %w", err)
	}
	now := time.Now()
	for i := range accounts {
		s.decorate(&accounts[i], entries, now)
	}
	return accounts, nil
}

func (s *AccountService) GetByID(ctx context.Context, id string) (*models.BankAccount, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	acc, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	entries, err := s.ledgerRepo.GetAll(ctx, acc.UserID)
	if err != nil {
		return nil, fmt.Errorf("load account transactions: %w", err)
	}
	s.decorate(acc, entries, time.Now())
	return acc, nil
}

func (s *AccountService) Update(ctx context.Context, id string, req models.BankAccountRequest) (*models.BankAccount, error) {
	if err := validateAccount(&req); err != nil {
		return nil, err
	}
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	acc, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	acc.Name = strings.TrimSpace(req.Name)
	acc.Institution = req.Institution
	acc.AccountType = req.AccountType
	acc.LastFour = req.LastFour
	acc.InterestRate = req.InterestRate
	acc.IsEmergencyFund = req.IsEmergencyFund
	acc.Status = req.Status
	acc.Notes = req.Notes
	if req.Balance != acc.Balance {
		asOf, err := optionalDate(req.BalanceAsOf, time.Now())
		if err != nil {
			return nil, err
		}
		applyBalance(acc, req.Balance, asOf, "manual")
	}
	acc.UpdatedAt = time.Now()
	if err := s.repo.Replace(ctx, oid, acc); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *AccountService) UpdateBalance(ctx context.Context, id string, req models.UpdateBalanceRequest) (*models.BankAccount, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	acc, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if req.Balance < 0 && acc.AccountType != "current" {
		return nil, invalidf("balance cannot be negative")
	}
	asOf, err := optionalDate(req.AsOf, time.Now())
	if err != nil {
		return nil, err
	}
	source := req.Source
	if source != "sms" {
		source = "manual"
	}
	applyBalance(acc, req.Balance, asOf, source)
	acc.UpdatedAt = time.Now()
	if err := s.repo.Replace(ctx, oid, acc); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *AccountService) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, oid)
}

// applyBalance records a balance observation. An older observation than the
// current one is kept in history but doesn't overwrite the current balance.
func applyBalance(acc *models.BankAccount, balance float64, asOf time.Time, source string) {
	acc.BalanceHistory = append(acc.BalanceHistory, models.BalancePoint{Date: asOf, Balance: balance, Source: source})
	sort.SliceStable(acc.BalanceHistory, func(i, j int) bool { return acc.BalanceHistory[i].Date.Before(acc.BalanceHistory[j].Date) })
	if len(acc.BalanceHistory) > maxBalanceHistory {
		acc.BalanceHistory = acc.BalanceHistory[len(acc.BalanceHistory)-maxBalanceHistory:]
	}
	if !asOf.Before(acc.BalanceAsOf) {
		acc.Balance = balance
		acc.BalanceAsOf = asOf
	}
}

func (s *AccountService) decorate(acc *models.BankAccount, entries []models.LedgerEntry, now time.Time) {
	if acc.BalanceHistory == nil {
		acc.BalanceHistory = []models.BalancePoint{}
	}
	if !acc.BalanceAsOf.IsZero() {
		acc.DaysStale = int(now.Sub(acc.BalanceAsOf).Hours() / 24)
	}
	month := now.Format("2006-01")
	for _, e := range entries {
		if e.Status != "confirmed" || !strings.HasPrefix(e.Date, month) {
			continue
		}
		if e.LinkedAccountType != "bank_account" || e.LinkedAccountID != acc.ID.Hex() {
			continue
		}
		amt := float64(e.AmountPaise) / 100
		switch e.Kind {
		case "income":
			acc.MonthInflow += amt
		case "expense", "bill":
			acc.MonthOutflow += amt
		}
	}
	acc.MonthInflow = round2(acc.MonthInflow)
	acc.MonthOutflow = round2(acc.MonthOutflow)
}

// LiquidTotals splits active account balances into bank, cash-in-hand and the
// emergency fund (flagged accounts, or all liquid money when none is flagged).
func LiquidTotals(accounts []models.BankAccount) (bank, cash, emergency float64, stale int) {
	flagged := false
	var flaggedSum float64
	now := time.Now()
	for _, a := range accounts {
		if a.Status == "closed" {
			continue
		}
		if a.AccountType == "cash" {
			cash += a.Balance
		} else {
			bank += a.Balance
		}
		if a.IsEmergencyFund {
			flagged = true
			flaggedSum += a.Balance
		}
		if a.AccountType != "cash" && !a.BalanceAsOf.IsZero() && now.Sub(a.BalanceAsOf) > staleAfterDays*24*time.Hour {
			stale++
		}
	}
	emergency = bank + cash
	if flagged {
		emergency = flaggedSum
	}
	return bank, cash, emergency, stale
}

// ---------------------------------------------------------------------------
// Budgets and the monthly income/expense summary built on the ledger.

type BudgetService struct {
	repo       *repository.BudgetRepo
	ledgerRepo *repository.LedgerRepo
}

func NewBudgetService(repo *repository.BudgetRepo, ledgerRepo *repository.LedgerRepo) *BudgetService {
	return &BudgetService{repo: repo, ledgerRepo: ledgerRepo}
}

func (s *BudgetService) GetAll(ctx context.Context, userID string) ([]models.Budget, error) {
	return s.repo.GetAll(ctx, userID)
}

// Upsert creates or updates the budget for a category (case-insensitive).
func (s *BudgetService) Upsert(ctx context.Context, userID string, req models.BudgetRequest) (*models.Budget, error) {
	cat := strings.TrimSpace(req.Category)
	if cat == "" || len(cat) > 60 {
		return nil, invalidf("category is required (max 60 characters)")
	}
	budgets, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, b := range budgets {
		if strings.EqualFold(b.Category, cat) {
			if err := s.repo.Set(ctx, b.ID, bson.M{"monthly_limit": req.MonthlyLimit, "category": cat}); err != nil {
				return nil, err
			}
			b.MonthlyLimit, b.Category, b.UpdatedAt = req.MonthlyLimit, cat, now
			return &b, nil
		}
	}
	b := &models.Budget{UserID: userID, Category: cat, MonthlyLimit: req.MonthlyLimit, CreatedAt: now, UpdatedAt: now}
	id, err := s.repo.Insert(ctx, b)
	if err != nil {
		return nil, err
	}
	b.ID = id
	return b, nil
}

func (s *BudgetService) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, oid)
}

func (s *BudgetService) DeleteLedgerEntry(ctx context.Context, userID, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.ledgerRepo.Delete(ctx, userID, oid)
}

// Summary aggregates confirmed ledger entries into monthly income, expenses and
// savings. Bills (card/utility dues) and transfers are reported separately and
// never counted as expenses, so card purchases aren't double counted.
func (s *BudgetService) Summary(ctx context.Context, userID string, months int) (*models.LedgerSummary, error) {
	if months <= 0 || months > 60 {
		months = 12
	}
	entries, err := s.ledgerRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	budgets, _ := s.repo.GetAll(ctx, userID)
	return BuildLedgerSummary(entries, budgets, months, time.Now()), nil
}

func BuildLedgerSummary(entries []models.LedgerEntry, budgets []models.Budget, months int, now time.Time) *models.LedgerSummary {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
	byMonth := map[string]*models.LedgerMonth{}
	order := make([]string, 0, months)
	for i := 0; i < months; i++ {
		key := start.AddDate(0, i, 0).Format("2006-01")
		byMonth[key] = &models.LedgerMonth{Month: key, ByCategory: map[string]float64{}, IncomeByCat: map[string]float64{}}
		order = append(order, key)
	}

	sum := &models.LedgerSummary{}
	for _, e := range entries {
		if e.Status == "pending" {
			sum.PendingReviewCount++
		}
		if e.Status != "confirmed" || len(e.Date) < 7 {
			continue
		}
		m, ok := byMonth[e.Date[:7]]
		if !ok {
			continue
		}
		amt := float64(e.AmountPaise) / 100
		cat := e.Category
		if cat == "" {
			cat = "Uncategorised"
		}
		m.EntryCount++
		switch e.Kind {
		case "income":
			m.Income += amt
			m.IncomeByCat[cat] += amt
		case "expense":
			m.Expense += amt
			m.ByCategory[cat] += amt
		case "bill":
			m.Bills += amt
		case "transfer":
			m.Transfers += amt
		}
	}

	var incomeSum, expenseSum float64
	activeMonths := 0
	window := 6
	for i, key := range order {
		m := byMonth[key]
		m.Income, m.Expense, m.Bills, m.Transfers = round2(m.Income), round2(m.Expense), round2(m.Bills), round2(m.Transfers)
		m.NetSavings = round2(m.Income - m.Expense)
		if m.Income > 0 {
			m.SavingsRatePct = round2(m.NetSavings / m.Income * 100)
		}
		sum.Months = append(sum.Months, *m)
		if i >= len(order)-window && (m.Income > 0 || m.Expense > 0) {
			incomeSum += m.Income
			expenseSum += m.Expense
			activeMonths++
		}
	}
	if activeMonths > 0 {
		sum.AvgMonthlyIncome = round2(incomeSum / float64(activeMonths))
		sum.AvgMonthlyExpense = round2(expenseSum / float64(activeMonths))
		if incomeSum > 0 {
			sum.AvgSavingsRatePct = round2((incomeSum - expenseSum) / incomeSum * 100)
		}
	}

	cur := sum.Months[len(sum.Months)-1]
	sum.CurrentMonth = cur
	for cat, amt := range cur.ByCategory {
		pct := 0.0
		if cur.Expense > 0 {
			pct = round2(amt / cur.Expense * 100)
		}
		sum.TopCategories = append(sum.TopCategories, models.CategorySpend{Category: cat, Amount: round2(amt), Pct: pct})
	}
	sort.Slice(sum.TopCategories, func(i, j int) bool { return sum.TopCategories[i].Amount > sum.TopCategories[j].Amount })
	if sum.TopCategories == nil {
		sum.TopCategories = []models.CategorySpend{}
	}

	sum.Budgets = []models.BudgetStatus{}
	for _, b := range budgets {
		spent := 0.0
		for cat, amt := range cur.ByCategory {
			if strings.EqualFold(cat, b.Category) {
				spent += amt
			}
		}
		st := models.BudgetStatus{Category: b.Category, Limit: b.MonthlyLimit, Spent: round2(spent), Remaining: round2(b.MonthlyLimit - spent)}
		if b.MonthlyLimit > 0 {
			st.UsedPct = round2(spent / b.MonthlyLimit * 100)
		}
		switch {
		case st.UsedPct >= 100:
			st.Status = "over"
		case st.UsedPct >= 80:
			st.Status = "warning"
		default:
			st.Status = "ok"
		}
		sum.Budgets = append(sum.Budgets, st)
	}
	sort.Slice(sum.Budgets, func(i, j int) bool { return sum.Budgets[i].UsedPct > sum.Budgets[j].UsedPct })
	return sum
}
