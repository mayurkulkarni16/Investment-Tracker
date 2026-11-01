package taxengine

import (
	"math"
)

// Input is one individual's income and deduction claims for a financial year.
// All amounts are annual rupees. Deduction fields are amounts paid/claimed;
// the engine applies the statutory caps.
type Input struct {
	FY         string
	Age        int
	Resident   bool
	Disability string // "", "disabled" (40%+), "severe" (80%+) — section 80U

	// Salary head
	GrossSalary     float64 // all taxable salary incl. HRA, LTA, bonus, perquisites, employer NPS
	BasicDA         float64 // for HRA and NPS percentage caps
	HRAExempt       float64 // 10(13A), old regime only (see HRAExemption)
	HRAReceived     float64 // used to decide 80GG eligibility
	LTAExempt       float64 // 10(5), old regime only
	ProfessionalTax float64 // 16(iii), old regime only
	EmployerNPS     float64 // 80CCD(2)
	Pension         float64 // uncommuted pension, taxed as salary

	FamilyPension  float64 // other sources, with 57(iia) deduction
	BusinessIncome float64 // net taxable business / professional profit

	// House property (owner's share)
	SelfOccupiedInterest float64
	LetOutRent           float64 // annual gross rent
	LetOutMunicipalTax   float64
	LetOutInterest       float64

	// Other sources taxed at slab
	InterestIncome   float64 // deposits / bonds (counts for 80TTB)
	SavingsInterest  float64 // savings accounts (80TTA / 80TTB)
	DividendIncome   float64
	OtherIncome      float64
	SlabCapitalGains float64 // debt MF and other gains taxed at slab
	MinorExemption   float64 // 10(32): ₹1,500 per minor child on income clubbed from them

	// Special-rate capital gains (net of losses)
	STCGEquity float64 // 111A
	LTCGEquity float64 // 112A, before the annual exemption
	LTCGOther  float64 // 112

	// Chapter VI-A claims
	Ded80C           float64 // EPF, PPF, ELSS, LIC, principal, tuition, SSY… (excl. NPS)
	OwnNPS           float64 // own NPS Tier-1 contributions: 80CCD(1) + 80CCD(1B)
	Ded80DSelf       float64
	SelfSeniorHealth bool // a senior citizen is covered under the self/family policy
	Ded80DParents    float64
	ParentsSenior    bool
	Ded80E           float64
	Ded80G           float64 // qualifying amount after 50%/100% rules
	RentPaid         float64 // annual rent; used for 80GG when no HRA is received
	SelfEmployed     bool    // 80CCD(1) cap is 20% of GTI instead of 10% of salary

	// ExtraDeductionOld is an uncapped what-if deduction used for break-even.
	ExtraDeductionOld float64
}

type Line struct {
	Code       string  `json:"code"`                  // Income-tax Act, 1961 section
	NewSection string  `json:"new_section,omitempty"` // Income-tax Act, 2025 section
	Label      string  `json:"label"`
	Claimed    float64 `json:"claimed"`
	Allowed    float64 `json:"allowed"`
	Limit      float64 `json:"limit,omitempty"`
}

type SlabLine struct {
	From   float64 `json:"from"`
	To     float64 `json:"to"` // 0 = no upper bound
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
	Tax    float64 `json:"tax"`
}

type Result struct {
	Regime Regime `json:"regime"`

	GrossSalary       float64 `json:"gross_salary"`
	StandardDeduction float64 `json:"standard_deduction"`
	Exemptions        float64 `json:"exemptions"` // HRA + LTA
	ProfessionalTax   float64 `json:"professional_tax"`
	SalaryIncome      float64 `json:"salary_income"`
	HouseProperty     float64 `json:"house_property"` // may be negative (loss set off)
	BusinessIncome    float64 `json:"business_income"`
	OtherSources      float64 `json:"other_sources"`
	SpecialIncome     float64 `json:"special_income"` // STCG 111A + LTCG 112A/112
	GrossTotalIncome  float64 `json:"gross_total_income"`

	Deductions      []Line  `json:"deductions"`
	TotalDeductions float64 `json:"total_deductions"`
	NormalIncome    float64 `json:"normal_income"` // taxed at slab rates
	TotalIncome     float64 `json:"total_income"`

	SlabTax     float64    `json:"slab_tax"`
	SpecialTax  float64    `json:"special_tax"`
	Rebate87A   float64    `json:"rebate_87a"`
	Surcharge   float64    `json:"surcharge"`
	Cess        float64    `json:"cess"`
	TotalTax    float64    `json:"total_tax"`
	EffectivePc float64    `json:"effective_rate"` // % of gross total income
	MarginalPc  float64    `json:"marginal_rate"`  // % on the next rupee of slab income incl. cess
	Slabs       []SlabLine `json:"slabs"`
}

type Headroom struct {
	Sec80C        float64 `json:"sec_80c"`
	Sec80CCD1B    float64 `json:"sec_80ccd1b"`
	Sec80DSelf    float64 `json:"sec_80d_self"`
	Sec80DParents float64 `json:"sec_80d_parents"`
	LTCGExemption float64 `json:"ltcg_exemption"`
	// Distance to the new-regime 87A rebate limit (negative once crossed).
	RebateNewRoom float64 `json:"rebate_new_room"`
}

type Comparison struct {
	FY          string  `json:"fy"`
	Act         string  `json:"act"`
	Old         Result  `json:"old"`
	New         Result  `json:"new"`
	Recommended Regime  `json:"recommended"`
	Savings     float64 `json:"savings"`
	// Extra old-regime deductions needed to match the new regime; 0 when the
	// old regime already wins, -1 when no amount of deductions would.
	BreakEvenDeductions float64  `json:"break_even_deductions"`
	OldTaxWithHeadroom  float64  `json:"old_tax_with_headroom"` // old regime if 80C/80CCD(1B)/80D are maxed
	Headroom            Headroom `json:"headroom"`
}

func round(v float64) float64 { return math.Round(v) }

func pos(v float64) float64 { return math.Max(0, v) }

// HRAExemption is the 10(13A) exemption for one period (month or year):
// least of HRA received, rent paid minus 10% of basic+DA, and 50% (metro) or
// 40% of basic+DA.
func HRAExemption(basicDA, hraReceived, rentPaid float64, metro bool) float64 {
	if hraReceived <= 0 || rentPaid <= 0 {
		return 0
	}
	pct := 0.4
	if metro {
		pct = 0.5
	}
	return pos(math.Min(hraReceived, math.Min(rentPaid-0.1*basicDA, pct*basicDA)))
}

func (r *Rules) slabsFor(regime Regime, age int, resident bool) []Slab {
	if regime == New {
		return r.NewSlabs
	}
	if resident && age >= 80 {
		return r.OldSlabsSuperSenior
	}
	if resident && age >= 60 {
		return r.OldSlabsSenior
	}
	return r.OldSlabs
}

func slabTax(income float64, slabs []Slab) (float64, []SlabLine) {
	var tax, from float64
	lines := []SlabLine{}
	for _, s := range slabs {
		if income <= from {
			break
		}
		amt := math.Min(income, s.UpTo) - from
		t := amt * s.Rate / 100
		to := s.UpTo
		if math.IsInf(to, 1) {
			to = 0
		}
		lines = append(lines, SlabLine{From: from, To: to, Rate: s.Rate, Amount: round(amt), Tax: round(t)})
		tax += t
		from = s.UpTo
	}
	return tax, lines
}

func basicExemption(slabs []Slab) float64 {
	if len(slabs) > 0 && slabs[0].Rate == 0 {
		return slabs[0].UpTo
	}
	return 0
}

// Compute returns the tax for one regime.
func Compute(in Input, regime Regime) Result {
	r, _ := RulesFor(in.FY)
	res := Result{Regime: regime, Deductions: []Line{}}
	isOld := regime == Old
	line := func(code, label string, claimed, allowed, limit float64) {
		if claimed <= 0 && allowed <= 0 {
			return
		}
		res.Deductions = append(res.Deductions, Line{Code: code, NewSection: r.NewSection(code), Label: label,
			Claimed: round(claimed), Allowed: round(allowed), Limit: limit})
	}

	// Salary
	gross := pos(in.GrossSalary) + pos(in.Pension)
	res.GrossSalary = round(gross)
	if gross > 0 {
		std := r.StdDeductionNew
		if isOld {
			std = r.StdDeductionOld
		}
		res.StandardDeduction = math.Min(std, gross)
	}
	if isOld {
		res.Exemptions = math.Min(pos(in.HRAExempt)+pos(in.LTAExempt), gross)
		res.ProfessionalTax = math.Min(pos(in.ProfessionalTax), 2500)
	}
	res.SalaryIncome = pos(gross - res.StandardDeduction - res.Exemptions - res.ProfessionalTax)

	// House property
	letOut := 0.0
	if in.LetOutRent > 0 || in.LetOutInterest > 0 {
		nav := pos(in.LetOutRent - in.LetOutMunicipalTax)
		letOut = nav*0.7 - pos(in.LetOutInterest)
	}
	if isOld {
		hp := letOut - math.Min(pos(in.SelfOccupiedInterest), r.Lim24bSelfOccupied)
		res.HouseProperty = math.Max(hp, -r.LimHPLossSetOff)
	} else {
		res.HouseProperty = pos(letOut) // no set-off of house property loss in the new regime
	}

	res.BusinessIncome = in.BusinessIncome

	fpCap := r.FamilyPensionNew
	if isOld {
		fpCap = r.FamilyPensionOld
	}
	fpDed := math.Min(pos(in.FamilyPension)/3, fpCap)
	otherGross := pos(in.InterestIncome) + pos(in.SavingsInterest) + pos(in.DividendIncome) + pos(in.OtherIncome) +
		pos(in.SlabCapitalGains) + pos(in.FamilyPension) - fpDed
	res.OtherSources = pos(otherGross - pos(in.MinorExemption))

	normalGTI := pos(res.SalaryIncome + res.HouseProperty + res.BusinessIncome + res.OtherSources)
	stcg, ltcg, ltcgOther := pos(in.STCGEquity), pos(in.LTCGEquity), pos(in.LTCGOther)
	res.SpecialIncome = stcg + ltcg + ltcgOther
	res.GrossTotalIncome = round(normalGTI + res.SpecialIncome)

	// Chapter VI-A deductions (only against normal income)
	var ded float64
	empNPSPct := r.EmployerNPSPctNew
	if isOld {
		empNPSPct = r.EmployerNPSPctOld
	}
	empNPS := math.Min(pos(in.EmployerNPS), empNPSPct/100*pos(in.BasicDA))
	if in.BasicDA <= 0 {
		empNPS = 0
	}
	if isOld {
		c80 := math.Min(pos(in.Ded80C), r.Lim80C)
		own := pos(in.OwnNPS)
		ccd1b := math.Min(own, r.Lim80CCD1B)
		ccd1Cap := r.EmployeeNPSPctSalaried / 100 * pos(in.BasicDA)
		if in.SelfEmployed || in.BasicDA <= 0 {
			ccd1Cap = r.SelfEmployedNPSPct / 100 * normalGTI
		}
		ccd1 := math.Min(math.Min(own-ccd1b, ccd1Cap), r.Lim80C-c80)
		line("80C", "Investments & payments (EPF, PPF, ELSS, LIC, principal…)", in.Ded80C, c80, r.Lim80C)
		line("80CCD(1)", "Own NPS (within ₹1.5L)", own-ccd1b, pos(ccd1), 0)
		line("80CCD(1B)", "Own NPS (additional)", own, ccd1b, r.Lim80CCD1B)
		ded += c80 + pos(ccd1) + ccd1b

		selfLim := r.Lim80DSelf
		if in.Age >= 60 || in.SelfSeniorHealth {
			selfLim = r.Lim80DSelfSenior
		}
		parLim := r.Lim80DParents
		if in.ParentsSenior {
			parLim = r.Lim80DParentsSenior
		}
		dSelf := math.Min(pos(in.Ded80DSelf), selfLim)
		dPar := math.Min(pos(in.Ded80DParents), parLim)
		line("80D", "Health insurance – self & family", in.Ded80DSelf, dSelf, selfLim)
		line("80D", "Health insurance – parents", in.Ded80DParents, dPar, parLim)
		ded += dSelf + dPar

		line("80E", "Education loan interest", in.Ded80E, pos(in.Ded80E), 0)
		line("80G", "Donations", in.Ded80G, pos(in.Ded80G), 0)
		ded += pos(in.Ded80E) + pos(in.Ded80G)

		if in.Resident && in.Age >= 60 {
			claim := pos(in.InterestIncome) + pos(in.SavingsInterest)
			a := math.Min(claim, r.Lim80TTB)
			line("80TTB", "Interest on deposits (senior citizen)", claim, a, r.Lim80TTB)
			ded += a
		} else {
			a := math.Min(pos(in.SavingsInterest), r.Lim80TTA)
			line("80TTA", "Savings account interest", in.SavingsInterest, a, r.Lim80TTA)
			ded += a
		}

		switch in.Disability {
		case "disabled":
			line("80U", "Disability", r.Lim80U, r.Lim80U, r.Lim80U)
			ded += r.Lim80U
		case "severe":
			line("80U", "Severe disability", r.Lim80USevere, r.Lim80USevere, r.Lim80USevere)
			ded += r.Lim80USevere
		}

		if in.RentPaid > 0 && in.HRAReceived <= 0 {
			ati := pos(normalGTI - ded)
			a := pos(math.Min(r.Lim80GGPerMonth*12, math.Min(0.25*ati, in.RentPaid-0.1*ati)))
			line("80GG", "Rent paid (no HRA)", in.RentPaid, a, r.Lim80GGPerMonth*12)
			ded += a
		}

		if in.ExtraDeductionOld > 0 {
			line("what-if", "Additional deductions (what-if)", in.ExtraDeductionOld, in.ExtraDeductionOld, 0)
			ded += in.ExtraDeductionOld
		}
	}
	line("80CCD(2)", "Employer NPS contribution", in.EmployerNPS, empNPS, 0)
	ded += empNPS

	res.TotalDeductions = round(math.Min(ded, normalGTI))
	normal := pos(normalGTI - res.TotalDeductions)
	res.NormalIncome = round(normal)
	res.TotalIncome = round(normal + res.SpecialIncome)

	tax := r.taxBeforeCess(regime, in.Age, in.Resident, normal, stcg, ltcg, ltcgOther, &res)
	res.Cess = tax * r.CessRate / 100
	res.TotalTax = round(tax + res.Cess)
	res.SlabTax, res.SpecialTax = round(res.SlabTax), round(res.SpecialTax)
	res.Rebate87A, res.Surcharge, res.Cess = round(res.Rebate87A), round(res.Surcharge), round(res.Cess)
	if res.GrossTotalIncome > 0 {
		res.EffectivePc = math.Round(res.TotalTax/res.GrossTotalIncome*10000) / 100
	}
	return res
}

// taxBeforeCess computes slab + special-rate tax, 87A rebate and surcharge
// (with marginal relief). When res is non-nil the components are recorded.
func (r *Rules) taxBeforeCess(regime Regime, age int, resident bool, normal, stcg, ltcg, ltcgOther float64, res *Result) float64 {
	slabs := r.slabsFor(regime, age, resident)
	st, lines := slabTax(normal, slabs)

	// Residents may use unused basic exemption against special-rate gains.
	unused := 0.0
	if resident {
		unused = pos(basicExemption(slabs) - normal)
	}
	use := func(v float64) float64 {
		u := math.Min(v, unused)
		unused -= u
		return v - u
	}
	stcgT := use(stcg)
	ltcgT := use(pos(ltcg - r.LTCGEquityExemption))
	ltcgOT := use(ltcgOther)
	stcgTax := stcgT * r.STCGEquityRate / 100
	special := stcgTax + ltcgT*r.LTCGEquityRate/100 + ltcgOT*r.LTCGOtherRate/100

	total := normal + stcg + ltcg + ltcgOther
	rebate := 0.0
	if resident {
		if regime == New {
			if total <= r.RebateNewLimit {
				rebate = math.Min(st, r.RebateNewMax)
			} else if excess := total - r.RebateNewLimit; st > excess {
				// Marginal relief: slab tax may not exceed income above the limit.
				rebate = st - excess
			}
		} else if total <= r.RebateOldLimit {
			rebate = math.Min(st+stcgTax, r.RebateOldMax)
		}
	}
	tax := pos(st + special - rebate)

	bands := r.SurchargeNew
	if regime == Old {
		bands = r.SurchargeOld
	}
	rate, threshold := 0.0, 0.0
	for _, b := range bands {
		if total > b.Above {
			rate, threshold = b.Rate, b.Above
		}
	}
	surcharge := 0.0
	if rate > 0 {
		normalTax := pos(st - rebate)
		surcharge = normalTax*rate/100 + special*math.Min(rate, r.SurchargeSpecialCap)/100
		// Marginal relief: tax + surcharge may exceed the tax at the threshold
		// by no more than the income above the threshold.
		over := total - threshold
		n2, s2, l2, lo2 := normal, stcg, ltcg, ltcgOther
		for _, p := range []*float64{&n2, &lo2, &l2, &s2} {
			cut := math.Min(*p, over)
			*p -= cut
			over -= cut
		}
		atThreshold := r.taxBeforeCess(regime, age, resident, n2, s2, l2, lo2, nil)
		if limit := atThreshold + (total - threshold); tax+surcharge > limit {
			surcharge = pos(limit - tax)
		}
	}
	if res != nil {
		res.SlabTax, res.SpecialTax, res.Rebate87A, res.Surcharge, res.Slabs = st, special, rebate, surcharge, lines
	}
	return tax + surcharge
}

// Total returns the total tax (incl. cess) for a regime.
func Total(in Input, regime Regime) float64 { return Compute(in, regime).TotalTax }

// Best returns the cheaper regime and its tax (new wins ties, being the default).
func Best(in Input) (Regime, float64) {
	o, n := Total(in, Old), Total(in, New)
	if o < n {
		return Old, o
	}
	return New, n
}

func marginal(in Input, regime Regime, base float64) float64 {
	in.OtherIncome += 10000
	return math.Round((Total(in, regime)-base)/10000*10000) / 100
}

// ZeroTaxRoom returns how much more slab-rate income (e.g. interest) the
// person can earn, up to limit, without paying any more tax in their best regime.
func ZeroTaxRoom(in Input, limit float64) float64 {
	_, base := Best(in)
	probe := func(x float64) bool {
		p := in
		p.InterestIncome += x
		_, t := Best(p)
		return t-base < 1
	}
	if probe(limit) {
		return limit
	}
	lo, hi := 0.0, limit
	for i := 0; i < 40 && hi-lo > 100; i++ {
		mid := (lo + hi) / 2
		if probe(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return math.Floor(lo/100) * 100
}

// Compare computes both regimes, the recommendation and planning aids.
func Compare(in Input) Comparison {
	r, _ := RulesFor(in.FY)
	c := Comparison{FY: r.FY, Act: r.Act, Old: Compute(in, Old), New: Compute(in, New)}
	c.Old.MarginalPc = marginal(in, Old, c.Old.TotalTax)
	c.New.MarginalPc = marginal(in, New, c.New.TotalTax)
	c.Recommended = New
	if c.Old.TotalTax < c.New.TotalTax {
		c.Recommended = Old
	}
	c.Savings = math.Abs(c.Old.TotalTax - c.New.TotalTax)

	if c.Recommended == New && c.Old.TotalTax > c.New.TotalTax {
		hi := c.Old.NormalIncome
		probe := in
		probe.ExtraDeductionOld = hi
		if Total(probe, Old) > c.New.TotalTax {
			c.BreakEvenDeductions = -1
		} else {
			lo := 0.0
			for i := 0; i < 40 && hi-lo > 100; i++ {
				mid := (lo + hi) / 2
				probe.ExtraDeductionOld = mid
				if Total(probe, Old) > c.New.TotalTax {
					lo = mid
				} else {
					hi = mid
				}
			}
			c.BreakEvenDeductions = math.Ceil(hi/1000) * 1000
		}
	}

	// Headroom under the old regime
	used := func(code, label string) float64 {
		for _, l := range c.Old.Deductions {
			if l.Code == code && (label == "" || l.Label == label) {
				return l.Allowed
			}
		}
		return 0
	}
	c80Pool := used("80C", "") + used("80CCD(1)", "")
	selfLim := r.Lim80DSelf
	if in.Age >= 60 || in.SelfSeniorHealth {
		selfLim = r.Lim80DSelfSenior
	}
	parLim := r.Lim80DParents
	if in.ParentsSenior {
		parLim = r.Lim80DParentsSenior
	}
	c.Headroom = Headroom{
		Sec80C:        pos(r.Lim80C - c80Pool),
		Sec80CCD1B:    pos(r.Lim80CCD1B - used("80CCD(1B)", "")),
		Sec80DSelf:    pos(selfLim - used("80D", "Health insurance – self & family")),
		Sec80DParents: pos(parLim - used("80D", "Health insurance – parents")),
		LTCGExemption: pos(r.LTCGEquityExemption - pos(in.LTCGEquity)),
		RebateNewRoom: r.RebateNewLimit - c.New.TotalIncome,
	}
	maxed := in
	maxed.Ded80C += c.Headroom.Sec80C
	maxed.OwnNPS += c.Headroom.Sec80CCD1B
	maxed.Ded80DSelf += c.Headroom.Sec80DSelf
	c.OldTaxWithHeadroom = Total(maxed, Old)
	return c
}
