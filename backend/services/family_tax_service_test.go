package services

import (
	"math"
	"testing"
	"time"

	"investment-tracker/models"
	"investment-tracker/taxengine"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type famFixture struct {
	c                                  *famCtx
	self, spouse, child, parent, adult string
}

func dob(years int) *time.Time {
	t := time.Date(2026-years, 1, 1, 0, 0, 0, 0, time.Local)
	return &t
}

// newFixture: salaried self (₹30L), homemaker spouse, minor child, senior parent, adult child.
func newFixture(t *testing.T) *famFixture {
	t.Helper()
	mk := func(name, rel, occ string, birth *time.Time) models.Profile {
		return models.Profile{ID: primitive.NewObjectID(), Name: name, Relationship: rel, Occupation: occ, DateOfBirth: birth}
	}
	profiles := []models.Profile{
		mk("Asha", "spouse", "homemaker", dob(36)),
		mk("Ravi", "self", "salaried_private", dob(38)),
		mk("Kiara", "child", "minor", dob(8)),
		mk("Mohan", "parent", "pensioner", dob(68)),
		mk("Arjun", "child", "student", dob(20)),
	}
	c := newFamCtx("2025-26")
	c.setProfiles(profiles)
	f := &famFixture{c: c}
	for _, p := range c.profiles {
		switch p.Name {
		case "Ravi":
			f.self = p.ID.Hex()
		case "Asha":
			f.spouse = p.ID.Hex()
		case "Kiara":
			f.child = p.ID.Hex()
		case "Mohan":
			f.parent = p.ID.Hex()
		case "Arjun":
			f.adult = p.ID.Hex()
		}
	}
	months := []string{"2025-04", "2025-05"}
	var slips []models.Payslip
	for _, m := range months {
		slips = append(slips, models.Payslip{Month: m, Basic: 100000, HRA: 40000, SpecialAllowance: 110000, EmployeePF: 12000, TDS: 40000})
	}
	c.setIncomes([]models.MemberIncome{{ProfileID: f.self, FY: "2025-26", Payslips: slips, ProjectRemaining: true}})
	return f
}

func (f *famFixture) assign(assetType, assetID string, fundedBy string, shares ...models.OwnershipShare) {
	f.c.setAssignments(append(f.assignments(), models.ProfileAssignment{AssetType: assetType, AssetID: assetID,
		ProfileID: shares[0].ProfileID, Shares: shares, FundedBy: fundedBy}))
}

func (f *famFixture) assignments() []models.ProfileAssignment {
	var out []models.ProfileAssignment
	for _, a := range f.c.assign {
		out = append(out, *a)
	}
	return out
}

func sumLines(c *famCtx, pid, kind string) float64 {
	t := 0.0
	for _, l := range c.lines {
		if l.ProfileID == pid && l.Kind == kind {
			t += l.Amount
		}
	}
	return t
}

func TestDefaultMemberIsSelf(t *testing.T) {
	f := newFixture(t)
	if f.c.defaultID != f.self {
		t.Fatalf("default member should be the 'self' profile")
	}
}

func TestClubbingRules(t *testing.T) {
	f := newFixture(t)
	c := f.c
	if got, clubbed := c.clubTarget(f.spouse, f.self); got != f.self || !clubbed {
		t.Errorf("income on money given to spouse should be clubbed with the giver")
	}
	if got, clubbed := c.clubTarget(f.spouse, ""); got != f.spouse || clubbed {
		t.Errorf("spouse's own money must not be clubbed")
	}
	if got, clubbed := c.clubTarget(f.child, ""); got != f.self || !clubbed {
		t.Errorf("minor's income should go to the higher-income parent, got %s", c.name(got))
	}
	if got, clubbed := c.clubTarget(f.adult, f.self); got != f.adult || clubbed {
		t.Errorf("gift to an adult child is not clubbed")
	}
	if got, clubbed := c.clubTarget(f.parent, f.self); got != f.parent || clubbed {
		t.Errorf("gift to a parent is not clubbed")
	}
}

func TestAttributionJointLoanAndPayerDeductions(t *testing.T) {
	f := newFixture(t)
	loan, ppfSpouse, ppfParent, fd := primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex()
	f.assign("home_loan", loan, "", models.OwnershipShare{ProfileID: f.self, Pct: 60}, models.OwnershipShare{ProfileID: f.spouse, Pct: 40})
	f.assign("provident_fund", ppfSpouse, f.self, models.OwnershipShare{ProfileID: f.spouse, Pct: 100})
	f.assign("provident_fund", ppfParent, f.self, models.OwnershipShare{ProfileID: f.parent, Pct: 100})
	f.assign("fixed_deposit", fd, f.self, models.OwnershipShare{ProfileID: f.spouse, Pct: 100})

	f.c.ingest([]models.TaxLine{
		{AssetType: "home_loan", AssetID: loan, Name: "HDFC", Kind: models.Line24bSelfOccupied, Amount: 300000},
		{AssetType: "provident_fund", AssetID: ppfSpouse, Name: "PPF Asha", Kind: models.Line80CPPF, Amount: 100000},
		{AssetType: "provident_fund", AssetID: ppfParent, Name: "PPF Mohan", Kind: models.Line80CPPF, Amount: 50000},
		{AssetType: "fixed_deposit", AssetID: fd, Name: "SBI FD", Kind: models.LineInterest, Amount: 70000},
		{AssetType: "mutual_fund", AssetID: primitive.NewObjectID().Hex(), Name: "Unassigned fund", Kind: models.LineDividend, Amount: 1000},
	}, nil)

	c := f.c
	if got := sumLines(c, f.self, models.Line24bSelfOccupied); got != 180000 {
		t.Errorf("self share of joint loan interest = %v, want 180000", got)
	}
	if got := sumLines(c, f.spouse, models.Line24bSelfOccupied); got != 120000 {
		t.Errorf("spouse share of joint loan interest = %v, want 120000", got)
	}
	if got := sumLines(c, f.self, models.Line80CPPF); got != 100000 {
		t.Errorf("payer should claim PPF in spouse's name, got %v", got)
	}
	if got := sumLines(c, f.parent, models.Line80CPPF) + sumLines(c, f.self, models.Line80CPPF); got != 100000 {
		t.Errorf("PPF in a parent's name paid by a child must not qualify, got total %v", got)
	}
	if len(c.warnings[f.self]) == 0 {
		t.Errorf("expected a warning about the parent's PPF")
	}
	if got := sumLines(c, f.self, models.LineInterest); got != 70000 {
		t.Errorf("FD interest in spouse's name funded by self should be clubbed to self, got %v", got)
	}
	if !c.unassigned["mutual_fund:"+c.lines[len(c.lines)-1].AssetID] || sumLines(c, f.self, models.LineDividend) != 1000 {
		t.Errorf("unassigned items should go to the default member and be counted")
	}
}

func TestMemberInputProjectionAndEPF(t *testing.T) {
	f := newFixture(t)
	in, inc, _ := f.c.memberInput(f.self)
	if inc.ActualMonths != 2 || inc.ProjectedMonths != 10 {
		t.Fatalf("months actual=%d projected=%d", inc.ActualMonths, inc.ProjectedMonths)
	}
	if in.GrossSalary != 250000*12 {
		t.Errorf("gross salary %v want 3000000", in.GrossSalary)
	}
	if in.Ded80C != 12000*12 {
		t.Errorf("EPF from payslips %v want 144000", in.Ded80C)
	}
	// EPF also tracked in the PF module: take the larger figure, don't double count.
	pf := primitive.NewObjectID().Hex()
	f.assign("provident_fund", pf, "", models.OwnershipShare{ProfileID: f.self, Pct: 100})
	f.c.ingest([]models.TaxLine{{AssetType: "provident_fund", AssetID: pf, Kind: models.Line80CEPF, Amount: 150000}}, nil)
	in, _, _ = f.c.memberInput(f.self)
	if in.Ded80C != 150000 {
		t.Errorf("EPF should be max(payslip, module) = 150000, got %v", in.Ded80C)
	}
	if in.HRAExempt != 0 {
		t.Errorf("no rent paid, HRA exemption should be 0")
	}
}

func TestMinorClubbingExemption(t *testing.T) {
	f := newFixture(t)
	fd := primitive.NewObjectID().Hex()
	f.assign("fixed_deposit", fd, "", models.OwnershipShare{ProfileID: f.child, Pct: 100})
	f.c.ingest([]models.TaxLine{{AssetType: "fixed_deposit", AssetID: fd, Kind: models.LineInterest, Amount: 20000}}, nil)
	in, inc, _ := f.c.memberInput(f.self)
	if in.InterestIncome != 20000 || inc.ClubbedIncome != 20000 {
		t.Fatalf("minor's FD interest should be clubbed with the parent")
	}
	if in.MinorExemption != 1500 {
		t.Errorf("minor exemption = %v, want 1500", in.MinorExemption)
	}
}

func TestPlacementPrefersUnclubbedLowIncomeMember(t *testing.T) {
	f := newFixture(t)
	res := f.c.placement(models.PlacementRequest{FY: "2025-26", Amount: 1000000, Instrument: "fd", ExpectedReturnPct: 7, FundedBy: f.self})
	if len(res.Options) != 5 {
		t.Fatalf("expected an option per member, got %d", len(res.Options))
	}
	byID := map[string]models.PlacementOption{}
	for _, o := range res.Options {
		byID[o.ProfileID] = o
	}
	if byID[f.spouse].TaxpayerID != f.self || byID[f.child].TaxpayerID != f.self {
		t.Errorf("spouse and minor options should be clubbed with self")
	}
	if byID[f.parent].NetTaxImpact != 0 || byID[f.adult].NetTaxImpact != 0 {
		t.Errorf("parent/adult child with no income should pay no tax on ₹70k interest: %+v %+v", byID[f.parent], byID[f.adult])
	}
	if byID[f.self].NetTaxImpact <= 20000 {
		t.Errorf("self at 30%% slab should pay >₹20k on ₹70k interest, got %v", byID[f.self].NetTaxImpact)
	}
	if res.BestProfile != f.parent && res.BestProfile != f.adult {
		t.Errorf("best placement should be parent or adult child, got %s", f.c.name(res.BestProfile))
	}

	scss := f.c.placement(models.PlacementRequest{Amount: 500000, Instrument: "scss", ExpectedReturnPct: 8.2})
	for _, o := range scss.Options {
		if o.Eligible != (o.ProfileID == f.parent) {
			t.Errorf("SCSS eligibility wrong for %s", o.Name)
		}
	}
}

func TestMemberReportRegimeAndBalance(t *testing.T) {
	f := newFixture(t)
	r := f.c.memberReport(f.self)
	best := math.Min(r.Comparison.Old.TotalTax, r.Comparison.New.TotalTax)
	if r.ChosenTax != best || r.ExtraTaxVsBest != 0 {
		t.Errorf("without a preference the chosen regime should be the cheapest")
	}
	if r.TaxPaid != 40000*12 {
		t.Errorf("TDS incl. projected months = %v, want 480000", r.TaxPaid)
	}
	if r.BalancePayable != r.ChosenTax-r.TaxPaid {
		t.Errorf("balance payable mismatch")
	}
	f.c.byID[f.self].PreferredRegime = string(taxengine.Old)
	delete(f.c.reports, f.self)
	r = f.c.memberReport(f.self)
	if r.ChosenRegime != taxengine.Old || r.ExtraTaxVsBest != r.Comparison.Old.TotalTax-best {
		t.Errorf("preferred regime should be honoured and its extra cost reported")
	}
}

func TestClubbingToggles(t *testing.T) {
	f := newFixture(t)
	f.c.settings = models.TaxSettings{IgnoreSpouseClubbing: true}
	if got, clubbed := f.c.clubTarget(f.spouse, f.self); got != f.spouse || clubbed {
		t.Errorf("with spouse clubbing off, income stays with the spouse")
	}
	if got, _ := f.c.clubTarget(f.child, ""); got != f.self {
		t.Errorf("minor clubbing should still apply when only spouse clubbing is off")
	}

	fd, kidFD := primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex()
	f.c.settings.IgnoreMinorClubbing = true
	f.assign("fixed_deposit", fd, f.self, models.OwnershipShare{ProfileID: f.spouse, Pct: 100})
	f.assign("fixed_deposit", kidFD, "", models.OwnershipShare{ProfileID: f.child, Pct: 100})
	f.c.ingest([]models.TaxLine{
		{AssetType: "fixed_deposit", AssetID: fd, Kind: models.LineInterest, Amount: 70000},
		{AssetType: "fixed_deposit", AssetID: kidFD, Kind: models.LineInterest, Amount: 20000},
	}, nil)
	if sumLines(f.c, f.spouse, models.LineInterest) != 70000 || sumLines(f.c, f.child, models.LineInterest) != 20000 {
		t.Fatalf("income should stay with spouse and child when clubbing is off")
	}
	if f.c.unclubbed[f.spouse][f.self] != 70000 || f.c.unclubbed[f.child][f.self] != 20000 {
		t.Errorf("unclubbed income not tracked: %+v", f.c.unclubbed)
	}
	in, inc, _ := f.c.memberInput(f.self)
	if in.InterestIncome != 0 || inc.ClubbedIncome != 0 || in.MinorExemption != 0 {
		t.Errorf("self should have no clubbed income")
	}
	if r := f.c.memberReport(f.spouse); len(r.Warnings) == 0 {
		t.Errorf("spouse report should warn that clubbing is off")
	}

	res := f.c.placement(models.PlacementRequest{Amount: 1000000, Instrument: "fd", ExpectedReturnPct: 7, FundedBy: f.self})
	for _, o := range res.Options {
		if o.ProfileID == f.spouse && (o.TaxpayerID != f.spouse || o.NetTaxImpact != 0) {
			t.Errorf("spouse FD should be taxed (at nil) in spouse's hands, got %+v", o)
		}
	}
}

func TestNormalizeShares(t *testing.T) {
	out := normalizeShares([]models.OwnershipShare{{ProfileID: "a", Pct: 1}, {ProfileID: "b", Pct: 3}})
	if out[0].ProfileID != "b" || out[0].Pct != 75 || out[1].Pct != 25 {
		t.Fatalf("unexpected %+v", out)
	}
}
