package services

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type OverdraftService struct {
	repo *repository.OverdraftRepo
}

func NewOverdraftService(repo *repository.OverdraftRepo) *OverdraftService {
	return &OverdraftService{repo: repo}
}

func (s *OverdraftService) Create(ctx context.Context, userID string, req models.CreateOverdraftRequest) (*models.Overdraft, error) {
	openDate, err := parseDate(req.OpenDate)
	if err != nil {
		return nil, err
	}
	od := &models.Overdraft{
		UserID:          userID,
		LenderName:      strings.TrimSpace(req.LenderName),
		AccountNumber:   strings.TrimSpace(req.AccountNumber),
		SanctionedLimit: req.SanctionedLimit,
		InterestRate:    req.InterestRate,
		OpenDate:        altCalendarDate(openDate),
		Status:          models.LoanActive,
		Notes:           req.Notes,
		Transactions:    []models.OverdraftTransaction{},
	}
	if req.OpeningBalance > 0 {
		od.Transactions = append(od.Transactions, models.OverdraftTransaction{
			TxnID: uuid.New().String(), Date: od.OpenDate, Type: models.OverdraftWithdrawal,
			Amount: round2(req.OpeningBalance), Notes: "Opening balance",
		})
	}
	if err := validateOverdraft(od, true); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, od); err != nil {
		return nil, err
	}
	ComputeOverdraft(od, time.Now())
	return od, nil
}

func (s *OverdraftService) GetAll(ctx context.Context, userID string) ([]models.Overdraft, error) {
	items, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		ComputeOverdraft(&items[i], now)
	}
	return items, nil
}

func (s *OverdraftService) GetByID(ctx context.Context, id string) (*models.Overdraft, error) {
	od, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	ComputeOverdraft(od, time.Now())
	return od, nil
}

func (s *OverdraftService) Update(ctx context.Context, id string, req models.UpdateOverdraftRequest) (*models.Overdraft, error) {
	od, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	openDate, err := parseDate(req.OpenDate)
	if err != nil {
		return nil, err
	}
	od.LenderName = strings.TrimSpace(req.LenderName)
	od.AccountNumber = strings.TrimSpace(req.AccountNumber)
	od.SanctionedLimit = req.SanctionedLimit
	od.InterestRate = req.InterestRate
	od.OpenDate = altCalendarDate(openDate)
	od.Notes = req.Notes
	switch req.Status {
	case "":
	case models.LoanActive, models.LoanClosed:
		od.Status = req.Status
	default:
		return nil, invalidf("status must be active or closed")
	}
	// A lowered limit is allowed: banks can reduce it after money was drawn.
	if err := validateOverdraft(od, false); err != nil {
		return nil, err
	}
	return s.save(ctx, od)
}

func (s *OverdraftService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *OverdraftService) AddTransaction(ctx context.Context, id string, req models.AddOverdraftTransactionRequest) (*models.Overdraft, error) {
	od, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	od.Transactions = append(od.Transactions, models.OverdraftTransaction{
		TxnID: uuid.New().String(), Date: altCalendarDate(date), Type: req.Type,
		Amount: round2(req.Amount), Notes: strings.TrimSpace(req.Notes),
	})
	if err := validateOverdraft(od, req.Type == models.OverdraftWithdrawal); err != nil {
		return nil, err
	}
	return s.save(ctx, od)
}

func (s *OverdraftService) DeleteTransaction(ctx context.Context, id, txnID string) (*models.Overdraft, error) {
	od, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	kept := od.Transactions[:0:0]
	for _, t := range od.Transactions {
		if t.TxnID != txnID {
			kept = append(kept, t)
		}
	}
	if len(kept) == len(od.Transactions) {
		return nil, notFoundf("transaction %s not found", txnID)
	}
	od.Transactions = kept
	if err := validateOverdraft(od, false); err != nil {
		return nil, err
	}
	return s.save(ctx, od)
}

func (s *OverdraftService) load(ctx context.Context, id string) (*models.Overdraft, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, objID)
}

func (s *OverdraftService) save(ctx context.Context, od *models.Overdraft) (*models.Overdraft, error) {
	if err := s.repo.Update(ctx, od); err != nil {
		return nil, err
	}
	ComputeOverdraft(od, time.Now())
	return od, nil
}

// sortOverdraftTransactions orders transactions by date, keeping the recorded
// order for same-day entries.
func sortOverdraftTransactions(od *models.Overdraft) {
	sort.SliceStable(od.Transactions, func(i, j int) bool {
		return od.Transactions[i].Date.Before(od.Transactions[j].Date)
	})
}

// validateOverdraft checks that no transaction predates the account, repayments
// never exceed the drawn balance, an account with a balance is not closed and,
// when checkLimit is set, that the balance never goes over the sanctioned limit.
func validateOverdraft(od *models.Overdraft, checkLimit bool) error {
	sortOverdraftTransactions(od)
	balance := 0.0
	for _, t := range od.Transactions {
		if t.Date.Before(od.OpenDate) {
			return invalidf("a transaction on %s is before the account was opened", t.Date.Format("2006-01-02"))
		}
		switch t.Type {
		case models.OverdraftWithdrawal:
			balance += t.Amount
			if checkLimit && balance > od.SanctionedLimit+0.005 {
				return invalidf("the balance on %s (%.2f) would exceed the sanctioned limit of %.2f", t.Date.Format("2006-01-02"), balance, od.SanctionedLimit)
			}
		case models.OverdraftRepayment:
			balance -= t.Amount
			if balance < -0.005 {
				return invalidf("the repayment on %s is more than the outstanding balance", t.Date.Format("2006-01-02"))
			}
		case models.OverdraftInterest:
		default:
			return invalidf("unknown transaction type %q", t.Type)
		}
	}
	if od.Status == models.LoanClosed && balance > 0.005 {
		return invalidf("repay the outstanding balance of %.2f before closing the overdraft", balance)
	}
	return nil
}

// ComputeOverdraft fills in the balance after each transaction and the derived
// totals. Interest accrues daily (actual/365) on each day's closing balance.
func ComputeOverdraft(od *models.Overdraft, now time.Time) {
	if od.Status == "" {
		od.Status = models.LoanActive
	}
	sortOverdraftTransactions(od)
	od.TotalWithdrawn, od.TotalRepaid, od.TotalInterestPaid = 0, 0, 0
	od.LastInterestPaidDate = nil
	balance := 0.0
	for i := range od.Transactions {
		t := &od.Transactions[i]
		switch t.Type {
		case models.OverdraftWithdrawal:
			balance += t.Amount
			od.TotalWithdrawn += t.Amount
		case models.OverdraftRepayment:
			balance -= t.Amount
			od.TotalRepaid += t.Amount
		case models.OverdraftInterest:
			od.TotalInterestPaid += t.Amount
			d := t.Date
			od.LastInterestPaidDate = &d
		}
		t.BalanceAfter = round2(balance)
	}
	od.Outstanding = round2(math.Max(balance, 0))
	od.AvailableLimit = round2(math.Max(od.SanctionedLimit-od.Outstanding, 0))
	od.UtilizationPct = 0
	if od.SanctionedLimit > 0 {
		od.UtilizationPct = round2(od.Outstanding / od.SanctionedLimit * 100)
	}
	od.TotalWithdrawn = round2(od.TotalWithdrawn)
	od.TotalRepaid = round2(od.TotalRepaid)
	od.TotalInterestPaid = round2(od.TotalInterestPaid)
	od.MonthlyInterest = round2(od.Outstanding * od.InterestRate / 100 / 12)

	from := od.OpenDate
	if od.LastInterestPaidDate != nil {
		// Interest paid on a day settles that day's interest too.
		from = od.LastInterestPaidDate.AddDate(0, 0, 1)
	}
	od.AccruedInterest = round2(OverdraftInterestBetween(od, from, altCalendarDate(now).AddDate(0, 0, 1)))
}

// OverdraftInterestBetween returns the interest accrued on days in [from, to)
// at the current rate.
func OverdraftInterestBetween(od *models.Overdraft, from, to time.Time) float64 {
	from, to = altCalendarDate(from), altCalendarDate(to)
	if !to.After(from) || od.InterestRate <= 0 {
		return 0
	}
	daily := od.InterestRate / 100 / 365
	interest, balance := 0.0, 0.0
	cursor := from
	for _, t := range od.Transactions {
		d := altCalendarDate(t.Date)
		if d.After(cursor) {
			end := d
			if end.After(to) {
				end = to
			}
			if end.After(cursor) {
				interest += balance * daily * end.Sub(cursor).Hours() / 24
				cursor = end
			}
		}
		if !d.Before(to) {
			return interest
		}
		switch t.Type {
		case models.OverdraftWithdrawal:
			balance += t.Amount
		case models.OverdraftRepayment:
			balance -= t.Amount
		}
	}
	if to.After(cursor) {
		interest += math.Max(balance, 0) * daily * to.Sub(cursor).Hours() / 24
	}
	return interest
}

// OverdraftCashflows returns money drawn (inflow, positive) and repayments and
// interest paid (outflow, negative), for cashflow reporting.
func OverdraftCashflows(od *models.Overdraft) []Cashflow {
	flows := make([]Cashflow, 0, len(od.Transactions))
	for _, t := range od.Transactions {
		amount := -t.Amount
		if t.Type == models.OverdraftWithdrawal {
			amount = t.Amount
		}
		flows = append(flows, Cashflow{Date: t.Date, Amount: amount})
	}
	return flows
}
