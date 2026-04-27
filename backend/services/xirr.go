package services

import (
	"errors"
	"math"
	"sort"
	"time"
)

// Cashflow represents a single cashflow for XIRR calculation.
// Negative amounts are outflows (investments), positive are inflows (returns).
type Cashflow struct {
	Date   time.Time
	Amount float64
}

// CalculateXIRR computes the extended internal rate of return for irregular cashflows
// using the Newton-Raphson method. Returns the annualized rate as a decimal (e.g., 0.12 = 12%).
func CalculateXIRR(cashflows []Cashflow) (float64, error) {
	if len(cashflows) < 2 {
		return 0, errors.New("need at least 2 cashflows")
	}

	// Sort by date
	sorted := make([]Cashflow, len(cashflows))
	copy(sorted, cashflows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	// Check we have both positive and negative cashflows
	hasPos, hasNeg := false, false
	for _, cf := range sorted {
		if cf.Amount > 0 {
			hasPos = true
		} else if cf.Amount < 0 {
			hasNeg = true
		}
	}
	if !hasPos || !hasNeg {
		return 0, errors.New("need both positive and negative cashflows")
	}

	d0 := sorted[0].Date

	// NPV function: sum of cf_i / (1+rate)^(days_i/365)
	npv := func(rate float64) float64 {
		total := 0.0
		for _, cf := range sorted {
			days := cf.Date.Sub(d0).Hours() / 24.0
			exp := days / 365.0
			denom := math.Pow(1.0+rate, exp)
			if denom == 0 {
				return math.MaxFloat64
			}
			total += cf.Amount / denom
		}
		return total
	}

	// Derivative of NPV w.r.t. rate
	dnpv := func(rate float64) float64 {
		total := 0.0
		for _, cf := range sorted {
			days := cf.Date.Sub(d0).Hours() / 24.0
			exp := days / 365.0
			denom := math.Pow(1.0+rate, exp+1)
			if denom == 0 {
				return math.MaxFloat64
			}
			total -= cf.Amount * exp / denom
		}
		return total
	}

	// Newton-Raphson iteration
	rate := 0.1 // initial guess 10%
	const maxIter = 200
	const tolerance = 1e-9

	for i := 0; i < maxIter; i++ {
		fx := npv(rate)
		dfx := dnpv(rate)

		if math.Abs(dfx) < 1e-12 {
			break
		}

		newRate := rate - fx/dfx

		// Clamp to avoid divergence
		if newRate < -0.999 {
			newRate = -0.999
		}
		if newRate > 100 {
			newRate = 100
		}

		if math.Abs(newRate-rate) < tolerance {
			return newRate, nil
		}
		rate = newRate
	}

	// If Newton-Raphson didn't converge, try bisection
	lo, hi := -0.99, 10.0
	if npv(lo)*npv(hi) > 0 {
		// Try wider range
		hi = 100.0
		if npv(lo)*npv(hi) > 0 {
			return rate, nil // Return best Newton-Raphson estimate
		}
	}

	for i := 0; i < 1000; i++ {
		mid := (lo + hi) / 2.0
		fMid := npv(mid)
		if math.Abs(fMid) < tolerance || (hi-lo)/2.0 < tolerance {
			return mid, nil
		}
		if npv(lo)*fMid < 0 {
			hi = mid
		} else {
			lo = mid
		}
	}

	return rate, nil
}
