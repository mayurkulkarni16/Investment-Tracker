package services

import (
	"fmt"
	"math"
	"time"

	"investment-tracker/models"
)

// Calendar dates are stored at UTC midnight; billing spans whole days, not elapsed local hours.
func altCalendarDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func altPayoutDay(month time.Time, offset int) time.Time {
	return time.Date(month.Year(), month.Month()+time.Month(offset), 5, 0, 0, 0, 0, time.UTC)
}

func altBillingStart(v *models.AlternateInvestment) time.Time {
	return altCalendarDate(v.PurchaseDate).AddDate(0, 0, 15)
}

func useAltCalendarSchedule(v *models.AlternateInvestment) {
	v.ScheduleVersion = models.AltCalendarScheduleVersion
	start := v.PurchaseDate
	if v.AssetType == models.AltAssetLivestock {
		start = altBillingStart(v)
	}
	v.PayoutStartDate = altPayoutDay(start, 1)
}

func altScheduleCount(v *models.AlternateInvestment, now time.Time) int {
	if v.TenureMonths <= 0 {
		return openEndedPayoutCount(v, now)
	}
	if v.ScheduleVersion == models.AltCalendarScheduleVersion && v.AssetType == models.AltAssetLivestock {
		start := altBillingStart(v)
		lastDay := addMonthsClamped(start, v.TenureMonths).AddDate(0, 0, -1)
		return (lastDay.Year()-start.Year())*12 + int(lastDay.Month()) - int(start.Month()) + 1
	}
	return v.TenureMonths
}

func altScheduledPayout(v *models.AlternateInvestment, index int) (time.Time, float64) {
	d := addMonthsClamped(v.PayoutStartDate, index)
	monthly := float64(v.Units) * v.MonthlyPayoutPerUnit
	if v.ScheduleVersion != models.AltCalendarScheduleVersion || v.AssetType != models.AltAssetLivestock {
		return d, round2(monthly)
	}

	start := altBillingStart(v)
	month := time.Date(start.Year(), start.Month()+time.Month(index), 1, 0, 0, 0, 0, time.UTC)
	next := month.AddDate(0, 1, 0)
	first, end := month, next
	if start.After(first) {
		first = start
	}
	if v.TenureMonths > 0 {
		termEnd := addMonthsClamped(start, v.TenureMonths)
		if termEnd.Before(end) {
			end = termEnd
		}
	}
	coveredDays := end.Sub(first).Hours() / 24
	daysInMonth := next.Sub(month).Hours() / 24
	return d, round2(monthly * coveredDays / daysInMonth)
}

func altLegacyCompleted(v *models.AlternateInvestment) bool {
	if v.TenureMonths <= 0 || len(v.Payouts) != v.TenureMonths ||
		(v.BuybackAmount > 0 && v.BuybackStatus != models.PayoutReceived) {
		return false
	}
	for _, p := range v.Payouts {
		if p.Status != models.PayoutReceived {
			return false
		}
	}
	return true
}

// UpgradeAlternateInvestmentSchedule rewrites pending installments of a legacy active
// record. It is also used on restore so a pre-upgrade backup cannot undo the change.
func UpgradeAlternateInvestmentSchedule(v *models.AlternateInvestment, now time.Time) (bool, error) {
	if v.ScheduleVersion == models.AltCalendarScheduleVersion {
		return false, nil
	}
	if v.ScheduleVersion != 0 {
		return false, fmt.Errorf("unsupported payout schedule version %d", v.ScheduleVersion)
	}
	if altLegacyCompleted(v) {
		return false, nil
	}
	monthly := float64(v.Units) * v.MonthlyPayoutPerUnit
	if v.PurchaseDate.IsZero() || v.Units < 1 || monthly <= 0 || math.IsNaN(monthly) || math.IsInf(monthly, 0) ||
		v.TenureMonths < 0 || v.TenureMonths > 240 {
		return false, fmt.Errorf("invalid legacy purchase date, monthly payout or tenure")
	}
	switch v.AssetType {
	case "", models.AltAssetVehicle, models.AltAssetLivestock, models.AltAssetOther:
	default:
		return false, fmt.Errorf("unknown legacy asset type %q", v.AssetType)
	}
	if v.TenureMonths > 0 && len(v.Payouts) != v.TenureMonths {
		return false, fmt.Errorf("legacy payout count %d does not match tenure %d", len(v.Payouts), v.TenureMonths)
	}
	ids := make(map[string]bool, len(v.Payouts))
	for i, p := range v.Payouts {
		if p.PayoutID == "" || ids[p.PayoutID] || p.ScheduledDate.IsZero() ||
			(i > 0 && !p.ScheduledDate.After(v.Payouts[i-1].ScheduledDate)) ||
			(p.Status != models.PayoutPending && p.Status != models.PayoutReceived) {
			return false, fmt.Errorf("invalid legacy payout at installment %d", i+1)
		}
		ids[p.PayoutID] = true
	}
	useAltCalendarSchedule(v)
	v.Payouts = buildAltPayoutSchedule(v, v.Payouts, now)
	return true, nil
}
