package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
	"investment-tracker/taxengine"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const taxDisclaimer = "Estimates for planning, based on the data in this app and the rules for the selected financial year. " +
	"Not tax advice — confirm with a chartered accountant before filing or restructuring investments."

// FamilyTaxService computes each family member's tax under both regimes from
// their income records and the assets attributed to them, and recommends
// regimes and tax-efficient placement of new investments.
type FamilyTaxService struct {
	tax          *TaxService
	profileRepo  *repository.ProfileRepo
	assignRepo   *repository.ProfileAssignmentRepo
	incomeRepo   *repository.MemberIncomeRepo
	settingsRepo *repository.TaxSettingsRepo
}

func NewFamilyTaxService(tax *TaxService, profileRepo *repository.ProfileRepo, assignRepo *repository.ProfileAssignmentRepo, incomeRepo *repository.MemberIncomeRepo, settingsRepo *repository.TaxSettingsRepo) *FamilyTaxService {
	return &FamilyTaxService{tax: tax, profileRepo: profileRepo, assignRepo: assignRepo, incomeRepo: incomeRepo, settingsRepo: settingsRepo}
}

// GetSettings returns the household's tax settings (defaults if never saved).
func (s *FamilyTaxService) GetSettings(ctx context.Context, userID string) (models.TaxSettings, error) {
	if s.settingsRepo == nil {
		return models.TaxSettings{}, nil
	}
	st, err := s.settingsRepo.FindOne(ctx, bson.M{"user_id": userID})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.TaxSettings{}, nil
	}
	if err != nil {
		return models.TaxSettings{}, err
	}
	return *st, nil
}

func (s *FamilyTaxService) SaveSettings(ctx context.Context, userID string, req models.TaxSettings) (models.TaxSettings, error) {
	err := s.settingsRepo.Upsert(ctx, bson.M{"user_id": userID}, bson.M{"user_id": userID,
		"ignore_spouse_clubbing": req.IgnoreSpouseClubbing, "ignore_minor_clubbing": req.IgnoreMinorClubbing})
	if err != nil {
		return models.TaxSettings{}, err
	}
	return s.GetSettings(ctx, userID)
}

func resolveFY(fy string) (string, time.Time, time.Time) {
	n := taxengine.NormalizeFY(fy)
	if n == "" {
		n = fyLabel(getCurrentFYStart(time.Now()))
	}
	start, end := parseFYRange(n)
	return n, start, end
}

func (s *FamilyTaxService) checkMember(ctx context.Context, profileID string) (*models.Profile, error) {
	oid, err := parseObjectID(profileID)
	if err != nil {
		return nil, invalidf("invalid profile id")
	}
	p, err := s.profileRepo.GetByID(ctx, oid)
	if err != nil {
		return nil, notFoundf("family member not found")
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// Income records

func (s *FamilyTaxService) ListIncomes(ctx context.Context, userID, fy string) ([]models.MemberIncome, error) {
	n, _, _ := resolveFY(fy)
	return s.incomeRepo.Find(ctx, userID, bson.M{"fy": n})
}

// GetIncome returns the member's record for the FY, or an empty template.
func (s *FamilyTaxService) GetIncome(ctx context.Context, userID, profileID, fy string) (*models.MemberIncome, error) {
	n, _, _ := resolveFY(fy)
	if _, err := s.checkMember(ctx, profileID); err != nil {
		return nil, err
	}
	rec, err := s.incomeRepo.FindOne(ctx, bson.M{"user_id": userID, "profile_id": profileID, "fy": n})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &models.MemberIncome{UserID: userID, ProfileID: profileID, FY: n, Payslips: []models.Payslip{}, ProjectRemaining: true}, nil
	}
	if err != nil {
		return nil, err
	}
	if rec.Payslips == nil {
		rec.Payslips = []models.Payslip{}
	}
	return rec, nil
}

func checkNonNegative(v any) error {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		if f.Kind() != reflect.Float64 {
			continue
		}
		x := f.Float()
		if x < 0 || math.IsNaN(x) || math.IsInf(x, 0) {
			name := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
			return invalidf("%s must be a non-negative amount", name)
		}
	}
	return nil
}

func (s *FamilyTaxService) SaveIncome(ctx context.Context, userID, profileID, fy string, req models.MemberIncomeRequest) (*models.MemberIncome, error) {
	n, start, end := resolveFY(fy)
	if _, err := s.checkMember(ctx, profileID); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	payslips := make([]models.Payslip, 0, len(req.Payslips))
	for _, p := range req.Payslips {
		t, err := time.ParseInLocation("2006-01", strings.TrimSpace(p.Month), time.Local)
		if err != nil {
			return nil, invalidf("payslip month %q must be YYYY-MM", p.Month)
		}
		if t.Before(start) || !t.Before(end) {
			return nil, invalidf("payslip month %s is outside FY %s", p.Month, n)
		}
		p.Month = t.Format("2006-01")
		if seen[p.Month] {
			return nil, invalidf("duplicate payslip for %s", p.Month)
		}
		seen[p.Month] = true
		if err := checkNonNegative(p); err != nil {
			return nil, err
		}
		p.Employer = strings.TrimSpace(p.Employer)
		payslips = append(payslips, p)
	}
	sort.Slice(payslips, func(i, j int) bool { return payslips[i].Month < payslips[j].Month })
	switch req.Other.BusinessScheme {
	case "", "actual", "44AD", "44ADA":
	default:
		return nil, invalidf("business_scheme must be actual, 44AD or 44ADA")
	}
	if err := checkNonNegative(req.Other); err != nil {
		return nil, err
	}
	if err := checkNonNegative(req.Deductions); err != nil {
		return nil, err
	}

	now := time.Now()
	rec := models.MemberIncome{UserID: userID, ProfileID: profileID, FY: n, Payslips: payslips, ProjectRemaining: req.ProjectRemaining,
		Other: req.Other, Deductions: req.Deductions, Notes: strings.TrimSpace(req.Notes), UpdatedAt: now}
	existing, err := s.incomeRepo.FindOne(ctx, bson.M{"user_id": userID, "profile_id": profileID, "fy": n})
	switch {
	case err == nil:
		rec.ID, rec.CreatedAt = existing.ID, existing.CreatedAt
		if err := s.incomeRepo.Replace(ctx, existing.ID, &rec); err != nil {
			return nil, err
		}
	case errors.Is(err, mongo.ErrNoDocuments):
		rec.CreatedAt = now
		id, err := s.incomeRepo.Insert(ctx, &rec)
		if err != nil {
			return nil, err
		}
		rec.ID = id
	default:
		return nil, err
	}
	return &rec, nil
}

func (s *FamilyTaxService) DeleteIncome(ctx context.Context, userID, profileID, fy string) error {
	n, _, _ := resolveFY(fy)
	return s.incomeRepo.DeleteMany(ctx, bson.M{"user_id": userID, "profile_id": profileID, "fy": n})
}

// ---------------------------------------------------------------------------
// Household context: profiles, incomes and attributed tax lines

type attributedGain struct {
	taxpayer string
	entry    models.CapitalGainEntry
}

type famCtx struct {
	fy      string
	rules   *taxengine.Rules
	rulesOK bool
	fyStart time.Time
	fyEnd   time.Time

	profiles   []models.Profile
	byID       map[string]*models.Profile
	defaultID  string
	incomes    map[string]*models.MemberIncome
	assign     map[string]*models.ProfileAssignment
	baseIncome map[string]float64

	lines      []models.TaxLine
	gains      []attributedGain
	unassigned map[string]bool
	warnings   map[string][]string
	settings   models.TaxSettings
	// unclubbed[owner][giver] is income kept with owner only because clubbing is off.
	unclubbed map[string]map[string]float64

	inputs  map[string]taxengine.Input
	reports map[string]*models.MemberTaxReport
}

var (
	incomeKinds = map[string]bool{models.LineInterest: true, models.LineDividend: true, models.LineSTCGEquity: true,
		models.LineLTCGEquity: true, models.LineGainSlab: true, models.LineRent: true, models.LineOtherRental: true}
	// Deductions the payer may claim for a policy / deposit in a family member's name.
	payerKinds = map[string]bool{models.Line80CPPF: true, models.Line80CLife: true, models.Line80DSelf: true, models.Line80DParents: true}
)

func newFamCtx(fy string) *famCtx {
	n, start, end := resolveFY(fy)
	rules, ok := taxengine.RulesFor(n)
	return &famCtx{fy: n, rules: rules, rulesOK: ok, fyStart: start, fyEnd: end,
		byID: map[string]*models.Profile{}, incomes: map[string]*models.MemberIncome{}, assign: map[string]*models.ProfileAssignment{},
		baseIncome: map[string]float64{}, unassigned: map[string]bool{}, warnings: map[string][]string{},
		unclubbed: map[string]map[string]float64{},
		inputs:    map[string]taxengine.Input{}, reports: map[string]*models.MemberTaxReport{}}
}

func (s *FamilyTaxService) load(ctx context.Context, userID, fy string) (*famCtx, error) {
	c := newFamCtx(fy)
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	c.settings = settings
	profiles, err := s.profileRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	c.setProfiles(profiles)
	if len(profiles) == 0 {
		return c, nil
	}
	incomes, err := s.incomeRepo.Find(ctx, userID, bson.M{"fy": c.fy})
	if err != nil {
		return nil, err
	}
	assignments, err := s.assignRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	c.setIncomes(incomes)
	c.setAssignments(assignments)

	lines, cg, _, err := s.tax.CollectLines(ctx, userID, c.fy)
	if err != nil {
		return nil, err
	}
	c.ingest(lines, cg)
	return c, nil
}

func (c *famCtx) setProfiles(profiles []models.Profile) {
	sort.SliceStable(profiles, func(i, j int) bool {
		ri, rj := profileRank(&profiles[i]), profileRank(&profiles[j])
		if ri != rj {
			return ri < rj
		}
		return profiles[i].CreatedAt.Before(profiles[j].CreatedAt)
	})
	c.profiles = profiles
	for i := range c.profiles {
		p := &c.profiles[i]
		p.Age = p.AgeOn(c.ageDate())
		c.byID[p.ID.Hex()] = p
	}
	if len(profiles) > 0 {
		c.defaultID = c.profiles[0].ID.Hex()
	}
}

func (c *famCtx) setIncomes(incomes []models.MemberIncome) {
	for i := range incomes {
		rec := &incomes[i]
		if c.byID[rec.ProfileID] == nil {
			continue
		}
		c.incomes[rec.ProfileID] = rec
		total := rec.Other.Pension + businessProfit(rec.Other)
		for _, p := range c.payslips(rec) {
			total += p.Gross()
		}
		c.baseIncome[rec.ProfileID] = total
	}
}

func (c *famCtx) setAssignments(assignments []models.ProfileAssignment) {
	for i := range assignments {
		a := &assignments[i]
		c.assign[a.AssetType+":"+a.AssetID] = a
	}
}

// ingest attributes household tax lines and capital gains to members.
func (c *famCtx) ingest(lines []models.TaxLine, cg *models.CapitalGainsSummary) {
	for _, l := range lines {
		c.lines = append(c.lines, c.attributeLine(l)...)
	}
	if cg == nil {
		return
	}
	for _, e := range cg.Entries {
		shares, funder, _ := c.owners(e.InvestmentType, e.InvestmentID)
		for _, sh := range shares {
			taxpayer, _ := c.clubTarget(sh.ProfileID, funder)
			ne := e
			f := sh.Pct / 100
			ne.BuyAmount, ne.SellAmount, ne.Gain, ne.TaxLiability = round2(e.BuyAmount*f), round2(e.SellAmount*f), round2(e.Gain*f), round2(e.TaxLiability*f)
			c.gains = append(c.gains, attributedGain{taxpayer: taxpayer, entry: ne})
		}
	}
}

// profileRank orders the default / self member first.
func profileRank(p *models.Profile) int {
	switch {
	case p.IsDefault:
		return 0
	case p.Relationship == "self":
		return 1
	case p.Relationship == "spouse":
		return 2
	default:
		return 3
	}
}

// ageDate is the last day of the FY; age-based benefits apply if the
// threshold is reached at any time during the year.
func (c *famCtx) ageDate() time.Time { return c.fyEnd.AddDate(0, 0, -1) }

func (c *famCtx) name(id string) string {
	if p := c.byID[id]; p != nil {
		return p.Name
	}
	return "Unknown"
}

func (c *famCtx) warn(id, msg string) {
	for _, w := range c.warnings[id] {
		if w == msg {
			return
		}
	}
	c.warnings[id] = append(c.warnings[id], msg)
}

func (c *famCtx) owners(assetType, assetID string) ([]models.OwnershipShare, string, bool) {
	var shares []models.OwnershipShare
	funder := ""
	if a := c.assign[assetType+":"+assetID]; a != nil {
		for _, sh := range a.EffectiveShares() {
			if c.byID[sh.ProfileID] != nil {
				shares = append(shares, sh)
			}
		}
		if c.byID[a.FundedBy] != nil {
			funder = a.FundedBy
		}
	}
	if len(shares) == 0 {
		return []models.OwnershipShare{{ProfileID: c.defaultID, Pct: 100}}, funder, false
	}
	return normalizeShares(shares), funder, true
}

func (c *famCtx) spouses(a, b string) bool {
	pa, pb := c.byID[a], c.byID[b]
	if pa == nil || pb == nil || a == b {
		return false
	}
	if pa.SpouseProfileID == b || pb.SpouseProfileID == a {
		return true
	}
	if pa.SpouseProfileID != "" || pb.SpouseProfileID != "" {
		return false
	}
	pair := map[string]bool{pa.Relationship: true, pb.Relationship: true}
	return pair["self"] && pair["spouse"]
}

func (c *famCtx) isMinor(p *models.Profile) bool {
	if p.Occupation == "minor" {
		return true
	}
	return p.DateOfBirth != nil && p.AgeOn(c.ageDate()) < 18
}

// parentsOf lists members who are parents of p, based on relationships to the
// account holder ("self").
func (c *famCtx) parentsOf(p *models.Profile) []string {
	var rels map[string]bool
	switch p.Relationship {
	case "child":
		rels = map[string]bool{"self": true, "spouse": true}
	case "self", "spouse", "sibling":
		rels = map[string]bool{"parent": true}
	}
	var ids []string
	for i := range c.profiles {
		q := &c.profiles[i]
		if q.ID != p.ID && rels[q.Relationship] {
			ids = append(ids, q.ID.Hex())
		}
	}
	return ids
}

// clubRule returns who the law taxes on income from an asset owned by owner
// and paid for by funder (Sec 64 clubbing): a minor's income goes to the parent
// with the higher income; income from money given to a spouse stays with the
// giver. Gifts to adult children, parents and siblings are not clubbed.
// kind is "minor" or "spouse" when clubbing applies.
func (c *famCtx) clubRule(owner, funder string) (target string, kind string) {
	p := c.byID[owner]
	if p == nil {
		return owner, ""
	}
	if c.isMinor(p) && p.Disability == "" {
		parents := c.parentsOf(p)
		if len(parents) == 0 {
			if funder != "" && funder != owner {
				return funder, "minor"
			}
			return owner, ""
		}
		best := parents[0]
		for _, id := range parents[1:] {
			if c.baseIncome[id] > c.baseIncome[best] {
				best = id
			}
		}
		return best, "minor"
	}
	if funder != "" && funder != owner && c.spouses(owner, funder) {
		return funder, "spouse"
	}
	return owner, ""
}

// clubTarget applies clubRule unless the household turned that rule off.
func (c *famCtx) clubTarget(owner, funder string) (string, bool) {
	target, kind := c.clubRule(owner, funder)
	if kind == "" || c.clubbingOff(kind) {
		return owner, false
	}
	return target, true
}

func (c *famCtx) clubbingOff(kind string) bool {
	return (kind == "spouse" && c.settings.IgnoreSpouseClubbing) || (kind == "minor" && c.settings.IgnoreMinorClubbing)
}

// payerMayClaim reports whether payer can claim 80C for a PPF / life policy in
// owner's name (self, spouse or children only).
func (c *famCtx) payerMayClaim(payer, owner string) bool {
	if payer == owner || c.spouses(payer, owner) {
		return true
	}
	if p := c.byID[owner]; p != nil {
		for _, id := range c.parentsOf(p) {
			if id == payer {
				return true
			}
		}
	}
	return false
}

func (c *famCtx) attributeLine(l models.TaxLine) []models.TaxLine {
	shares, funder, assigned := c.owners(l.AssetType, l.AssetID)
	if !assigned {
		c.unassigned[l.AssetType+":"+l.AssetID] = true
	}
	var out []models.TaxLine
	for _, sh := range shares {
		nl := l
		nl.Amount = round2(l.Amount * sh.Pct / 100)
		nl.SharePct = sh.Pct
		nl.OwnerID = sh.ProfileID
		target := sh.ProfileID
		switch {
		case incomeKinds[l.Kind]:
			target, nl.Clubbed = c.clubTarget(sh.ProfileID, funder)
			if lawTarget, kind := c.clubRule(sh.ProfileID, funder); kind != "" && !nl.Clubbed {
				nl.ClubbingIgnored = true
				if c.unclubbed[sh.ProfileID] == nil {
					c.unclubbed[sh.ProfileID] = map[string]float64{}
				}
				c.unclubbed[sh.ProfileID][lawTarget] += nl.Amount
			}
		case payerKinds[l.Kind] && funder != "" && funder != sh.ProfileID:
			if l.Kind == models.Line80DSelf || l.Kind == models.Line80DParents || c.payerMayClaim(funder, sh.ProfileID) {
				target = funder
			} else {
				c.warn(funder, fmt.Sprintf("%s: 80C covers only self, spouse and children, so payments for %s don't qualify.", l.Name, c.name(sh.ProfileID)))
				continue
			}
		}
		nl.ProfileID = target
		out = append(out, nl)
	}
	return out
}

// payslips returns actual payslips plus, when enabled, projected months after
// the latest payslip using its fixed pay (variable pay is not projected).
func (c *famCtx) payslips(rec *models.MemberIncome) []models.Payslip {
	out := append([]models.Payslip(nil), rec.Payslips...)
	if !rec.ProjectRemaining || len(rec.Payslips) == 0 {
		return out
	}
	latest := rec.Payslips[len(rec.Payslips)-1]
	last, err := time.ParseInLocation("2006-01", latest.Month, time.Local)
	if err != nil {
		return out
	}
	for m := last.AddDate(0, 1, 0); m.Before(c.fyEnd); m = m.AddDate(0, 1, 0) {
		p := latest
		p.Month, p.Projected = m.Format("2006-01"), true
		p.Bonus, p.Incentives, p.ESOPPerquisite, p.OtherVariable = 0, 0, 0, 0
		out = append(out, p)
	}
	return out
}

func businessProfit(o models.OtherIncome) float64 {
	switch o.BusinessScheme {
	case "44ADA":
		return 0.5 * o.BusinessReceipts
	case "44AD":
		return 0.06 * o.BusinessReceipts // digital receipts; 8% for cash
	default:
		return o.BusinessProfit
	}
}

// memberInput assembles the engine input for one member.
func (c *famCtx) memberInput(pid string) (taxengine.Input, models.IncomeBreakdown, []models.TaxLine) {
	p := c.byID[pid]
	in := taxengine.Input{FY: c.fy, Age: p.AgeOn(c.ageDate()), Resident: p.IsResident(), Disability: p.Disability}
	var inc models.IncomeBreakdown
	var payslipPF, payslipEmpNPS float64

	if rec := c.incomes[pid]; rec != nil {
		for _, ps := range c.payslips(rec) {
			if ps.Projected {
				inc.ProjectedMonths++
			} else {
				inc.ActualMonths++
			}
			inc.SalaryFixed += ps.Fixed()
			inc.SalaryVariable += ps.Variable()
			in.GrossSalary += ps.Fixed() + ps.Variable() + ps.EmployerNPS
			basic := ps.Basic + ps.DA
			in.BasicDA += basic
			in.HRAReceived += ps.HRA
			in.HRAExempt += taxengine.HRAExemption(basic, ps.HRA, ps.RentPaid, p.MetroCity)
			in.RentPaid += ps.RentPaid
			in.ProfessionalTax += ps.ProfessionalTax
			payslipPF += ps.EmployeePF
			payslipEmpNPS += ps.EmployerNPS
		}
		o, d := rec.Other, rec.Deductions
		in.Pension, in.FamilyPension = o.Pension, o.FamilyPension
		in.BusinessIncome = businessProfit(o)
		in.SelfEmployed = in.BusinessIncome > 0 && in.GrossSalary == 0
		in.SavingsInterest = o.SavingsInterest
		in.InterestIncome += o.OtherInterest
		in.DividendIncome += o.Dividends
		in.LetOutRent += o.RentalIncome
		in.LetOutMunicipalTax += o.MunicipalTax
		in.OtherIncome += o.OtherIncome
		in.STCGEquity += o.STCGEquity
		in.LTCGEquity += o.LTCGEquity
		in.SlabCapitalGains += o.OtherGainsSlab

		in.Ded80C += d.Sec80COther
		in.Ded80DSelf += d.Sec80DSelf
		in.Ded80DParents += d.Sec80DParents
		in.ParentsSenior = d.ParentsSenior
		in.Ded80E, in.Ded80G = d.Sec80E, d.Sec80G
		in.OwnNPS += d.NPSOther
		in.LTAExempt = d.LTAExempt
		in.SelfOccupiedInterest += d.HomeLoanInterest
		inc.Pension = o.Pension + o.FamilyPension
		inc.Business = in.BusinessIncome
	}
	if p.Occupation == "professional" || p.Occupation == "business" {
		in.SelfEmployed = in.GrossSalary == 0
	}

	var lines []models.TaxLine
	var moduleEPF, moduleEmpNPS float64
	minorClubbed := map[string]float64{}
	for _, l := range c.lines {
		if l.ProfileID != pid {
			continue
		}
		lines = append(lines, l)
		switch l.Kind {
		case models.LineInterest:
			in.InterestIncome += l.Amount
		case models.LineDividend:
			in.DividendIncome += l.Amount
		case models.LineSTCGEquity:
			in.STCGEquity += l.Amount
		case models.LineLTCGEquity:
			in.LTCGEquity += l.Amount
		case models.LineGainSlab:
			in.SlabCapitalGains += l.Amount
		case models.LineRent:
			in.LetOutRent += l.Amount
		case models.LineOtherRental:
			in.OtherIncome += l.Amount
		case models.Line80CEPF:
			moduleEPF += l.Amount
		case models.Line80CPPF, models.Line80CELSS, models.Line80CPrincipal, models.Line80CLife:
			in.Ded80C += l.Amount
		case models.Line80DSelf:
			in.Ded80DSelf += l.Amount
			in.SelfSeniorHealth = in.SelfSeniorHealth || l.Senior
		case models.Line80DParents:
			in.Ded80DParents += l.Amount
			in.ParentsSenior = in.ParentsSenior || l.Senior
		case models.LineNPSOwn:
			in.OwnNPS += l.Amount
		case models.LineNPSEmployer:
			moduleEmpNPS += l.Amount
		case models.Line24bSelfOccupied, models.Line24bPreEMI:
			in.SelfOccupiedInterest += l.Amount
		case models.Line24bLetOut:
			in.LetOutInterest += l.Amount
		}
		if l.Clubbed && incomeKinds[l.Kind] {
			inc.ClubbedIncome += l.Amount
			if owner := c.byID[l.OwnerID]; owner != nil && c.isMinor(owner) {
				minorClubbed[l.OwnerID] += l.Amount
			}
		}
	}
	for _, amt := range minorClubbed {
		in.MinorExemption += math.Min(math.Max(amt, 0), c.rules.MinorClubbingExemption)
	}
	// EPF and employer NPS may be on the payslip and in the PF/NPS modules;
	// take the larger figure rather than counting twice.
	in.Ded80C += math.Max(payslipPF, moduleEPF)
	in.EmployerNPS = math.Max(payslipEmpNPS, moduleEmpNPS)
	if moduleEmpNPS > payslipEmpNPS && in.GrossSalary > 0 {
		in.GrossSalary += moduleEmpNPS - payslipEmpNPS
	}

	inc.EmployerNPS = in.EmployerNPS
	inc.Interest = round2(in.InterestIncome + in.SavingsInterest)
	inc.Dividends = round2(in.DividendIncome)
	inc.Rent = round2(in.LetOutRent)
	inc.OtherIncome = round2(in.OtherIncome)
	inc.STCGEquity, inc.LTCGEquity, inc.GainsAtSlab = round2(in.STCGEquity), round2(in.LTCGEquity), round2(in.SlabCapitalGains)
	inc.SalaryFixed, inc.SalaryVariable, inc.ClubbedIncome = round2(inc.SalaryFixed), round2(inc.SalaryVariable), round2(inc.ClubbedIncome)
	inc.Total = round2(in.GrossSalary + in.Pension + in.FamilyPension + in.BusinessIncome + inc.Interest + inc.Dividends + inc.Rent +
		inc.OtherIncome + inc.STCGEquity + inc.LTCGEquity + inc.GainsAtSlab)
	for _, l := range lines {
		if l.Kind == models.LineOtherRental {
			inc.OtherRental += l.Amount
		}
	}
	inc.OtherRental = round2(inc.OtherRental)
	return in, inc, lines
}

func (c *famCtx) memberReport(pid string) *models.MemberTaxReport {
	if r := c.reports[pid]; r != nil {
		return r
	}
	p := c.byID[pid]
	in, inc, lines := c.memberInput(pid)
	c.inputs[pid] = in
	cmp := taxengine.Compare(in)
	r := &models.MemberTaxReport{
		ProfileID: pid, Name: p.Name, Relationship: p.Relationship, Color: p.Color, Occupation: p.Occupation,
		Age: in.Age, Resident: in.Resident, HasIncomeRecord: c.incomes[pid] != nil, PreferredRegime: p.PreferredRegime,
		Income: inc, Comparison: cmp, Lines: lines, Warnings: append([]string{}, c.warnings[pid]...),
	}
	if r.Lines == nil {
		r.Lines = []models.TaxLine{}
	}
	r.ChosenRegime = cmp.Recommended
	if p.PreferredRegime == "old" || p.PreferredRegime == "new" {
		r.ChosenRegime = taxengine.Regime(p.PreferredRegime)
	}
	best := math.Min(cmp.Old.TotalTax, cmp.New.TotalTax)
	r.ChosenTax = cmp.New.TotalTax
	if r.ChosenRegime == taxengine.Old {
		r.ChosenTax = cmp.Old.TotalTax
	}
	r.ExtraTaxVsBest = r.ChosenTax - best
	if rec := c.incomes[pid]; rec != nil {
		for _, ps := range c.payslips(rec) {
			r.TaxPaid += ps.TDS
		}
		r.TaxPaid += rec.Deductions.OtherTDS + rec.Deductions.AdvanceTax
	}
	r.TaxPaid = round2(r.TaxPaid)
	r.BalancePayable = round2(r.ChosenTax - r.TaxPaid)

	warn := func(msg string) { r.Warnings = append(r.Warnings, msg) }
	earner := map[string]bool{"salaried_private": true, "salaried_govt": true, "professional": true, "business": true, "pensioner": true}
	if !r.HasIncomeRecord && earner[p.Occupation] {
		warn(fmt.Sprintf("No income entered for FY %s — add payslips or other income for an accurate estimate.", c.fy))
	}
	if p.Occupation == "" {
		warn("Set an occupation so recommendations match this member's situation.")
	}
	if p.DateOfBirth == nil && p.Occupation != "minor" {
		warn("Add a date of birth — senior citizens (60+) get higher exemptions and deductions.")
	}
	if inc.ProjectedMonths > 0 {
		warn(fmt.Sprintf("%d month(s) are projected from the latest payslip (fixed pay only).", inc.ProjectedMonths))
	}
	if in.BusinessIncome > 0 || p.Occupation == "business" || p.Occupation == "professional" {
		warn("With business/professional income, you can switch back to the old regime only once after leaving it — choose carefully.")
	}
	if !in.Resident {
		warn("NRIs don't get the 87A rebate or senior-citizen slabs.")
	}
	if inc.ClubbedIncome != 0 {
		warn(fmt.Sprintf("Includes %s of income from assets in other members' names, taxed here under clubbing rules.", formatINR(inc.ClubbedIncome)))
	}
	for giver, amt := range c.unclubbed[pid] {
		if amt == 0 {
			continue
		}
		warn(fmt.Sprintf("Clubbing is turned off: %s of income that the law taxes in %s's hands is counted here instead. The tax department can add it back to %s's income.",
			formatINR(amt), c.name(giver), c.name(giver)))
	}
	if excess := in.SelfOccupiedInterest - c.rules.Lim24bSelfOccupied; excess > 0 {
		warn(fmt.Sprintf("Home loan interest exceeds the ₹2L self-occupied cap by %s (old regime only).", formatINR(excess)))
	}
	c.reports[pid] = r
	return r
}

// ---------------------------------------------------------------------------
// Reports

func (s *FamilyTaxService) Report(ctx context.Context, userID, fy string) (*models.FamilyTaxReport, error) {
	c, err := s.load(ctx, userID, fy)
	if err != nil {
		return nil, err
	}
	rep := &models.FamilyTaxReport{FY: c.fy, Settings: c.settings, Act: c.rules.Act, SupportedFYs: taxengine.SupportedFYs(),
		Members: []models.MemberTaxReport{}, Suggestions: []models.TaxSuggestion{}, Disclaimer: taxDisclaimer}
	if !c.rulesOK {
		rep.Suggestions = append(rep.Suggestions, models.TaxSuggestion{ID: "fy-rules", Category: "data",
			Title:  "Tax rules not available for FY " + c.fy,
			Detail: fmt.Sprintf("Using FY %s rules as an approximation.", c.rules.FY)})
	}
	if len(c.profiles) == 0 {
		rep.Suggestions = append(rep.Suggestions, models.TaxSuggestion{ID: "no-members", Category: "data",
			Title: "Add family members", Detail: "Add yourself and your family on the Family page to get per-member tax estimates."})
		return rep, nil
	}
	for i := range c.profiles {
		r := c.memberReport(c.profiles[i].ID.Hex())
		rep.Members = append(rep.Members, *r)
		rep.TotalTaxBest += math.Min(r.Comparison.Old.TotalTax, r.Comparison.New.TotalTax)
		rep.TotalTaxPreferred += r.ChosenTax
	}
	rep.TotalTaxBest, rep.TotalTaxPreferred = round2(rep.TotalTaxBest), round2(rep.TotalTaxPreferred)
	rep.RegimeSavings = round2(rep.TotalTaxPreferred - rep.TotalTaxBest)
	rep.UnassignedItems = len(c.unassigned)
	rep.DefaultMemberName = c.name(c.defaultID)
	for _, byGiver := range c.unclubbed {
		for _, amt := range byGiver {
			rep.ClubbingIgnored += amt
		}
	}
	rep.ClubbingIgnored = round2(rep.ClubbingIgnored)
	if rep.ClubbingIgnored > 0 {
		rep.Suggestions = append(rep.Suggestions, models.TaxSuggestion{ID: "clubbing-off", Category: "data",
			Title: fmt.Sprintf("Clubbing is off — %s of income is taxed in a spouse's or child's hands", formatINR(rep.ClubbingIgnored)),
			Detail: "Estimates assume this income is taxed where the asset is held. Under the Income-tax Act, income from money gifted to a spouse or held by a minor child is added to the giver's / parent's income, " +
				"so the real tax may be higher. It isn't clubbed if the spouse invests their own money, money gifted by their own relatives, or money lent at a fair interest rate."})
	}
	rep.Suggestions = append(rep.Suggestions, s.suggestions(ctx, userID, c)...)
	sort.SliceStable(rep.Suggestions, func(i, j int) bool {
		di, dj := rep.Suggestions[i].Category == "data", rep.Suggestions[j].Category == "data"
		if di != dj {
			return !di
		}
		return rep.Suggestions[i].EstSaving > rep.Suggestions[j].EstSaving
	})
	return rep, nil
}

// MemberSummary is the deduction/income summary for one member's attributed assets.
func (s *FamilyTaxService) MemberSummary(ctx context.Context, userID, fy, profileID string) (*models.TaxSummary, error) {
	c, err := s.load(ctx, userID, fy)
	if err != nil {
		return nil, err
	}
	if c.byID[profileID] == nil {
		return nil, notFoundf("family member not found")
	}
	var lines []models.TaxLine
	for _, l := range c.lines {
		if l.ProfileID == profileID {
			lines = append(lines, l)
		}
	}
	sum := SummaryFromLines(c.fy, lines, c.memberGains(profileID))
	sum.ProfileID = profileID
	return sum, nil
}

func (s *FamilyTaxService) MemberCapitalGains(ctx context.Context, userID, fy, profileID string) (*models.CapitalGainsSummary, error) {
	c, err := s.load(ctx, userID, fy)
	if err != nil {
		return nil, err
	}
	if c.byID[profileID] == nil {
		return nil, notFoundf("family member not found")
	}
	return c.memberGains(profileID), nil
}

func (c *famCtx) memberGains(pid string) *models.CapitalGainsSummary {
	cg := &models.CapitalGainsSummary{Entries: []models.CapitalGainEntry{}, LTCGExemption: c.rules.LTCGEquityExemption, HarvestingTips: []string{}}
	for _, g := range c.gains {
		if g.taxpayer == pid {
			cg.Entries = append(cg.Entries, g.entry)
		}
	}
	FinalizeCapitalGains(cg)
	if room := cg.LTCGExemption - math.Max(0, cg.LTCG); room > 5000 {
		cg.HarvestingTips = append(cg.HarvestingTips, fmt.Sprintf("%s can still book %s of long-term equity gains tax-free this year (₹1.25L exemption per person).", c.name(pid), formatINR(room)))
	}
	return cg
}

// ---------------------------------------------------------------------------
// Suggestions

func (s *FamilyTaxService) suggestions(ctx context.Context, userID string, c *famCtx) []models.TaxSuggestion {
	var out []models.TaxSuggestion
	add := func(sg models.TaxSuggestion) {
		sg.EstSaving = math.Round(sg.EstSaving)
		if sg.ProfileID != "" && sg.MemberName == "" {
			sg.MemberName = c.name(sg.ProfileID)
		}
		out = append(out, sg)
	}
	regimeName := func(r taxengine.Regime) string {
		if r == taxengine.Old {
			return "old"
		}
		return "new"
	}
	salaried := func(p *models.Profile) bool {
		return p.Occupation == "salaried_private" || p.Occupation == "salaried_govt" || c.inputs[p.ID.Hex()].BasicDA > 0
	}
	chosenTotal := func(pid string, in taxengine.Input) float64 {
		return taxengine.Total(in, c.reports[pid].ChosenRegime)
	}

	var parentIDs []string
	for i := range c.profiles {
		if c.profiles[i].Relationship == "parent" {
			parentIDs = append(parentIDs, c.profiles[i].ID.Hex())
		}
	}
	claimsParents80D := false
	for _, pr := range c.profiles {
		if in := c.inputs[pr.ID.Hex()]; in.Ded80DParents > 0 {
			claimsParents80D = true
		}
	}

	for i := range c.profiles {
		p := &c.profiles[i]
		pid := p.ID.Hex()
		r := c.reports[pid]
		in := c.inputs[pid]
		cmp := r.Comparison
		rec := regimeName(cmp.Recommended)
		other := "old"
		if rec == "old" {
			other = "new"
		}

		// 1. Regime choice
		if r.ExtraTaxVsBest >= 500 {
			detail := fmt.Sprintf("%s's declared %s regime costs %s more than the %s regime this year.", p.Name, p.PreferredRegime, formatINR(r.ExtraTaxVsBest), rec)
			if salaried(p) {
				detail += " Tell your employer at the start of the year (or choose it when filing the return)."
			}
			add(models.TaxSuggestion{ID: "regime-switch:" + pid, ProfileID: pid, Category: "regime",
				Title: fmt.Sprintf("Switch %s to the %s regime", p.Name, rec), Detail: detail, EstSaving: r.ExtraTaxVsBest})
		} else if p.PreferredRegime == "" && cmp.Savings >= 500 {
			add(models.TaxSuggestion{ID: "regime-choose:" + pid, ProfileID: pid, Category: "regime",
				Title: fmt.Sprintf("%s: choose the %s regime", p.Name, rec),
				Detail: fmt.Sprintf("Estimated tax %s under the %s regime vs %s under the %s regime.",
					formatINR(math.Min(cmp.Old.TotalTax, cmp.New.TotalTax)), rec, formatINR(math.Max(cmp.Old.TotalTax, cmp.New.TotalTax)), other),
				EstSaving: cmp.Savings})
		}

		// 2. Deduction headroom when the old regime wins or could win
		h := cmp.Headroom
		headroomTotal := h.Sec80C + h.Sec80CCD1B + h.Sec80DSelf
		if cmp.Recommended == taxengine.New && cmp.BreakEvenDeductions > 0 && cmp.OldTaxWithHeadroom < cmp.New.TotalTax {
			add(models.TaxSuggestion{ID: "old-breakeven:" + pid, ProfileID: pid, Category: "deduction",
				Title: fmt.Sprintf("%s: old regime wins with %s more deductions", p.Name, formatINR(cmp.BreakEvenDeductions)),
				Detail: fmt.Sprintf("Using the remaining 80C (%s), NPS 80CCD(1B) (%s) and 80D (%s) limits brings old-regime tax to %s, below %s under the new regime.",
					formatINR(h.Sec80C), formatINR(h.Sec80CCD1B), formatINR(h.Sec80DSelf), formatINR(cmp.OldTaxWithHeadroom), formatINR(cmp.New.TotalTax)),
				EstSaving: cmp.New.TotalTax - cmp.OldTaxWithHeadroom})
		} else if r.ChosenRegime == taxengine.Old && headroomTotal > 0 && in.GrossSalary+in.BusinessIncome+in.Pension > 0 {
			maxed := in
			maxed.Ded80C += h.Sec80C
			maxed.OwnNPS += h.Sec80CCD1B
			maxed.Ded80DSelf += h.Sec80DSelf
			if saving := cmp.Old.TotalTax - taxengine.Total(maxed, taxengine.Old); saving >= 500 {
				parts := []string{}
				if h.Sec80C > 0 {
					parts = append(parts, formatINR(h.Sec80C)+" under 80C (PPF, ELSS, SSY…)")
				}
				if h.Sec80CCD1B > 0 {
					parts = append(parts, formatINR(h.Sec80CCD1B)+" in NPS (80CCD(1B))")
				}
				if h.Sec80DSelf > 0 {
					parts = append(parts, formatINR(h.Sec80DSelf)+" of health insurance (80D)")
				}
				add(models.TaxSuggestion{ID: "headroom:" + pid, ProfileID: pid, Category: "deduction",
					Title:  fmt.Sprintf("%s: use remaining deductions before 31 March", p.Name),
					Detail: "Room left: " + strings.Join(parts, ", ") + ".", EstSaving: saving})
			}
		}

		// 3. Employer NPS (80CCD(2)) works in both regimes
		if salaried(p) && in.BasicDA > 0 && in.EmployerNPS == 0 {
			pct := c.rules.EmployerNPSPctNew
			if r.ChosenRegime == taxengine.Old {
				pct = c.rules.EmployerNPSPctOld
			}
			probe := in
			probe.EmployerNPS = pct / 100 * in.BasicDA
			if saving := r.ChosenTax - chosenTotal(pid, probe); saving >= 1000 {
				add(models.TaxSuggestion{ID: "employer-nps:" + pid, ProfileID: pid, Category: "deduction",
					Title: fmt.Sprintf("%s: route part of salary to NPS via the employer", p.Name),
					Detail: fmt.Sprintf("Employer NPS up to %.0f%% of basic+DA (%s/yr) is deductible under 80CCD(2) in both regimes. Ask HR to restructure CTC; the money is locked until 60.",
						pct, formatINR(probe.EmployerNPS)), EstSaving: saving})
			}
		}

		// 4. HRA by paying rent to parents (old regime)
		if salaried(p) && in.HRAReceived > 0 && in.RentPaid == 0 && len(parentIDs) > 0 {
			probe := in
			probe.HRAExempt = taxengine.HRAExemption(in.BasicDA, in.HRAReceived, in.HRAReceived+0.1*in.BasicDA, p.MetroCity)
			oldWith := taxengine.Total(probe, taxengine.Old)
			if saving := r.ChosenTax - oldWith; saving >= 2000 {
				add(models.TaxSuggestion{ID: "hra-parents:" + pid, ProfileID: pid, Category: "deduction",
					Title: fmt.Sprintf("%s: claim HRA by paying rent to a parent", p.Name),
					Detail: fmt.Sprintf("If you live in a home owned by %s, paying them ~%s/yr in rent (by bank transfer, with an agreement) gives an HRA exemption under the old regime. The parent reports it as rental income (30%% standard deduction).",
						c.name(parentIDs[0]), formatINR(in.HRAReceived+0.1*in.BasicDA)), EstSaving: saving})
			}
		}

		// 5. Joint home loan: unused interest above the ₹2L cap
		if excess := in.SelfOccupiedInterest - c.rules.Lim24bSelfOccupied; excess > 0 {
			for j := range c.profiles {
				q := c.profiles[j].ID.Hex()
				if !c.spouses(pid, q) || c.baseIncome[q] <= 0 {
					continue
				}
				probe := c.inputs[q]
				probe.SelfOccupiedInterest += math.Min(excess, c.rules.Lim24bSelfOccupied)
				saving := taxengine.Total(c.inputs[q], taxengine.Old) - taxengine.Total(probe, taxengine.Old)
				if saving >= 1000 {
					add(models.TaxSuggestion{ID: "joint-loan:" + pid, ProfileID: pid, Category: "deduction",
						Title: fmt.Sprintf("Share the home loan with %s", c.name(q)),
						Detail: fmt.Sprintf("%s pays %s of interest above the ₹2L cap. If %s is a co-owner and co-borrower paying their share of EMIs, they can claim it too (old regime). Set the ownership split on the Family page.",
							p.Name, formatINR(excess), c.name(q)), EstSaving: saving})
				}
			}
		}
	}

	// 6. Parents' health insurance (80D) paid by an earning child
	if !claimsParents80D && len(parentIDs) > 0 {
		senior := false
		for _, id := range parentIDs {
			if pp := c.byID[id]; pp.DateOfBirth == nil || pp.AgeOn(c.ageDate()) >= 60 {
				senior = true
			}
		}
		limit := c.rules.Lim80DParents
		if senior {
			limit = c.rules.Lim80DParentsSenior
		}
		bestID, bestSaving := "", 0.0
		for i := range c.profiles {
			pid := c.profiles[i].ID.Hex()
			if c.reports[pid].ChosenRegime != taxengine.Old && c.reports[pid].Comparison.Recommended != taxengine.Old {
				continue
			}
			probe := c.inputs[pid]
			probe.Ded80DParents, probe.ParentsSenior = limit, senior
			if saving := c.reports[pid].Comparison.Old.TotalTax - taxengine.Total(probe, taxengine.Old); saving > bestSaving {
				bestID, bestSaving = pid, saving
			}
		}
		if bestID != "" && bestSaving >= 1000 {
			add(models.TaxSuggestion{ID: "parents-80d", ProfileID: bestID, Category: "deduction",
				Title:     "Buy health cover for parents",
				Detail:    fmt.Sprintf("Premiums paid for parents' health insurance are deductible up to %s under 80D (old regime), on top of your own limit. Add the policy with 'cover for: parents'.", formatINR(limit)),
				EstSaving: bestSaving})
		}
	}

	// 7. Move interest-earning deposits to low-income members (not clubbed)
	for i := range c.profiles {
		eid := c.profiles[i].ID.Hex()
		er := c.reports[eid]
		earnerRate := er.Comparison.New.MarginalPc
		if er.ChosenRegime == taxengine.Old {
			earnerRate = er.Comparison.Old.MarginalPc
		}
		// Interest taxed in the earner's hands comes from their money, whether
		// held in their own name or clubbed back from a spouse / minor.
		interest := 0.0
		for _, l := range c.lines {
			if l.ProfileID == eid && l.Kind == models.LineInterest {
				interest += l.Amount
			}
		}
		if earnerRate < 10 || interest < 10000 {
			continue
		}
		bestID, bestRoom := "", 0.0
		for j := range c.profiles {
			q := &c.profiles[j]
			qid := q.ID.Hex()
			if qid == eid {
				continue
			}
			if target, _ := c.clubTarget(qid, eid); target != qid {
				continue // income would be clubbed back
			}
			if room := taxengine.ZeroTaxRoom(c.inputs[qid], interest); room > bestRoom {
				bestID, bestRoom = qid, room
			}
		}
		if bestID != "" && bestRoom >= 10000 {
			moved := math.Min(interest, bestRoom)
			detail := fmt.Sprintf("%s earns %s of FD/bond interest at a %.1f%% marginal rate. %s can earn about %s more interest tax-free. Gifts to parents, adult children or siblings aren't clubbed — but the money legally becomes theirs.",
				c.name(eid), formatINR(interest), earnerRate, c.name(bestID), formatINR(bestRoom))
			if _, kind := c.clubRule(bestID, eid); kind != "" {
				detail = fmt.Sprintf("%s earns %s of FD/bond interest at a %.1f%% marginal rate; %s could earn about %s more tax-free. Clubbing is turned off, so this assumes income from money you give %s isn't added back to yours — the law says it is, unless it's %s's own money.",
					c.name(eid), formatINR(interest), earnerRate, c.name(bestID), formatINR(bestRoom), c.name(bestID), c.name(bestID))
			}
			add(models.TaxSuggestion{ID: "gift-deposits:" + eid, ProfileID: eid, Category: "placement",
				Title:     fmt.Sprintf("Hold deposits in %s's name instead of %s's", c.name(bestID), c.name(eid)),
				Detail:    detail,
				EstSaving: moved * earnerRate / 100})
		}
	}

	// 8. LTCG harvesting with each member's own ₹1.25L exemption
	unrealised := map[string]float64{}
	mfs, _ := s.tax.mfRepo.GetAll(ctx, userID)
	for _, mf := range mfs {
		if gain := mf.CurrentValue - mf.TotalInvested; isEquityFund(mf.FundType) && mf.TotalInvested > 0 && gain > 0 {
			c.spreadGain("mutual_fund", mf.ID.Hex(), gain, unrealised)
		}
	}
	stocks, _ := s.tax.stockRepo.GetAll(ctx, userID)
	for _, st := range stocks {
		if gain := st.CurrentValue - st.TotalInvested; st.TotalInvested > 0 && gain > 0 {
			c.spreadGain("stock", st.ID.Hex(), gain, unrealised)
		}
	}
	for i := range c.profiles {
		pid := c.profiles[i].ID.Hex()
		room := c.reports[pid].Comparison.Headroom.LTCGExemption
		if harvest := math.Min(unrealised[pid], room); harvest >= 10000 {
			add(models.TaxSuggestion{ID: "ltcg-harvest:" + pid, ProfileID: pid, Category: "capital_gains",
				Title:     fmt.Sprintf("%s: book %s of long-term gains tax-free", c.profiles[i].Name, formatINR(harvest)),
				Detail:    "Each person gets ₹1.25L of equity LTCG tax-free every year. Sell units held over 12 months and buy back to reset the cost (check exit loads).",
				EstSaving: harvest * c.rules.LTCGEquityRate / 100 * (1 + c.rules.CessRate/100)})
		}
	}

	// 9. Data quality
	for i := range c.profiles {
		p := &c.profiles[i]
		pid := p.ID.Hex()
		if p.Relationship == "spouse" || p.Relationship == "self" {
			hasOwnLines := false
			for _, l := range c.lines {
				if l.OwnerID == pid && incomeKinds[l.Kind] && !l.Clubbed {
					hasOwnLines = true
				}
			}
			if hasOwnLines && !c.settings.IgnoreSpouseClubbing && (p.Occupation == "homemaker" || p.Occupation == "unemployed") {
				add(models.TaxSuggestion{ID: "funded-by:" + pid, ProfileID: pid, Category: "clubbing",
					Title:  fmt.Sprintf("Check who funded %s's investments", p.Name),
					Detail: "If the money for investments in a spouse's name came from the other spouse, the income is taxed in the giver's hands (clubbing). Set 'Funded by' on the Family page so estimates are right. Her/his own savings, or gifts from her/his relatives, are not clubbed."})
			}
		}
	}
	if n := len(c.unassigned); n > 0 {
		add(models.TaxSuggestion{ID: "unassigned", Category: "data",
			Title:  fmt.Sprintf("%d tax-relevant item(s) aren't assigned to a member", n),
			Detail: fmt.Sprintf("They're counted under %s. Assign owners on the Family page for accurate per-member tax.", c.name(c.defaultID))})
	}
	return out
}

// spreadGain attributes an asset's unrealised gain to its taxpayers.
func (c *famCtx) spreadGain(assetType, assetID string, gain float64, into map[string]float64) {
	shares, funder, _ := c.owners(assetType, assetID)
	for _, sh := range shares {
		t, _ := c.clubTarget(sh.ProfileID, funder)
		into[t] += gain * sh.Pct / 100
	}
}

// nonNegZero turns -0 into 0 so JSON doesn't show "-0".
func nonNegZero(v float64) float64 {
	if v == 0 {
		return 0
	}
	return v
}

// ---------------------------------------------------------------------------
// Placement optimizer

func (s *FamilyTaxService) Placement(ctx context.Context, userID string, req models.PlacementRequest) (*models.PlacementResult, error) {
	if _, ok := models.PlacementInstruments[req.Instrument]; !ok {
		return nil, invalidf("unknown instrument %q", req.Instrument)
	}
	c, err := s.load(ctx, userID, req.FY)
	if err != nil {
		return nil, err
	}
	if len(c.profiles) == 0 {
		return nil, invalidf("add family members first")
	}
	if req.FundedBy != "" && c.byID[req.FundedBy] == nil {
		return nil, notFoundf("funding member not found")
	}
	return c.placement(req), nil
}

func (c *famCtx) placement(req models.PlacementRequest) *models.PlacementResult {
	for i := range c.profiles {
		c.memberReport(c.profiles[i].ID.Hex())
	}
	req.FY = c.fy
	res := &models.PlacementResult{FY: c.fy, Request: req, Options: []models.PlacementOption{}, Notes: []string{}}
	note := func(n string) { res.Notes = append(res.Notes, n) }
	income := req.Amount * req.ExpectedReturnPct / 100

	switch req.Instrument {
	case "equity_mf", "elss":
		note("Equity returns are modelled as long-term gains booked every year, so each person's ₹1.25L LTCG exemption is used. Gains held unbooked are taxed only on sale.")
	case "debt_mf":
		note("Debt fund gains are taxed at the holder's slab rate when redeemed; this treats the yearly return as realised.")
	case "ppf", "ssy":
		note("PPF and SSY interest is tax-free (EEE), so the holder doesn't matter for income tax; the 80C deduction goes to the depositor.")
	case "nps":
		note("NPS growth is tax-free until 60; the 80CCD(1)/(1B) deduction goes to the account holder (old regime).")
	}
	if req.FundedBy != "" {
		note(fmt.Sprintf("Money comes from %s. Income from money given to a spouse or minor child is clubbed back to the giver; gifts to adult children, parents or siblings are not.", c.name(req.FundedBy)))
	} else {
		note("Each option assumes the member invests their own money. If one person funds another's investment, set 'Funded by' to see clubbing effects.")
	}
	if c.settings.IgnoreSpouseClubbing || c.settings.IgnoreMinorClubbing {
		off := []string{}
		if c.settings.IgnoreSpouseClubbing {
			off = append(off, "spouse")
		}
		if c.settings.IgnoreMinorClubbing {
			off = append(off, "minor child")
		}
		note(fmt.Sprintf("Clubbing is turned off for: %s. Income is taxed in the holder's own hands here; under the law it may be added to the giver's / parent's income.", strings.Join(off, ", ")))
	}
	if (req.Instrument == "ppf" || req.Instrument == "ssy") && req.Amount > 150000 {
		note("PPF and SSY accept at most ₹1.5L per account per year.")
	}
	if req.Instrument == "scss" && req.Amount > 3000000 {
		note("SCSS is capped at ₹30L per person.")
	}

	// Members whose tax changes in a scenario; delta uses each member's best regime.
	bestTax := func(in taxengine.Input) float64 { _, t := taxengine.Best(in); return t }
	delta := func(mods map[string]taxengine.Input) float64 {
		d := 0.0
		for pid, in := range mods {
			d += bestTax(in) - bestTax(c.inputs[pid])
		}
		return d
	}

	for i := range c.profiles {
		p := &c.profiles[i]
		pid := p.ID.Hex()
		opt := models.PlacementOption{ProfileID: pid, Name: p.Name, Eligible: true, AnnualIncome: round2(income), Notes: []string{}}
		age := p.AgeOn(c.ageDate())
		ineligible := func(why string) { opt.Eligible = false; opt.Notes = append(opt.Notes, why) }
		switch req.Instrument {
		case "scss":
			if p.DateOfBirth == nil {
				ineligible("Add a date of birth — SCSS is for those aged 60+.")
			} else if age < 60 {
				ineligible("SCSS is only for those aged 60+ (55+ for some retirees).")
			}
		case "ssy":
			if p.Relationship != "child" || p.DateOfBirth == nil || age >= 10 {
				ineligible("SSY is only for a daughter under 10 (add her date of birth).")
			} else {
				opt.Notes = append(opt.Notes, "Only for a girl child.")
			}
		case "ppf":
			if !p.IsResident() {
				ineligible("NRIs can't open a new PPF account.")
			}
		case "nps":
			if p.DateOfBirth != nil && (age < 18 || age > 70) {
				ineligible("NPS accounts are for ages 18–70.")
			}
		case "elss", "equity_mf":
			if c.isMinor(p) {
				opt.Notes = append(opt.Notes, "Held by a minor through a guardian.")
			}
		}

		taxpayer, clubbed := c.clubTarget(pid, req.FundedBy)
		opt.TaxpayerID, opt.TaxpayerName = taxpayer, c.name(taxpayer)
		if clubbed {
			opt.Notes = append(opt.Notes, fmt.Sprintf("Income is clubbed with %s.", c.name(taxpayer)))
		} else if lawTarget, kind := c.clubRule(pid, req.FundedBy); kind != "" {
			opt.Notes = append(opt.Notes, fmt.Sprintf("Clubbing is off: by law this income would be taxed in %s's hands.", c.name(lawTarget)))
		}

		incomeMods := map[string]taxengine.Input{}
		addIncome := func(in taxengine.Input) taxengine.Input {
			switch req.Instrument {
			case "fd", "bond", "scss":
				in.InterestIncome += income
			case "savings":
				in.SavingsInterest += income
			case "debt_mf":
				in.SlabCapitalGains += income
			case "equity_mf", "elss":
				in.LTCGEquity += income
			}
			return in
		}
		switch req.Instrument {
		case "ppf", "ssy", "nps":
		default:
			incomeMods[taxpayer] = addIncome(c.inputs[taxpayer])
		}

		// Deduction claimant
		claimant, dedAmount := "", 0.0
		switch req.Instrument {
		case "elss", "scss":
			claimant, dedAmount = pid, req.Amount
		case "nps":
			claimant, dedAmount = pid, req.Amount
		case "ppf":
			claimant, dedAmount = pid, math.Min(req.Amount, 150000)
			if req.FundedBy != "" && req.FundedBy != pid {
				if c.payerMayClaim(req.FundedBy, pid) {
					claimant = req.FundedBy
				} else {
					claimant = ""
					opt.Notes = append(opt.Notes, "80C isn't available for deposits in this member's name by the funder.")
				}
			}
		case "ssy":
			dedAmount = math.Min(req.Amount, 150000)
			claimant = req.FundedBy
			if claimant == "" {
				if parents := c.parentsOf(p); len(parents) > 0 {
					claimant = parents[0]
					for _, id := range parents[1:] {
						if c.baseIncome[id] > c.baseIncome[claimant] {
							claimant = id
						}
					}
				}
			}
		}
		if req.FundedBy != "" && req.Instrument == "elss" && req.FundedBy != pid {
			opt.Notes = append(opt.Notes, "ELSS 80C is claimed by the unit holder.")
		}
		dedMods := map[string]taxengine.Input{}
		if claimant != "" && dedAmount > 0 {
			in := c.inputs[claimant]
			if req.Instrument == "nps" {
				in.OwnNPS += dedAmount
			} else {
				in.Ded80C += dedAmount
			}
			dedMods[claimant] = in
			if claimant != pid {
				opt.Notes = append(opt.Notes, fmt.Sprintf("80C deduction claimed by %s.", c.name(claimant)))
			}
		}

		combined := map[string]taxengine.Input{}
		for k, v := range incomeMods {
			combined[k] = v
		}
		for k, v := range dedMods {
			if base, ok := combined[k]; ok {
				if req.Instrument == "nps" {
					base.OwnNPS = v.OwnNPS
				} else {
					base.Ded80C = v.Ded80C
				}
				combined[k] = base
			} else {
				combined[k] = v
			}
		}
		opt.IncrementalTax = nonNegZero(round2(delta(incomeMods)))
		opt.DeductionSaving = nonNegZero(round2(-delta(dedMods)))
		opt.NetTaxImpact = nonNegZero(round2(delta(combined)))
		opt.PostTaxIncome = round2(income - opt.NetTaxImpact)
		if req.Amount > 0 {
			opt.PostTaxYieldPct = math.Round(opt.PostTaxIncome/req.Amount*10000) / 100
		}
		if opt.DeductionSaving > 0 {
			opt.Notes = append(opt.Notes, "Deduction saving applies in the year of investment (first-year view).")
		}
		if req.Instrument == "equity_mf" || req.Instrument == "elss" {
			if room := c.reports[taxpayer].Comparison.Headroom.LTCGExemption; room > 0 {
				opt.Notes = append(opt.Notes, fmt.Sprintf("%s has %s of LTCG exemption left this year.", c.name(taxpayer), formatINR(room)))
			}
		}
		res.Options = append(res.Options, opt)
	}

	sort.SliceStable(res.Options, func(i, j int) bool {
		a, b := res.Options[i], res.Options[j]
		if a.Eligible != b.Eligible {
			return a.Eligible
		}
		return a.NetTaxImpact < b.NetTaxImpact
	})
	if len(res.Options) > 0 && res.Options[0].Eligible {
		res.BestProfile = res.Options[0].ProfileID
	}
	return res
}
