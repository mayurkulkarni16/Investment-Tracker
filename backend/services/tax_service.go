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

type TaxService struct {
	mfRepo           *repository.MutualFundRepo
	fdRepo           *repository.FixedDepositRepo
	pfRepo           *repository.ProvidentFundRepo
	npsRepo          *repository.NPSRepo
	bondRepo         *repository.CorporateBondRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	stockRepo        *repository.StockRepo
	insuranceRepo    *repository.InsuranceRepo
	propertyRepo     *repository.PropertyRepo
	rentalRepo       *repository.VehicleRentalRepo
}

func NewTaxService(
	mfRepo *repository.MutualFundRepo,
	fdRepo *repository.FixedDepositRepo,
	pfRepo *repository.ProvidentFundRepo,
	npsRepo *repository.NPSRepo,
	bondRepo *repository.CorporateBondRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	stockRepo *repository.StockRepo,
) *TaxService {
	return &TaxService{
		mfRepo: mfRepo, fdRepo: fdRepo, pfRepo: pfRepo, npsRepo: npsRepo,
		bondRepo: bondRepo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		stockRepo: stockRepo,
	}
}

// SetInsuranceRepo counts insurance premiums under 80C (life) and 80D (health).
func (s *TaxService) SetInsuranceRepo(r *repository.InsuranceRepo) { s.insuranceRepo = r }

// SetIncomeRepos adds rental income from let-out property and vehicle leasing.
func (s *TaxService) SetIncomeRepos(property *repository.PropertyRepo, rentals *repository.VehicleRentalRepo) {
	s.propertyRepo, s.rentalRepo = property, rentals
}

// parseFYRange parses a financial year string like "2025-2026" or "2025-26" into start and end dates.
// Returns (2025-04-01, 2026-04-01) for FY 2025-2026. Falls back to current FY if empty or invalid.
func parseFYRange(fy string) (time.Time, time.Time) {
	if fy != "" {
		parts := strings.Split(fy, "-")
		if len(parts) == 2 {
			startYear := 0
			if _, err := fmt.Sscanf(parts[0], "%d", &startYear); err == nil && startYear > 2000 {
				fyStart := time.Date(startYear, 4, 1, 0, 0, 0, 0, time.Local)
				fyEnd := fyStart.AddDate(1, 0, 0)
				return fyStart, fyEnd
			}
		}
	}
	// Fallback to current FY
	fyStart := getCurrentFYStart(time.Now())
	return fyStart, fyStart.AddDate(1, 0, 0)
}

// fyLabel returns "2025-26" for the FY starting in 2025.
func fyLabel(fyStart time.Time) string {
	return fmt.Sprintf("%d-%02d", fyStart.Year(), (fyStart.Year()+1)%100)
}

// GetTaxSummary returns the household's deductions and investment income for a FY.
func (s *TaxService) GetTaxSummary(ctx context.Context, userID string, fy string) (*models.TaxSummary, error) {
	lines, cg, label, err := s.CollectLines(ctx, userID, fy)
	if err != nil {
		return nil, err
	}
	return SummaryFromLines(label, lines, cg), nil
}

// CollectLines gathers every tax-relevant amount from tracked assets for the
// FY (household level, before splitting between members), plus capital gains.
func (s *TaxService) CollectLines(ctx context.Context, userID string, fy string) ([]models.TaxLine, *models.CapitalGainsSummary, string, error) {
	fyStart, fyEnd := parseFYRange(fy)
	var lines []models.TaxLine
	add := func(assetType, id, name, kind string, amount float64) {
		if amount == 0 {
			return
		}
		lines = append(lines, models.TaxLine{AssetType: assetType, AssetID: id, Name: name, Kind: kind, Amount: amount})
	}
	inFY := func(t time.Time) bool { return !t.Before(fyStart) && t.Before(fyEnd) }

	// EPF / VPF / PPF employee deposits (80C)
	pfs, _ := s.pfRepo.GetAll(ctx, userID)
	for _, pf := range pfs {
		kind := models.Line80CEPF
		if pf.AccountType == models.PFTypePPF {
			kind = models.Line80CPPF
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", pf.AccountType, pf.EmployerName))
		amt := 0.0
		for _, fyEntry := range pf.FinancialYearEntries {
			for _, mc := range fyEntry.MonthlyContributions {
				t := parsePFMonth(mc.Month, fyEntry.FinancialYear)
				if !t.IsZero() && inFY(t) {
					amt += mc.EmployeeContribution
				}
			}
		}
		add("provident_fund", pf.ID.Hex(), name, kind, amt)
	}

	// ELSS purchases (80C)
	mfs, _ := s.mfRepo.GetAll(ctx, userID)
	for _, mf := range mfs {
		if !mf.IsELSS {
			continue
		}
		amt := 0.0
		for _, txn := range mf.Transactions {
			if (txn.Type == models.TransactionPurchase || txn.Type == models.TransactionSIP) && inFY(txn.Date) {
				amt += txn.Amount
			}
		}
		add("mutual_fund", mf.ID.Hex(), mf.FundName, models.Line80CELSS, amt)
	}

	// Home loans: principal (80C) and interest (24(b))
	letOutLoans := map[string]bool{}
	if s.propertyRepo != nil {
		props, _ := s.propertyRepo.GetAll(ctx, userID)
		for _, p := range props {
			if p.HomeLoanID != "" && !p.IsSelfOccupied && p.MonthlyRent > 0 {
				letOutLoans[p.HomeLoanID] = true
			}
		}
		for _, p := range props {
			if p.IsSelfOccupied || p.MonthlyRent <= 0 {
				continue
			}
			months := 0
			for m := fyStart; m.Before(fyEnd); m = m.AddDate(0, 1, 0) {
				if !m.AddDate(0, 1, 0).Before(p.PurchaseDate) {
					months++
				}
			}
			own := p.OwnershipPct
			if own <= 0 || own > 100 {
				own = 100
			}
			add("property", p.ID.Hex(), p.Name, models.LineRent, round2(p.MonthlyRent*float64(months)*own/100))
		}
	}
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	for _, loan := range homeLoans {
		id := loan.ID.Hex()
		var principal, interest, preEMI float64
		for _, emi := range loan.EMIsPaid {
			if emi.PaidDate != nil && inFY(*emi.PaidDate) {
				principal += emi.PrincipalPortion
				interest += emi.InterestPortion
			}
		}
		for _, pe := range loan.PreEMIsPaid {
			if pe.PaidDate != nil && inFY(*pe.PaidDate) {
				preEMI += pe.InterestAmount
			}
		}
		add("home_loan", id, loan.BankName, models.Line80CPrincipal, principal)
		if letOutLoans[id] {
			add("home_loan", id, loan.BankName, models.Line24bLetOut, interest)
		} else {
			add("home_loan", id, loan.BankName, models.Line24bSelfOccupied, interest)
		}
		add("home_loan", id, loan.BankName, models.Line24bPreEMI, preEMI)
	}

	// Insurance premiums: life (80C, capped at 10% of sum assured) and health (80D)
	if s.insuranceRepo != nil {
		policies, _ := s.insuranceRepo.GetAll(ctx, userID)
		for i := range policies {
			p := &policies[i]
			if p.Status == "lapsed" {
				continue
			}
			total := float64(len(PremiumDates(p, fyStart, fyEnd))) * p.PremiumAmount
			name := policyLabel(p)
			switch p.PolicyType {
			case "term", "life", "ulip":
				if p.SumAssured > 0 {
					total = math.Min(total, 0.10*p.SumAssured)
				}
				add("insurance", p.ID.Hex(), name, models.Line80CLife, round2(total))
			case "health":
				kind := models.Line80DSelf
				if p.HealthCoverFor == "parents" {
					kind = models.Line80DParents
				}
				if total != 0 {
					lines = append(lines, models.TaxLine{AssetType: "insurance", AssetID: p.ID.Hex(), Name: name, Kind: kind,
						Amount: round2(total), Senior: p.InsuredSenior})
				}
			}
		}
	}

	// NPS Tier-1: own (80CCD(1)/(1B)) and employer (80CCD(2)). Tier-2 is not deductible.
	npsAccounts, _ := s.npsRepo.GetAll(ctx, userID)
	for _, nps := range npsAccounts {
		if nps.AccountType == models.NPSTierII {
			continue
		}
		var own, employer float64
		for _, c := range nps.Contributions {
			if inFY(c.Date) {
				if c.Type == "employer" {
					employer += c.Amount
				} else {
					own += c.Amount
				}
			}
		}
		name := "NPS " + nps.PRAN
		add("nps", nps.ID.Hex(), name, models.LineNPSOwn, own)
		add("nps", nps.ID.Hex(), name, models.LineNPSEmployer, employer)
	}

	// Interest income for the whole FY (taxable as it accrues / is paid, so
	// future payouts within the FY are included as an estimate).
	fds, _ := s.fdRepo.GetAll(ctx, userID)
	for i := range fds {
		add("fixed_deposit", fds[i].ID.Hex(), fds[i].BankName, models.LineInterest, round2(FixedDepositInterestBetween(&fds[i], fyStart, fyEnd)))
	}
	bonds, _ := s.bondRepo.GetAll(ctx, userID)
	for _, bond := range bonds {
		amt := 0.0
		for _, p := range bond.InterestPayouts {
			if inFY(p.ScheduledDate) {
				amt += p.Amount
			}
		}
		add("corporate_bond", bond.ID.Hex(), bond.BondName, models.LineInterest, round2(amt))
	}

	stocks, _ := s.stockRepo.GetAll(ctx, userID)
	for _, st := range stocks {
		amt := 0.0
		for _, d := range st.Dividends {
			if inFY(d.Date) {
				amt += d.TotalAmount
			}
		}
		add("stock", st.ID.Hex(), st.Symbol, models.LineDividend, round2(amt))
	}

	if s.rentalRepo != nil {
		rentals, _ := s.rentalRepo.GetAll(ctx, userID)
		for _, r := range rentals {
			amt := 0.0
			for _, p := range r.Payouts {
				if !inFY(p.ScheduledDate) {
					continue
				}
				if p.ReceivedAmount > 0 {
					amt += p.ReceivedAmount
				} else {
					amt += p.Amount
				}
			}
			add("vehicle_rental", r.ID.Hex(), strings.TrimSpace(r.Provider+" "+r.VehicleModel), models.LineOtherRental, round2(amt))
		}
	}

	cg := s.capitalGains(fyStart, fyEnd, mfs, stocks)
	for _, e := range cg.Entries {
		kind := models.LineGainSlab
		if e.IsEquity && e.IsLongTerm {
			kind = models.LineLTCGEquity
		} else if e.IsEquity {
			kind = models.LineSTCGEquity
		}
		add(e.InvestmentType, e.InvestmentID, e.InvestmentName, kind, e.Gain)
	}
	return lines, cg, fyLabel(fyStart), nil
}

// SummaryFromLines builds the deduction / income summary from tax lines
// (household-wide or one member's attributed lines).
func SummaryFromLines(fy string, lines []models.TaxLine, cg *models.CapitalGainsSummary) *models.TaxSummary {
	sum := &models.TaxSummary{
		FY:         fy,
		Section80C: models.Section80CBreakdown{Limit: 150000},
		Section80D: models.Section80DBreakdown{Limit: 25000, ParentsLimit: 25000},
		Section24b: models.Section24bBreakdown{Limit: 200000},
	}
	var letOutInterest float64
	for _, l := range lines {
		switch l.Kind {
		case models.Line80CEPF:
			sum.Section80C.EPFContribution += l.Amount
		case models.Line80CPPF:
			sum.Section80C.PPFContribution += l.Amount
		case models.Line80CELSS:
			sum.Section80C.ELSSInvestment += l.Amount
		case models.Line80CPrincipal:
			sum.Section80C.HomeLoanPrincipal += l.Amount
		case models.Line80CLife:
			sum.Section80C.LifeInsurance += l.Amount
		case models.Line80DSelf:
			sum.Section80D.HealthInsurance += l.Amount
			if l.Senior {
				sum.Section80D.Limit = 50000
			}
		case models.Line80DParents:
			sum.Section80D.Parents += l.Amount
			if l.Senior {
				sum.Section80D.ParentsLimit = 50000
			}
		case models.LineNPSOwn:
			sum.Section80CCD.NPS80CCD1 += l.Amount
		case models.LineNPSEmployer:
			sum.Section80CCD.NPS80CCD2 += l.Amount
		case models.Line24bSelfOccupied:
			sum.Section24b.HomeLoanInterest += l.Amount
		case models.Line24bPreEMI:
			sum.Section24b.PreEMIInterest += l.Amount
		case models.Line24bLetOut:
			letOutInterest += l.Amount
		case models.LineInterest:
			sum.InterestIncome += l.Amount
		case models.LineDividend:
			sum.DividendIncome += l.Amount
		case models.LineRent:
			sum.RentalIncome += l.Amount
		case models.LineOtherRental:
			sum.OtherRentalIncome += l.Amount
		}
	}
	c := &sum.Section80C
	c.Total = round2(c.EPFContribution + c.PPFContribution + c.ELSSInvestment + c.HomeLoanPrincipal + c.LifeInsurance)
	c.Deduction = math.Min(c.Total, c.Limit)

	d := &sum.Section80D
	d.Deduction = round2(math.Min(d.HealthInsurance, d.Limit) + math.Min(d.Parents, d.ParentsLimit))

	sum.Section80CCD.NPS80CCD1B = math.Min(sum.Section80CCD.NPS80CCD1, 50000)

	b := &sum.Section24b
	selfOcc := b.HomeLoanInterest + b.PreEMIInterest
	b.HomeLoanInterest += letOutInterest
	b.Total = round2(b.HomeLoanInterest + b.PreEMIInterest)
	// Let-out interest is fully deductible against rent; self-occupied is capped.
	b.Deduction = round2(math.Min(selfOcc, b.Limit) + letOutInterest)

	sum.InterestIncome = round2(sum.InterestIncome)
	sum.DividendIncome = round2(sum.DividendIncome)
	sum.RentalIncome = round2(sum.RentalIncome)
	sum.OtherRentalIncome = round2(sum.OtherRentalIncome)

	if cg != nil {
		sum.CapitalGains = *cg
		sum.TotalTaxableGains = round2(math.Max(0, cg.STCG) + math.Max(0, cg.LTCG-cg.LTCGExemption) + math.Max(0, cg.SlabGains))
	}
	sum.TotalDeductions = round2(c.Deduction + sum.Section80CCD.NPS80CCD1B + sum.Section80CCD.NPS80CCD2 + b.Deduction + d.Deduction)
	return sum
}

// GetCapitalGains calculates capital gains from MF redemptions and stock sales
// Uses FIFO cost basis, proper holding periods, and grandfathering for pre-31-Jan-2018 equity.
func (s *TaxService) GetCapitalGains(ctx context.Context, userID string, fy string) (*models.CapitalGainsSummary, error) {
	fyStart, fyEnd := parseFYRange(fy)
	mfs, _ := s.mfRepo.GetAll(ctx, userID)
	stocks, _ := s.stockRepo.GetAll(ctx, userID)
	return s.capitalGains(fyStart, fyEnd, mfs, stocks), nil
}

func isEquityFund(ft models.FundType) bool {
	return ft == models.FundTypeEquity || ft == models.FundTypeELSS || ft == models.FundTypeIndex || ft == models.FundTypeHybrid ||
		ft == models.FundTypeSmallCap || ft == models.FundTypeMidCap || ft == models.FundTypeLargeCap ||
		ft == models.FundTypeMultiCap || ft == models.FundTypeFlexiCap || ft == models.FundTypeSectoral || ft == models.FundTypeThematic
}

func (s *TaxService) capitalGains(fyStart, fyEnd time.Time, mfs []models.MutualFund, stocks []models.Stock) *models.CapitalGainsSummary {
	// Grandfathering cutoff date for equity LTCG
	grandfatherDate := time.Date(2018, 1, 31, 0, 0, 0, 0, time.UTC)

	cg := &models.CapitalGainsSummary{
		Entries:        []models.CapitalGainEntry{},
		LTCGExemption:  125000,
		HarvestingTips: []string{},
	}

	// MF redemptions — FIFO based
	for _, mf := range mfs {
		// Build buy lot queue (FIFO)
		type buyLot struct {
			date  time.Time
			units float64
			nav   float64
		}
		var lots []buyLot
		for _, txn := range mf.Transactions {
			if txn.Type == models.TransactionPurchase || txn.Type == models.TransactionSIP || txn.Type == models.TransactionSwitchIn {
				lots = append(lots, buyLot{date: txn.Date, units: txn.Units, nav: txn.NAVAtPurchase})
			}
		}
		sort.SliceStable(lots, func(i, j int) bool { return lots[i].date.Before(lots[j].date) })
		isEquity := isEquityFund(mf.FundType)

		// Process redemptions in this FY
		for _, txn := range mf.Transactions {
			if (txn.Type != models.TransactionRedemption && txn.Type != models.TransactionSwitchOut) || txn.Date.Before(fyStart) || !txn.Date.Before(fyEnd) {
				continue
			}

			sellUnits := math.Abs(txn.Units)
			if sellUnits == 0 {
				continue
			}
			sellNAV := txn.Amount / sellUnits
			remaining := sellUnits

			for remaining > 0 && len(lots) > 0 {
				lot := &lots[0]
				consumed := math.Min(remaining, lot.units)

				buyAmount := consumed * lot.nav
				sellAmount := consumed * sellNAV
				holdingDays := int(txn.Date.Sub(lot.date).Hours() / 24)

				var isLongTerm bool
				var taxRate float64
				if isEquity {
					isLongTerm = holdingDays > 365
					taxRate = 20.0 // STCG on equity
					if isLongTerm {
						// Grandfathered lots would need the 31-Jan-2018 NAV for the cost step-up; flagged only.
						taxRate = 12.5
					}
				} else {
					// Debt MFs bought after 1-Apr-2023: no LTCG benefit, taxed at the slab rate.
					taxRate = 30.0 // indicative top slab; the family tax engine uses each member's slab
				}

				gain := sellAmount - buyAmount
				entry := models.CapitalGainEntry{
					InvestmentType: "mutual_fund",
					InvestmentID:   mf.ID.Hex(),
					InvestmentName: mf.FundName,
					BuyDate:        lot.date.Format("2006-01-02"),
					SellDate:       txn.Date.Format("2006-01-02"),
					BuyAmount:      math.Round(buyAmount*100) / 100,
					SellAmount:     math.Round(sellAmount*100) / 100,
					Gain:           math.Round(gain*100) / 100,
					HoldingDays:    holdingDays,
					IsLongTerm:     isLongTerm,
					TaxRate:        taxRate,
					Grandfathered:  isEquity && lot.date.Before(grandfatherDate),
					IsEquity:       isEquity,
				}
				if gain > 0 {
					entry.TaxLiability = math.Round(gain*taxRate/100*100) / 100
				}
				cg.Entries = append(cg.Entries, entry)

				lot.units -= consumed
				remaining -= consumed
				if lot.units <= 0.001 {
					lots = lots[1:]
				}
			}
		}
	}

	// Stock sales — FIFO based
	for _, stock := range stocks {
		type buyLot struct {
			date  time.Time
			qty   float64
			price float64
		}
		var lots []buyLot
		for _, txn := range stock.Transactions {
			if txn.Type == "buy" {
				lots = append(lots, buyLot{date: txn.Date, qty: float64(txn.Quantity), price: txn.PricePerShare})
			}
		}
		sort.SliceStable(lots, func(i, j int) bool { return lots[i].date.Before(lots[j].date) })

		for _, txn := range stock.Transactions {
			if txn.Type != "sell" || txn.Date.Before(fyStart) || !txn.Date.Before(fyEnd) {
				continue
			}

			remaining := float64(txn.Quantity)
			for remaining > 0 && len(lots) > 0 {
				lot := &lots[0]
				consumed := math.Min(remaining, lot.qty)

				buyAmount := consumed * lot.price
				sellAmount := consumed * txn.PricePerShare
				holdingDays := int(txn.Date.Sub(lot.date).Hours() / 24)
				isLongTerm := holdingDays > 365
				taxRate := 12.5 // LTCG equity
				if !isLongTerm {
					taxRate = 20.0 // STCG equity
				}

				gain := sellAmount - buyAmount
				entry := models.CapitalGainEntry{
					InvestmentType: "stock",
					InvestmentID:   stock.ID.Hex(),
					InvestmentName: stock.Symbol,
					BuyDate:        lot.date.Format("2006-01-02"),
					SellDate:       txn.Date.Format("2006-01-02"),
					BuyAmount:      math.Round(buyAmount*100) / 100,
					SellAmount:     math.Round(sellAmount*100) / 100,
					Gain:           math.Round(gain*100) / 100,
					HoldingDays:    holdingDays,
					IsLongTerm:     isLongTerm,
					TaxRate:        taxRate,
					Grandfathered:  lot.date.Before(grandfatherDate),
					IsEquity:       true,
				}
				if gain > 0 {
					entry.TaxLiability = math.Round(gain*taxRate/100*100) / 100
				}
				cg.Entries = append(cg.Entries, entry)

				lot.qty -= consumed
				remaining -= consumed
				if lot.qty <= 0.001 {
					lots = lots[1:]
				}
			}
		}
	}

	FinalizeCapitalGains(cg)

	// Tax harvesting tips: unrealised equity gains that fit in the remaining exemption
	for _, mf := range mfs {
		if !isEquityFund(mf.FundType) || mf.TotalInvested <= 0 {
			continue
		}
		unrealizedGain := mf.CurrentValue - mf.TotalInvested
		if unrealizedGain > 0 && cg.LTCG < cg.LTCGExemption {
			harvestable := math.Min(unrealizedGain, cg.LTCGExemption-cg.LTCG)
			if harvestable > 5000 {
				cg.HarvestingTips = append(cg.HarvestingTips,
					fmt.Sprintf("You can book ₹%.0f of LTCG in %s tax-free (within ₹1.25L exemption). Redeem and reinvest to reset cost basis.", harvestable, mf.FundName))
			}
		}
	}
	for _, stock := range stocks {
		if stock.TotalInvested <= 0 {
			continue
		}
		unrealizedGain := stock.CurrentValue - stock.TotalInvested
		if unrealizedGain > 0 && cg.LTCG < cg.LTCGExemption {
			harvestable := math.Min(unrealizedGain, cg.LTCGExemption-cg.LTCG)
			if harvestable > 5000 {
				cg.HarvestingTips = append(cg.HarvestingTips,
					fmt.Sprintf("Book ₹%.0f LTCG in %s tax-free. Sell and rebuy to reset cost basis.", harvestable, stock.Symbol))
			}
		}
	}

	// Tip: offset losses against gains
	if cg.STCG < 0 && cg.LTCG > 0 {
		cg.HarvestingTips = append(cg.HarvestingTips,
			fmt.Sprintf("Your STCG loss of ₹%.0f can offset LTCG gains, saving up to ₹%.0f in tax.", -cg.STCG, math.Min(-cg.STCG, cg.LTCG)*12.5/100))
	}
	return cg
}

// FinalizeCapitalGains recomputes totals and tax from cg.Entries. Slab-rate
// gains are reported separately; their tax depends on the member's slab.
func FinalizeCapitalGains(cg *models.CapitalGainsSummary) {
	cg.STCG, cg.LTCG, cg.SlabGains = 0, 0, 0
	for _, e := range cg.Entries {
		switch {
		case !e.IsEquity:
			cg.SlabGains += e.Gain
		case e.IsLongTerm:
			cg.LTCG += e.Gain
		default:
			cg.STCG += e.Gain
		}
	}
	cg.STCG, cg.LTCG, cg.SlabGains = round2(cg.STCG), round2(cg.LTCG), round2(cg.SlabGains)
	cg.LTCGTax = math.Round(math.Max(0, cg.LTCG-cg.LTCGExemption)*12.5) / 100
	cg.STCGTax = math.Round(math.Max(0, cg.STCG)*20) / 100
	cg.TotalTax = round2(cg.LTCGTax + cg.STCGTax)
}
