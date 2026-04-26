package services

import (
	"context"
	"math"
	"sort"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type HomeLoanService struct {
	repo *repository.HomeLoanRepo
}

func NewHomeLoanService(repo *repository.HomeLoanRepo) *HomeLoanService {
	return &HomeLoanService{repo: repo}
}

func (s *HomeLoanService) Create(ctx context.Context, req models.CreateHomeLoanRequest) (*models.HomeLoan, error) {
	emiStart, err := parseDate(req.EMIStartDate)
	if err != nil {
		return nil, err
	}
	disbDate, err := parseDate(req.DisbursementDate)
	if err != nil {
		return nil, err
	}

	emi := calculateEMI(req.DisbursedAmount, req.InterestRate, req.TenureMonths)
	endDate := emiStart.AddDate(0, req.TenureMonths, 0)

	// For under-construction: EMI is still paid on disbursed amount from the start
	isUC := req.IsUnderConstruction

	initialDisbursements := []models.DisbursementEntry{}
	if isUC {
		// Record initial disbursement as tranche 1 (no pre-EMI for first tranche)
		initialDisbursements = append(initialDisbursements, models.DisbursementEntry{
			DisbursementID: uuid.New().String(),
			Date:           disbDate,
			Amount:         req.DisbursedAmount,
			Tranche:        1,
			Notes:          "Initial disbursement",
		})
	}

	loan := &models.HomeLoan{
		BankName:              req.BankName,
		LoanAccountNumber:     req.LoanAccountNumber,
		PropertyAddress:       req.PropertyAddress,
		LoanPurpose:           req.LoanPurpose,
		SanctionedAmount:      req.SanctionedAmount,
		DisbursedAmount:       req.DisbursedAmount,
		InterestRate:          req.InterestRate,
		RateType:              req.RateType,
		TenureMonths:          req.TenureMonths,
		EMIAmount:             emi,
		EMIStartDate:          emiStart,
		DisbursementDate:      disbDate,
		CoBorrower:            req.CoBorrower,
		IsUnderConstruction:   isUC,
		FullEMIStarted:        true,
		Disbursements:         initialDisbursements,
		PreEMIsPaid:           []models.PreEMIEntry{},
		RateChangeHistory:     []models.RateChangeEntry{},
		EMIsPaid:              []models.EMIEntry{},
		Prepayments:           []models.Prepayment{},
		OutstandingPrincipal:  req.DisbursedAmount,
		RemainingTenureMonths: req.TenureMonths,
		LoanEndDate:           endDate,
		Status:                models.LoanActive,
		Notes:                 req.Notes,
	}

	if err := s.repo.Create(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

func (s *HomeLoanService) GetAll(ctx context.Context) ([]models.HomeLoan, error) {
	loans, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range loans {
		s.computeDerived(&loans[i])
	}
	return loans, nil
}

func (s *HomeLoanService) GetByID(ctx context.Context, id string) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

func (s *HomeLoanService) Update(ctx context.Context, id string, req models.UpdateHomeLoanRequest) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.BankName != "" {
		loan.BankName = req.BankName
	}
	if req.LoanAccountNumber != "" {
		loan.LoanAccountNumber = req.LoanAccountNumber
	}
	if req.PropertyAddress != "" {
		loan.PropertyAddress = req.PropertyAddress
	}
	if req.LoanPurpose != "" {
		loan.LoanPurpose = req.LoanPurpose
	}
	if req.CoBorrower != "" {
		loan.CoBorrower = req.CoBorrower
	}
	if req.Notes != "" {
		loan.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

func (s *HomeLoanService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

// RecordEMIPayment marks an EMI month as paid
func (s *HomeLoanService) RecordEMIPayment(ctx context.Context, id string, req models.AddEMIPaymentRequest) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	paidDate, err := parseDate(req.PaidDate)
	if err != nil {
		return nil, err
	}

	// Calculate principal/interest split for this EMI
	monthlyRate := loan.InterestRate / 100 / 12
	interestPortion := math.Round(loan.OutstandingPrincipal*monthlyRate*100) / 100
	principalPortion := math.Round((loan.EMIAmount-interestPortion)*100) / 100
	outstandingAfter := math.Round((loan.OutstandingPrincipal-principalPortion)*100) / 100
	if outstandingAfter < 0 {
		outstandingAfter = 0
	}

	entry := models.EMIEntry{
		EMIID:            uuid.New().String(),
		Month:            req.Month,
		DueDate:          paidDate,
		EMIAmount:        loan.EMIAmount,
		PrincipalPortion: principalPortion,
		InterestPortion:  interestPortion,
		OutstandingAfter: outstandingAfter,
		Paid:             true,
		PaidDate:         &paidDate,
	}

	loan.EMIsPaid = append(loan.EMIsPaid, entry)
	loan.OutstandingPrincipal = outstandingAfter
	loan.TotalPrincipalPaid += principalPortion
	loan.TotalInterestPaid += interestPortion
	loan.TotalAmountPaid += loan.EMIAmount

	// Recalculate remaining tenure
	if loan.OutstandingPrincipal > 0 && loan.EMIAmount > 0 {
		loan.RemainingTenureMonths = calculateRemainingTenure(loan.OutstandingPrincipal, loan.InterestRate, loan.EMIAmount)
	} else {
		loan.RemainingTenureMonths = 0
		loan.Status = models.LoanClosed
	}

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

// AddPrepayment adds a lump-sum prepayment, reducing outstanding principal
func (s *HomeLoanService) AddPrepayment(ctx context.Context, id string, req models.AddPrepaymentRequest) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	prepayDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	prepay := models.Prepayment{
		PrepaymentID: uuid.New().String(),
		Date:         prepayDate,
		Amount:       req.Amount,
		Type:         req.Type,
		Notes:        req.Notes,
	}

	loan.OutstandingPrincipal -= req.Amount
	if loan.OutstandingPrincipal < 0 {
		loan.OutstandingPrincipal = 0
	}
	loan.TotalPrepayments += req.Amount
	loan.TotalAmountPaid += req.Amount
	loan.TotalPrincipalPaid += req.Amount

	if req.Type == models.PrepaymentForeclosure || loan.OutstandingPrincipal == 0 {
		loan.Status = models.LoanForeclosed
		loan.RemainingTenureMonths = 0
	} else {
		// Keep EMI same, reduce tenure
		loan.RemainingTenureMonths = calculateRemainingTenure(loan.OutstandingPrincipal, loan.InterestRate, loan.EMIAmount)
		prepay.NewTenure = loan.RemainingTenureMonths
	}

	loan.Prepayments = append(loan.Prepayments, prepay)

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

// ChangeRate records a rate change for floating-rate loans
func (s *HomeLoanService) ChangeRate(ctx context.Context, id string, req models.ChangeRateRequest) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	effectiveDate, err := parseDate(req.EffectiveDate)
	if err != nil {
		return nil, err
	}

	oldRate := loan.InterestRate
	newEMI := calculateEMI(loan.OutstandingPrincipal, req.NewRate, loan.RemainingTenureMonths)

	entry := models.RateChangeEntry{
		EffectiveDate: effectiveDate,
		OldRate:       oldRate,
		NewRate:       req.NewRate,
		NewEMI:        newEMI,
	}

	loan.RateChangeHistory = append(loan.RateChangeHistory, entry)
	loan.InterestRate = req.NewRate
	loan.EMIAmount = newEMI

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

// GetAmortizationSchedule generates the remaining amortization schedule
func (s *HomeLoanService) GetAmortizationSchedule(ctx context.Context, id string) ([]models.AmortizationEntry, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	return generateAmortization(loan.OutstandingPrincipal, loan.InterestRate, loan.EMIAmount, loan.RemainingTenureMonths), nil
}

// computeDerived calculates FY tax benefits (not stored)
func (s *HomeLoanService) computeDerived(loan *models.HomeLoan) {
	now := time.Now()
	fyStart := getCurrentFYStart(now)

	interestThisFY := 0.0
	principalThisFY := 0.0

	for _, emi := range loan.EMIsPaid {
		if emi.PaidDate != nil && !emi.PaidDate.Before(fyStart) {
			interestThisFY += emi.InterestPortion
			principalThisFY += emi.PrincipalPortion
		}
	}

	preEMIThisFY := 0.0
	for _, pe := range loan.PreEMIsPaid {
		if pe.PaidDate != nil && !pe.PaidDate.Before(fyStart) {
			preEMIThisFY += pe.InterestAmount
		}
	}

	loan.InterestPaidThisFY = math.Round((interestThisFY+preEMIThisFY)*100) / 100
	loan.PrincipalPaidThisFY = math.Round(principalThisFY*100) / 100
	loan.PreEMIPaidThisFY = math.Round(preEMIThisFY*100) / 100
}

// AddDisbursement adds a new tranche disbursement for under-construction loans
func (s *HomeLoanService) AddDisbursement(ctx context.Context, id string, req models.AddDisbursementRequest) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	disbDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	tranche := len(loan.Disbursements) + 1

	entry := models.DisbursementEntry{
		DisbursementID: uuid.New().String(),
		Date:           disbDate,
		Amount:         req.Amount,
		Tranche:        tranche,
		Notes:          req.Notes,
	}

	loan.Disbursements = append(loan.Disbursements, entry)
	loan.DisbursedAmount += req.Amount
	loan.OutstandingPrincipal += req.Amount

	// Auto-calculate pro-rata pre-EMI: interest on new tranche for remaining days of the month
	daysInMonth := time.Date(disbDate.Year(), disbDate.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	remainingDays := daysInMonth - disbDate.Day() + 1 // include disbursement day
	dailyRate := loan.InterestRate / 100.0 / 365.0
	preEMIAmount := math.Round(req.Amount*dailyRate*float64(remainingDays)*100) / 100

	// Store pre-EMI on the disbursement entry
	entry.PreEMIAmount = preEMIAmount
	entry.PreEMIDays = remainingDays
	entry.DaysInMonth = daysInMonth
	loan.Disbursements[len(loan.Disbursements)-1] = entry

	// Also create a PreEMIEntry for tracking
	preEmiEntry := models.PreEMIEntry{
		PreEMIID:        uuid.New().String(),
		Month:           disbDate.Format("2006-01"),
		DisbursedAtTime: loan.DisbursedAmount,
		TrancheAmount:   req.Amount,
		InterestRate:    loan.InterestRate,
		InterestAmount:  preEMIAmount,
		DaysCharged:     remainingDays,
		DaysInMonth:     daysInMonth,
		Paid:            true,
		PaidDate:        &disbDate,
	}
	loan.PreEMIsPaid = append(loan.PreEMIsPaid, preEmiEntry)
	loan.TotalPreEMIPaid += preEMIAmount
	loan.TotalInterestPaid += preEMIAmount
	loan.TotalAmountPaid += preEMIAmount

	// Recalculate EMI based on new outstanding principal (effective next month)
	loan.EMIAmount = calculateEMI(loan.OutstandingPrincipal, loan.InterestRate, loan.RemainingTenureMonths)

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

// MarkConstructionComplete transitions from under-construction to regular loan
func (s *HomeLoanService) MarkConstructionComplete(ctx context.Context, id string, req models.MarkConstructionCompleteRequest) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	loan.IsUnderConstruction = false

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

// Recalculate replays all transactions chronologically, recalculating EMIs and pre-EMIs
// with the correct interest rate at each point. Use after recording a rate change retroactively.
func (s *HomeLoanService) Recalculate(ctx context.Context, id string) (*models.HomeLoan, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	loan, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	// Determine initial interest rate (before any rate changes)
	initialRate := loan.InterestRate
	if len(loan.RateChangeHistory) > 0 {
		sort.Slice(loan.RateChangeHistory, func(i, j int) bool {
			return loan.RateChangeHistory[i].EffectiveDate.Before(loan.RateChangeHistory[j].EffectiveDate)
		})
		initialRate = loan.RateChangeHistory[0].OldRate
	}

	// Sort EMIs by paid date
	sort.Slice(loan.EMIsPaid, func(i, j int) bool {
		di, dj := loan.EMIsPaid[i].DueDate, loan.EMIsPaid[j].DueDate
		if loan.EMIsPaid[i].PaidDate != nil {
			di = *loan.EMIsPaid[i].PaidDate
		}
		if loan.EMIsPaid[j].PaidDate != nil {
			dj = *loan.EMIsPaid[j].PaidDate
		}
		return di.Before(dj)
	})

	// Sort disbursements and prepayments by date
	sort.Slice(loan.Disbursements, func(i, j int) bool {
		return loan.Disbursements[i].Date.Before(loan.Disbursements[j].Date)
	})
	sort.Slice(loan.Prepayments, func(i, j int) bool {
		return loan.Prepayments[i].Date.Before(loan.Prepayments[j].Date)
	})

	// Build chronological event list
	// Priority on same day: disbursement(0) > emi(1) > prepayment(2) > rate_change(3)
	// Rate change on same day as EMI means EMI uses old rate; new rate applies next month
	type replayEvent struct {
		date     time.Time
		priority int // lower = processed first on same day
		index    int
	}
	var events []replayEvent

	for i, d := range loan.Disbursements {
		events = append(events, replayEvent{date: d.Date, priority: 0, index: i})
	}
	for i, e := range loan.EMIsPaid {
		d := e.DueDate
		if e.PaidDate != nil {
			d = *e.PaidDate
		}
		events = append(events, replayEvent{date: d, priority: 1, index: 1000 + i}) // offset to distinguish
	}
	for i, p := range loan.Prepayments {
		events = append(events, replayEvent{date: p.Date, priority: 2, index: 2000 + i})
	}
	for i, r := range loan.RateChangeHistory {
		events = append(events, replayEvent{date: r.EffectiveDate, priority: 3, index: 3000 + i})
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].date.Equal(events[j].date) {
			return events[i].priority < events[j].priority
		}
		return events[i].date.Before(events[j].date)
	})

	// Replay state
	isUC := len(loan.Disbursements) > 0
	currentRate := initialRate
	outstanding := 0.0
	disbursedAmount := 0.0
	if !isUC {
		outstanding = loan.DisbursedAmount
		disbursedAmount = loan.DisbursedAmount
	}
	remainingTenure := loan.TenureMonths
	currentEMI := calculateEMI(outstanding, currentRate, remainingTenure)

	// Reset totals
	totalPrincipalPaid := 0.0
	totalInterestPaid := 0.0
	totalAmountPaid := 0.0
	totalPrepayments := 0.0
	totalPreEMIPaid := 0.0

	// Clear pre-EMIs (regenerated from disbursements)
	loan.PreEMIsPaid = []models.PreEMIEntry{}

	for _, ev := range events {
		switch {
		case ev.priority == 0: // disbursement
			idx := ev.index
			d := &loan.Disbursements[idx]
			outstanding += d.Amount
			disbursedAmount += d.Amount

			// Pre-EMI for non-first tranches
			if d.Tranche > 1 {
				daysInMonth := time.Date(d.Date.Year(), d.Date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
				remainingDays := daysInMonth - d.Date.Day() + 1
				dailyRate := currentRate / 100.0 / 365.0
				preEMIAmount := math.Round(d.Amount*dailyRate*float64(remainingDays)*100) / 100

				d.PreEMIAmount = preEMIAmount
				d.PreEMIDays = remainingDays
				d.DaysInMonth = daysInMonth

				paidDate := d.Date
				loan.PreEMIsPaid = append(loan.PreEMIsPaid, models.PreEMIEntry{
					PreEMIID:        uuid.New().String(),
					Month:           d.Date.Format("2006-01"),
					DisbursedAtTime: disbursedAmount,
					TrancheAmount:   d.Amount,
					InterestRate:    currentRate,
					InterestAmount:  preEMIAmount,
					DaysCharged:     remainingDays,
					DaysInMonth:     daysInMonth,
					Paid:            true,
					PaidDate:        &paidDate,
				})
				totalPreEMIPaid += preEMIAmount
				totalInterestPaid += preEMIAmount
				totalAmountPaid += preEMIAmount
			}

			currentEMI = calculateEMI(outstanding, currentRate, remainingTenure)

		case ev.priority == 1: // emi
			idx := ev.index - 1000
			e := &loan.EMIsPaid[idx]

			monthlyRate := currentRate / 100 / 12
			interestPortion := math.Round(outstanding*monthlyRate*100) / 100
			principalPortion := math.Round((currentEMI-interestPortion)*100) / 100
			outstandingAfter := math.Round((outstanding-principalPortion)*100) / 100
			if outstandingAfter < 0 {
				outstandingAfter = 0
			}

			e.EMIAmount = currentEMI
			e.InterestPortion = interestPortion
			e.PrincipalPortion = principalPortion
			e.OutstandingAfter = outstandingAfter

			outstanding = outstandingAfter
			totalPrincipalPaid += principalPortion
			totalInterestPaid += interestPortion
			totalAmountPaid += currentEMI
			remainingTenure--

		case ev.priority == 2: // prepayment
			idx := ev.index - 2000
			p := &loan.Prepayments[idx]
			outstanding -= p.Amount
			if outstanding < 0 {
				outstanding = 0
			}
			totalPrepayments += p.Amount
			totalAmountPaid += p.Amount
			totalPrincipalPaid += p.Amount

			if outstanding > 0 && currentEMI > 0 {
				remainingTenure = calculateRemainingTenure(outstanding, currentRate, currentEMI)
				p.NewTenure = remainingTenure
			}

		case ev.priority == 3: // rate_change
			idx := ev.index - 3000
			rc := &loan.RateChangeHistory[idx]
			rc.OldRate = currentRate
			currentRate = rc.NewRate
			currentEMI = calculateEMI(outstanding, currentRate, remainingTenure)
			rc.NewEMI = currentEMI
		}
	}

	// Update loan state
	loan.InterestRate = currentRate
	loan.EMIAmount = currentEMI
	loan.OutstandingPrincipal = outstanding
	loan.DisbursedAmount = disbursedAmount
	loan.TotalPrincipalPaid = totalPrincipalPaid
	loan.TotalInterestPaid = totalInterestPaid
	loan.TotalAmountPaid = totalAmountPaid
	loan.TotalPrepayments = totalPrepayments
	loan.TotalPreEMIPaid = totalPreEMIPaid

	if outstanding > 0 && currentEMI > 0 {
		loan.RemainingTenureMonths = calculateRemainingTenure(outstanding, currentRate, currentEMI)
	} else {
		loan.RemainingTenureMonths = 0
		if outstanding <= 0 {
			loan.Status = models.LoanClosed
		}
	}

	if err := s.repo.Update(ctx, loan); err != nil {
		return nil, err
	}
	s.computeDerived(loan)
	return loan, nil
}

// calculateEMI computes EMI using standard formula: P * r * (1+r)^n / ((1+r)^n - 1)
func calculateEMI(principal float64, annualRate float64, tenureMonths int) float64 {
	r := annualRate / 100 / 12
	n := float64(tenureMonths)

	if r == 0 {
		return math.Round(principal / n)
	}

	emi := principal * r * math.Pow(1+r, n) / (math.Pow(1+r, n) - 1)
	return math.Round(emi)
}

func calculateRemainingTenure(outstanding float64, annualRate float64, emi float64) int {
	r := annualRate / 100 / 12
	if r == 0 {
		return int(math.Ceil(outstanding / emi))
	}
	// n = log(EMI / (EMI - P*r)) / log(1+r)
	monthlyInterest := outstanding * r
	if emi <= monthlyInterest {
		return 999 // EMI too low to cover interest
	}
	n := math.Log(emi/(emi-outstanding*r)) / math.Log(1+r)
	return int(math.Ceil(n))
}

func generateAmortization(principal float64, annualRate float64, emi float64, months int) []models.AmortizationEntry {
	r := annualRate / 100 / 12
	balance := principal
	schedule := make([]models.AmortizationEntry, 0, months)

	for m := 1; m <= months && balance > 0; m++ {
		interest := math.Round(balance*r*100) / 100
		principalPart := math.Round((emi-interest)*100) / 100
		if principalPart > balance {
			principalPart = balance
		}
		balance = math.Round((balance-principalPart)*100) / 100
		if balance < 0 {
			balance = 0
		}

		schedule = append(schedule, models.AmortizationEntry{
			Month:            m,
			EMI:              emi,
			PrincipalPortion: principalPart,
			InterestPortion:  interest,
			OutstandingAfter: balance,
		})
	}
	return schedule
}
