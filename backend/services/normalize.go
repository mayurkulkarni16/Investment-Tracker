package services

import (
	"math"
	"time"

	"investment-tracker/models"
)

// Bonds and FDs derive part of their state at read time (past payouts count as
// received, an FD past its maturity date is matured). These helpers apply that
// derivation so every feature that reads them from the repository directly
// (dashboard, net worth, cashflow, tax, goals, export, projections) sees the same
// state as the Bonds and FD pages.

// NormalizeFixedDeposit marks past interest payouts as received and derives the
// matured status for FDs past their maturity date (unless auto-renewed).
func NormalizeFixedDeposit(fd *models.FixedDeposit, now time.Time) {
	markPastFDPayouts(fd, now)
	if fd.Status == models.FDActive && !fd.IsAutoRenewed && !fd.MaturityDate.IsZero() && !now.Before(fd.MaturityDate) {
		fd.Status = models.FDMatured
	}
}

func markPastFDPayouts(fd *models.FixedDeposit, now time.Time) {
	for i := range fd.InterestPayouts {
		p := &fd.InterestPayouts[i]
		if p.Status == models.PayoutPending && !p.ScheduledDate.After(now) {
			p.Status = models.PayoutReceived
			d := p.ScheduledDate
			p.ReceivedDate = &d
		}
	}
}

// FixedDepositCurrentValue is the value still held in the FD today: accrued value
// for active FDs, zero once it has matured or been closed.
func FixedDepositCurrentValue(fd *models.FixedDeposit, now time.Time) float64 {
	if fd.Status != models.FDActive {
		return 0
	}
	return fdValueAt(fd, now, now)
}

// FixedDepositCashReturned is the cash the FD has already paid out: received
// interest payouts plus the maturity proceeds once matured.
func FixedDepositCashReturned(fd *models.FixedDeposit) float64 {
	total := 0.0
	for _, p := range fd.InterestPayouts {
		if p.Status == models.PayoutReceived {
			total += p.Amount
		}
	}
	switch fd.Status {
	case models.FDMatured:
		if fd.InterestType == models.InterestCumulative && fd.MaturityAmount > 0 {
			total += fd.MaturityAmount
		} else {
			total += fd.PrincipalAmount
		}
	case models.FDPrematureClosed:
		total += fd.PrincipalAmount
	}
	return total
}

// FixedDepositCashflows returns the FD's actual cash movements (for XIRR).
func FixedDepositCashflows(fd *models.FixedDeposit) []Cashflow {
	flows := []Cashflow{{Date: fd.StartDate, Amount: -fd.PrincipalAmount}}
	for _, p := range fd.InterestPayouts {
		if p.Status == models.PayoutReceived {
			d := p.ScheduledDate
			if p.ReceivedDate != nil {
				d = *p.ReceivedDate
			}
			flows = append(flows, Cashflow{Date: d, Amount: p.Amount})
		}
	}
	if fd.Status == models.FDMatured {
		proceeds := fd.PrincipalAmount
		if fd.InterestType == models.InterestCumulative && fd.MaturityAmount > 0 {
			proceeds = fd.MaturityAmount
		}
		flows = append(flows, Cashflow{Date: fd.MaturityDate, Amount: proceeds})
	}
	return flows
}

// NormalizeCorporateBond marks past payouts/repayments as received and
// recomputes remaining principal, totals and status.
func NormalizeCorporateBond(bond *models.CorporateBond) {
	(&CorporateBondService{}).recalculateBond(bond)
}

// CorporateBondCashReturned is interest received plus principal repaid.
func CorporateBondCashReturned(bond *models.CorporateBond) float64 {
	return bond.TotalInterestEarned + bond.TotalPrincipalReturned
}

// FixedDepositInterestBetween is the interest an FD earns in [from, to): the
// growth in accrued value for cumulative FDs, or the payouts scheduled in the
// period for payout FDs.
func FixedDepositInterestBetween(fd *models.FixedDeposit, from, to time.Time) float64 {
	if fd.InterestType == models.InterestCumulative {
		start, end := fd.StartDate, fd.MaturityDate
		if from.After(start) {
			start = from
		}
		if to.Before(end) {
			end = to
		}
		if !end.After(start) {
			return 0
		}
		return math.Max(0, fdValueAt(fd, start, end)-fdValueAt(fd, start, start))
	}
	total := 0.0
	for _, p := range fd.InterestPayouts {
		if !p.ScheduledDate.Before(from) && p.ScheduledDate.Before(to) {
			total += p.Amount
		}
	}
	return total
}
