package services

import (
	"context"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type CreditCardService struct {
	repo *repository.CreditCardRepo
}

func NewCreditCardService(repo *repository.CreditCardRepo) *CreditCardService {
	return &CreditCardService{repo: repo}
}

func (s *CreditCardService) Create(ctx context.Context, req models.CreateCreditCardRequest) (*models.CreditCard, error) {
	joinDate, err := parseDate(req.JoiningDate)
	if err != nil {
		return nil, err
	}

	dueDateOffset := req.DueDateOffset
	if dueDateOffset == 0 {
		dueDateOffset = 20
	}

	card := &models.CreditCard{
		CardName:         req.CardName,
		BankName:         req.BankName,
		CardNetwork:      req.CardNetwork,
		LastFourDigits:   req.LastFourDigits,
		CardHolderName:   req.CardHolderName,
		CreditLimit:      req.CreditLimit,
		BillingDate:      req.BillingDate,
		DueDateOffset:    dueDateOffset,
		AnnualFee:        req.AnnualFee,
		JoiningDate:      joinDate,
		AvailableCredit:  req.CreditLimit,
		RewardPointValue: req.RewardPointValue,
		CardEMIs:         []models.CardEMI{},
		Statements:       []models.CardStatement{},
		CreditScores:     []models.CreditScoreEntry{},
		Status:           "active",
		Notes:            req.Notes,
	}

	if err := s.repo.Create(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) GetAll(ctx context.Context) ([]models.CreditCard, error) {
	cards, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range cards {
		s.computeDerived(&cards[i])
	}
	return cards, nil
}

func (s *CreditCardService) GetByID(ctx context.Context, id string) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) Update(ctx context.Context, id string, req models.UpdateCreditCardRequest) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.CardName != "" {
		card.CardName = req.CardName
	}
	if req.CreditLimit > 0 {
		card.CreditLimit = req.CreditLimit
	}
	if req.BillingDate > 0 {
		card.BillingDate = req.BillingDate
	}
	if req.DueDateOffset > 0 {
		card.DueDateOffset = req.DueDateOffset
	}
	if req.AnnualFee > 0 {
		card.AnnualFee = req.AnnualFee
	}
	if req.RewardPoints > 0 {
		card.RewardPoints = req.RewardPoints
	}
	if req.CurrentOutstanding >= 0 {
		card.CurrentOutstanding = req.CurrentOutstanding
		card.AvailableCredit = card.CreditLimit - card.CurrentOutstanding
	}
	if req.Notes != "" {
		card.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *CreditCardService) AddStatement(ctx context.Context, id string, req models.AddCardStatementRequest) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	stmtDate, err := parseDate(req.StatementDate)
	if err != nil {
		return nil, err
	}
	dueDate, err := parseDate(req.DueDate)
	if err != nil {
		return nil, err
	}

	stmt := models.CardStatement{
		StatementID:   uuid.New().String(),
		Month:         req.Month,
		StatementDate: stmtDate,
		DueDate:       dueDate,
		TotalAmount:   req.TotalAmount,
		MinimumDue:    req.MinimumDue,
		Transactions:  []models.CardTransaction{},
	}

	card.Statements = append(card.Statements, stmt)
	card.CurrentOutstanding = req.TotalAmount
	card.AvailableCredit = card.CreditLimit - card.CurrentOutstanding

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) PayStatement(ctx context.Context, id string, req models.PayStatementRequest) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	paidDate, err := parseDate(req.PaidDate)
	if err != nil {
		return nil, err
	}

	for i, stmt := range card.Statements {
		if stmt.StatementID == req.StatementID {
			card.Statements[i].AmountPaid = req.AmountPaid
			card.Statements[i].PaidDate = &paidDate
			card.Statements[i].IsPaid = true
			card.Statements[i].PaidFull = req.AmountPaid >= stmt.TotalAmount
			card.CurrentOutstanding -= req.AmountPaid
			if card.CurrentOutstanding < 0 {
				card.CurrentOutstanding = 0
			}
			card.AvailableCredit = card.CreditLimit - card.CurrentOutstanding
			break
		}
	}

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) AddTransaction(ctx context.Context, id string, req models.AddCardTransactionRequest) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	txnDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	txn := models.CardTransaction{
		TransactionID: uuid.New().String(),
		Date:          txnDate,
		Description:   req.Description,
		Amount:        req.Amount,
		Category:      req.Category,
	}

	for i, stmt := range card.Statements {
		if stmt.StatementID == req.StatementID {
			card.Statements[i].Transactions = append(card.Statements[i].Transactions, txn)
			break
		}
	}

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) AddEMI(ctx context.Context, id string, req models.AddCardEMIRequest) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}

	emiAmount := math.Round(req.OriginalAmount/float64(req.TenureMonths)*100) / 100
	if req.InterestRate > 0 {
		r := req.InterestRate / 100 / 12
		n := float64(req.TenureMonths)
		emiAmount = math.Round(req.OriginalAmount*r*math.Pow(1+r, n)/(math.Pow(1+r, n)-1)*100) / 100
	}

	emi := models.CardEMI{
		EMIID:           uuid.New().String(),
		Description:     req.Description,
		MerchantName:    req.MerchantName,
		OriginalAmount:  req.OriginalAmount,
		EMIAmount:       emiAmount,
		TenureMonths:    req.TenureMonths,
		RemainingMonths: req.TenureMonths,
		InterestRate:    req.InterestRate,
		ProcessingFee:   req.ProcessingFee,
		StartDate:       startDate,
		Status:          "active",
	}

	card.CardEMIs = append(card.CardEMIs, emi)

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) AddCreditScore(ctx context.Context, id string, req models.AddCreditScoreRequest) (*models.CreditCard, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	card, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	entry := models.CreditScoreEntry{
		ScoreID: uuid.New().String(),
		Date:    date,
		Score:   req.Score,
		Bureau:  req.Bureau,
		Notes:   req.Notes,
	}

	card.CreditScores = append(card.CreditScores, entry)

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	s.computeDerived(card)
	return card, nil
}

func (s *CreditCardService) computeDerived(card *models.CreditCard) {
	if card.CreditLimit > 0 {
		card.UtilizationPct = math.Round(card.CurrentOutstanding/card.CreditLimit*10000) / 100
	}

	// Spending this month
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	card.TotalSpentThisMonth = 0
	card.SpendByCategory = make(map[string]float64)

	for _, stmt := range card.Statements {
		for _, txn := range stmt.Transactions {
			if !txn.Date.Before(monthStart) {
				card.TotalSpentThisMonth += txn.Amount
			}
			card.SpendByCategory[txn.Category] += txn.Amount
		}
	}

	// Credit score tips
	card.CreditScoreTips = generateCreditScoreTips(card)
}

func generateCreditScoreTips(card *models.CreditCard) []string {
	tips := []string{}

	if card.UtilizationPct > 30 {
		tips = append(tips, "🔴 Credit utilization is above 30%. Try to keep it below 30% for a better score.")
	} else if card.UtilizationPct > 0 {
		tips = append(tips, "🟢 Credit utilization is healthy (below 30%).")
	}

	unpaidCount := 0
	latePayments := 0
	for _, stmt := range card.Statements {
		if !stmt.IsPaid && time.Now().After(stmt.DueDate) {
			unpaidCount++
		}
		if stmt.IsPaid && stmt.PaidDate != nil && stmt.PaidDate.After(stmt.DueDate) {
			latePayments++
		}
	}

	if unpaidCount > 0 {
		tips = append(tips, "🔴 You have unpaid statements past due date. Pay immediately to avoid credit score damage.")
	}
	if latePayments > 0 {
		tips = append(tips, "🟡 You have late payments in history. Always pay before the due date.")
	}

	partialPayments := 0
	for _, stmt := range card.Statements {
		if stmt.IsPaid && !stmt.PaidFull {
			partialPayments++
		}
	}
	if partialPayments > 0 {
		tips = append(tips, "🟡 Paying only minimum due incurs high interest (36-42% p.a.). Always pay the full amount.")
	}

	activeEMIs := 0
	for _, emi := range card.CardEMIs {
		if emi.Status == "active" {
			activeEMIs++
		}
	}
	if activeEMIs > 3 {
		tips = append(tips, "🟡 Multiple active EMIs on card. Too many EMIs can impact your credit utilization.")
	}

	tips = append(tips, "💡 Maintain cards for 3+ years for better credit age score.")
	tips = append(tips, "💡 Don't apply for multiple cards/loans within a short period (hard inquiries).")
	tips = append(tips, "💡 Check your CIBIL report annually for errors and dispute if found.")

	return tips
}
