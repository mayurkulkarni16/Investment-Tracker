package services

import (
	"strings"
	"testing"
	"time"

	"investment-tracker/models"
)

func TestBuildLedgerSummary(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	entries := []models.LedgerEntry{
		{Kind: "income", Status: "confirmed", Date: "2025-03-01", AmountPaise: 10000000, Category: "Salary"},
		{Kind: "expense", Status: "confirmed", Date: "2025-03-05", AmountPaise: 3000000, Category: "Food"},
		{Kind: "expense", Status: "confirmed", Date: "2025-03-06", AmountPaise: 1000000, Category: "Rent"},
		{Kind: "bill", Status: "confirmed", Date: "2025-03-07", AmountPaise: 500000},
		{Kind: "transfer", Status: "confirmed", Date: "2025-03-08", AmountPaise: 700000},
		{Kind: "expense", Status: "pending", Date: "2025-03-09", AmountPaise: 999900},
		{Kind: "expense", Status: "rejected", Date: "2025-03-09", AmountPaise: 999900},
		{Kind: "expense", Status: "confirmed", Date: "2024-01-09", AmountPaise: 999900}, // outside window
		{Kind: "income", Status: "confirmed", Date: "2025-02-01", AmountPaise: 10000000},
		{Kind: "expense", Status: "confirmed", Date: "2025-02-03", AmountPaise: 6000000, Category: "Food"},
	}
	budgets := []models.Budget{{Category: "food", MonthlyLimit: 25000}, {Category: "Rent", MonthlyLimit: 20000}}

	s := BuildLedgerSummary(entries, budgets, 3, now)

	if len(s.Months) != 3 || s.Months[2].Month != "2025-03" || s.Months[0].Month != "2025-01" {
		t.Fatalf("unexpected months: %+v", s.Months)
	}
	cur := s.CurrentMonth
	if cur.Income != 100000 || cur.Expense != 40000 || cur.Bills != 5000 || cur.Transfers != 7000 {
		t.Fatalf("bills/transfers must not count as expenses: %+v", cur)
	}
	if cur.NetSavings != 60000 || cur.SavingsRatePct != 60 {
		t.Fatalf("savings = %v rate = %v", cur.NetSavings, cur.SavingsRatePct)
	}
	if s.PendingReviewCount != 1 {
		t.Fatalf("pending = %d", s.PendingReviewCount)
	}
	if s.AvgMonthlyIncome != 100000 || s.AvgMonthlyExpense != 50000 || s.AvgSavingsRatePct != 50 {
		t.Fatalf("averages: %v %v %v", s.AvgMonthlyIncome, s.AvgMonthlyExpense, s.AvgSavingsRatePct)
	}
	if len(s.TopCategories) != 2 || s.TopCategories[0].Category != "Food" || s.TopCategories[0].Pct != 75 {
		t.Fatalf("top categories: %+v", s.TopCategories)
	}
	if len(s.Budgets) != 2 {
		t.Fatalf("budgets: %+v", s.Budgets)
	}
	for _, b := range s.Budgets {
		switch b.Category {
		case "food":
			if b.Spent != 30000 || b.Status != "over" {
				t.Fatalf("food budget (case-insensitive): %+v", b)
			}
		case "Rent":
			if b.Spent != 10000 || b.Status != "ok" {
				t.Fatalf("rent budget: %+v", b)
			}
		}
	}
}

func TestBudgetAlertThresholds(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		paise  int64
		status string
		pct    float64
	}{
		{"below warning", 7999, "ok", 79.99},
		{"at warning", 8000, "warning", 80},
		{"below limit", 9999, "warning", 99.99},
		{"at limit", 10000, "over", 100},
		{"above limit", 10001, "over", 100.01},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := BuildLedgerSummary(
				[]models.LedgerEntry{{Kind: "expense", Status: "confirmed", Date: "2025-03-01", AmountPaise: tc.paise, Category: "Food"}},
				[]models.Budget{{Category: "Food", MonthlyLimit: 100}},
				1, now,
			)
			if len(s.Budgets) != 1 || s.Budgets[0].Status != tc.status || s.Budgets[0].UsedPct != tc.pct {
				t.Fatalf("expected %s at %.2f%%, got %+v", tc.status, tc.pct, s.Budgets)
			}
		})
	}
}

func TestPremiumDates(t *testing.T) {
	start := time.Date(2020, 4, 10, 0, 0, 0, 0, time.UTC)
	premEnd := time.Date(2024, 4, 10, 0, 0, 0, 0, time.UTC)
	p := &models.InsurancePolicy{PremiumAmount: 1000, PremiumFrequency: "quarterly", StartDate: start, PremiumEndDate: &premEnd}

	fy := PremiumDates(p, time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC))
	if len(fy) != 4 || !fy[0].Equal(time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("quarterly FY dates: %v", fy)
	}
	// Last premium is on the premium end date itself.
	last := PremiumDates(p, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(last) != 1 || !last[0].Equal(premEnd) {
		t.Fatalf("premium end handling: %v", last)
	}

	single := &models.InsurancePolicy{PremiumAmount: 5000, PremiumFrequency: "single", StartDate: start}
	if got := PremiumDates(single, start, start.AddDate(1, 0, 0)); len(got) != 1 {
		t.Fatalf("single premium: %v", got)
	}
	if got := PremiumDates(single, start.AddDate(0, 0, 1), start.AddDate(1, 0, 0)); len(got) != 0 {
		t.Fatalf("single premium outside range: %v", got)
	}
}

func TestBuildFIREPlan(t *testing.T) {
	req := models.FIREPlanRequest{
		CurrentAge: 30, RetirementAge: 50, LifeExpectancy: 85,
		MonthlyExpenses: 50000, CurrentCorpus: 2000000, MonthlySavings: 50000,
		AnnualStepUpPct: 5, PreRetirementReturn: 11, PostRetirementReturn: 7, Inflation: 6,
		RetirementExpensePct: 100,
	}
	p, err := BuildFIREPlan(req, 2025)
	if err != nil {
		t.Fatal(err)
	}
	if p.YearsToRetirement != 20 || p.TargetCorpus <= 0 || p.ProjectedCorpus <= 0 {
		t.Fatalf("basic plan: %+v", p)
	}
	if p.OnTrack != (p.ProjectedCorpus >= p.TargetCorpus) {
		t.Fatalf("on_track inconsistent: projected %v target %v", p.ProjectedCorpus, p.TargetCorpus)
	}
	if p.SuccessProbabilityPct < 0 || p.SuccessProbabilityPct > 100 {
		t.Fatalf("probability: %v", p.SuccessProbabilityPct)
	}
	if len(p.Years) != req.LifeExpectancy-req.CurrentAge+1 && len(p.Years) != req.LifeExpectancy-req.CurrentAge {
		t.Fatalf("years table length %d", len(p.Years))
	}

	// Saving the required amount should (approximately) close the gap.
	req.MonthlySavings = p.RequiredMonthlySavings
	q, err := BuildFIREPlan(req, 2025)
	if err != nil {
		t.Fatal(err)
	}
	if q.Gap > q.TargetCorpus*0.01 {
		t.Fatalf("required savings %v leaves gap %v of target %v", p.RequiredMonthlySavings, q.Gap, q.TargetCorpus)
	}

	// Deterministic Monte Carlo.
	r, _ := BuildFIREPlan(req, 2025)
	if r.SuccessProbabilityPct != q.SuccessProbabilityPct {
		t.Fatalf("monte carlo not deterministic: %v vs %v", r.SuccessProbabilityPct, q.SuccessProbabilityPct)
	}

	bad := req
	bad.RetirementAge = 25
	if _, err := BuildFIREPlan(bad, 2025); err == nil {
		t.Fatal("expected error when retirement age <= current age")
	}
}

func TestReadImportTable(t *testing.T) {
	zerodha := "Holdings statement for client AB1234\n\nSymbol,ISIN,Quantity Available,Average Price,Previous Closing Price\nINFY-EQ,INE009A01021,10,\"1,450.50\",1500\n"
	tbl, err := readImportTable(strings.NewReader(zerodha))
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.rows) != 1 || cleanSymbol(tbl.get(tbl.rows[0], "symbol")) != "INFY" {
		t.Fatalf("zerodha holdings: %+v", tbl)
	}
	if v, _ := parseImportNumber(tbl.get(tbl.rows[0], "avg")); v != 1450.5 {
		t.Fatalf("avg = %v", v)
	}
	if detectBroker(tbl, "") != "zerodha" {
		t.Fatal("expected zerodha")
	}

	groww := "\ufeffStock name,ISIN,Type,Quantity,Value,Exchange,Exchange Order Id,Execution date and time,Order status\nInfosys Ltd,INE009A01021,BUY,5,7500,NSE,1100000001,05-03-2024 10:15 AM,Executed\n"
	tbl, err = readImportTable(strings.NewReader(groww))
	if err != nil {
		t.Fatal(err)
	}
	if detectBroker(tbl, "") != "groww" || tbl.get(tbl.rows[0], "tradeid") != "1100000001" {
		t.Fatalf("groww tradebook: %+v", tbl.cols)
	}
	if d, err := parseImportDate(tbl.get(tbl.rows[0], "date")); err != nil || d.Format("2006-01-02") != "2024-03-05" {
		t.Fatalf("date = %v %v", d, err)
	}

	if _, err := readImportTable(strings.NewReader("foo,bar\n1,2\n")); err == nil {
		t.Fatal("expected error for unrecognised CSV")
	}
}
