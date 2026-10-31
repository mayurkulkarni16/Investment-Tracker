package services

import (
	"math"

	"investment-tracker/models"
)

// AssetProfile describes the long-term behaviour assumed for an asset class.
// ExpectedReturn is the median (geometric) annual return in %, Volatility is the
// annual standard deviation in %, and EquityShare is the fraction (0..1) that
// behaves like equity for allocation and correlation purposes.
type AssetProfile struct {
	EquityShare    float64
	ExpectedReturn float64
	Volatility     float64
}

// Long-term assumptions for Indian markets. Deliberately conservative: they are
// used as priors that recent fund performance is blended towards.
var (
	profileLargeCap    = AssetProfile{EquityShare: 1, ExpectedReturn: 11, Volatility: 16}
	profileDiversified = AssetProfile{EquityShare: 1, ExpectedReturn: 12, Volatility: 18}
	profileMidCap      = AssetProfile{EquityShare: 1, ExpectedReturn: 13, Volatility: 22}
	profileSmallCap    = AssetProfile{EquityShare: 1, ExpectedReturn: 14, Volatility: 26}
	profileSectoral    = AssetProfile{EquityShare: 1, ExpectedReturn: 12, Volatility: 25}
	profileHybrid      = AssetProfile{EquityShare: 0.65, ExpectedReturn: 10, Volatility: 11}
	profileDebt        = AssetProfile{EquityShare: 0, ExpectedReturn: 7, Volatility: 3}
	profileDynamicBond = AssetProfile{EquityShare: 0, ExpectedReturn: 7, Volatility: 4}
	profileGilt        = AssetProfile{EquityShare: 0, ExpectedReturn: 7, Volatility: 5}
	profileCash        = AssetProfile{EquityShare: 0, ExpectedReturn: 6.5, Volatility: 0.8}

	// StockProfile applies to individual listed stocks.
	StockProfile = AssetProfile{EquityShare: 1, ExpectedReturn: 12, Volatility: 25}
)

// MutualFundProfile returns the asset profile for a mutual fund category.
// Unknown categories are treated as diversified equity.
func MutualFundProfile(ft models.FundType) AssetProfile {
	switch ft {
	case models.FundTypeLargeCap, models.FundTypeIndex:
		return profileLargeCap
	case models.FundTypeEquity, models.FundTypeELSS, models.FundTypeFlexiCap, models.FundTypeMultiCap:
		return profileDiversified
	case models.FundTypeMidCap:
		return profileMidCap
	case models.FundTypeSmallCap:
		return profileSmallCap
	case models.FundTypeSectoral, models.FundTypeThematic:
		return profileSectoral
	case models.FundTypeHybrid:
		return profileHybrid
	case models.FundTypeDebt, models.FundTypeCorporateBond:
		return profileDebt
	case models.FundTypeDynamicBond:
		return profileDynamicBond
	case models.FundTypeGilt:
		return profileGilt
	case models.FundTypeLiquid, models.FundTypeOvernight, models.FundTypeMoneyMarket:
		return profileCash
	default:
		return profileDiversified
	}
}

// IsCashLikeFund reports whether a fund can be redeemed quickly with negligible
// price risk (suitable for an emergency fund).
func IsCashLikeFund(ft models.FundType) bool {
	return ft == models.FundTypeLiquid || ft == models.FundTypeOvernight || ft == models.FundTypeMoneyMarket
}

// NPSProfile blends asset profiles according to the account's scheme allocation.
func NPSProfile(nps *models.NPSAccount) AssetProfile {
	total := nps.EquityPct + nps.CorporateBondPct + nps.GovtSecPct + nps.AlternatePct
	if total <= 0 {
		return AssetProfile{EquityShare: 0.5, ExpectedReturn: 9.5, Volatility: 9}
	}
	w := func(pct float64) float64 { return pct / total }
	return AssetProfile{
		EquityShare:    w(nps.EquityPct),
		ExpectedReturn: w(nps.EquityPct)*11 + w(nps.CorporateBondPct)*8 + w(nps.GovtSecPct)*7.5 + w(nps.AlternatePct)*9,
		Volatility:     w(nps.EquityPct)*16 + w(nps.CorporateBondPct)*4 + w(nps.GovtSecPct)*5 + w(nps.AlternatePct)*8,
	}
}

// BlendedReturn shrinks an observed annualised return (XIRR, %) towards the
// long-term prior. Credibility grows with holding period but never exceeds 50%,
// so a short hot (or cold) streak cannot dominate a multi-year projection.
func BlendedReturn(observed, holdingYears float64, prior AssetProfile) float64 {
	if holdingYears <= 0 {
		return prior.ExpectedReturn
	}
	if observed < -10 {
		observed = -10
	} else if observed > 30 {
		observed = 30
	}
	w := holdingYears / (holdingYears + 5)
	if w > 0.5 {
		w = 0.5
	}
	return round1(w*observed + (1-w)*prior.ExpectedReturn)
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}
