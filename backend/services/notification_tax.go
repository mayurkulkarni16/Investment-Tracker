package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"investment-tracker/models"
	"investment-tracker/taxengine"
)

// SetTaxSource enables regime-declaration, deduction-deadline and advance-tax reminders.
func (s *NotificationService) SetTaxSource(f *FamilyTaxService) { s.familyTax = f }

type advanceTaxDue struct {
	date time.Time
	pct  float64 // cumulative share of the year's tax due by this date
}

func advanceTaxSchedule(fyStart time.Time) []advanceTaxDue {
	y := fyStart.Year()
	loc := fyStart.Location()
	return []advanceTaxDue{
		{time.Date(y, 6, 15, 0, 0, 0, 0, loc), 15},
		{time.Date(y, 9, 15, 0, 0, 0, 0, loc), 45},
		{time.Date(y, 12, 15, 0, 0, 0, 0, loc), 75},
		{time.Date(y+1, 3, 15, 0, 0, 0, 0, loc), 100},
	}
}

func (s *NotificationService) taxNotifications(ctx context.Context, userID string, now time.Time) []models.Notification {
	if s.familyTax == nil {
		return nil
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fyStart := getCurrentFYStart(now)
	declareWindow := now.Month() == time.April && now.Day() <= 20
	deadlineWindow := now.Month() >= time.January && now.Month() <= time.March
	var dueSoon *advanceTaxDue
	for _, d := range advanceTaxSchedule(fyStart) {
		if !today.After(d.date) && today.After(d.date.AddDate(0, 0, -10)) {
			d := d
			dueSoon = &d
		}
	}
	if !declareWindow && !deadlineWindow && dueSoon == nil {
		return nil
	}
	rep, err := s.familyTax.Report(ctx, userID, "")
	if err != nil {
		return nil
	}
	var out []models.Notification
	for _, m := range rep.Members {
		if !m.HasIncomeRecord {
			continue
		}
		cmp := m.Comparison
		salaried := m.Occupation == "salaried_private" || m.Occupation == "salaried_govt" || m.Income.SalaryFixed > 0
		if declareWindow && salaried {
			msg := fmt.Sprintf("%s: the %s regime looks cheaper for FY %s (saves about %s). Declare it to your employer so TDS is right.",
				m.Name, cmp.Recommended, rep.FY, formatINR(cmp.Savings))
			out = append(out, models.Notification{
				Type: "tax_regime", Title: "Declare Your Tax Regime", Message: msg, Date: now,
				ReferenceType: "profile", ReferenceID: m.ProfileID,
				DedupeKey: dedupeKey("tax_regime", m.ProfileID, rep.FY),
			})
		}
		h := cmp.Headroom
		if deadlineWindow && m.ChosenRegime == taxengine.Old && h.Sec80C+h.Sec80CCD1B+h.Sec80DSelf >= 5000 {
			msg := fmt.Sprintf("%s has %s of 80C, %s of NPS 80CCD(1B) and %s of 80D room left. Invest and submit proofs before 31 March.",
				m.Name, formatINR(h.Sec80C), formatINR(h.Sec80CCD1B), formatINR(h.Sec80DSelf))
			out = append(out, models.Notification{
				Type: "tax_deadline", Title: "Tax-Saving Deadline", Message: msg, Date: now,
				ReferenceType: "profile", ReferenceID: m.ProfileID,
				DedupeKey: dedupeKey("tax_deadline", m.ProfileID, rep.FY),
			})
		}
		if dueSoon != nil {
			// Senior citizens without business income are exempt from advance tax.
			exempt := m.Age >= 60 && m.Income.Business == 0
			tds := m.TaxPaid
			liability := m.ChosenTax - tds
			if !exempt && liability >= 10000 {
				due := math.Round(m.ChosenTax * dueSoon.pct / 100)
				msg := fmt.Sprintf("%s: pay advance tax so that at least %.0f%% of the estimated %s (%s) is paid by %s. Tax already paid/deducted: %s.",
					m.Name, dueSoon.pct, formatINR(m.ChosenTax), formatINR(due), dueSoon.date.Format("02 Jan"), formatINR(tds))
				out = append(out, models.Notification{
					Type: "advance_tax", Title: "Advance Tax Due", Message: msg, Date: dueSoon.date,
					ReferenceType: "profile", ReferenceID: m.ProfileID,
					DedupeKey: dedupeKey("advance_tax", m.ProfileID, dayKey(dueSoon.date)),
				})
			}
		}
	}
	return out
}
