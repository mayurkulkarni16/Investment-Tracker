package services

import (
	"context"
	"fmt"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type financeReminderSources struct {
	insuranceRepo *repository.InsuranceRepo
	budgets       *BudgetService
	documentRepo  *repository.DocumentRepo
}

// SetFinanceSources enables insurance premium/expiry, budget and document
// expiry reminders.
func (s *NotificationService) SetFinanceSources(insuranceRepo *repository.InsuranceRepo, budgets *BudgetService, documentRepo *repository.DocumentRepo) {
	s.finance = &financeReminderSources{insuranceRepo: insuranceRepo, budgets: budgets, documentRepo: documentRepo}
}

func (s *NotificationService) financeNotifications(ctx context.Context, userID string, now time.Time) []models.Notification {
	if s.finance == nil {
		return nil
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var out []models.Notification

	if s.finance.insuranceRepo != nil {
		policies, _ := s.finance.insuranceRepo.GetAll(ctx, userID)
		for i := range policies {
			p := &policies[i]
			if p.Status != "active" {
				continue
			}
			label := policyLabel(p)
			for _, d := range PremiumDates(p, today, today.AddDate(0, 0, 15)) {
				out = append(out, models.Notification{
					Type:          "insurance_premium",
					Title:         "Insurance Premium Due",
					Message:       fmt.Sprintf("%s premium of ₹%.0f due on %s", label, p.PremiumAmount, d.Format("02 Jan 2006")),
					Date:          d,
					ReferenceType: "insurance",
					ReferenceID:   p.ID.Hex(),
					DedupeKey:     dedupeKey("insurance_premium", p.ID.Hex(), dayKey(d)),
				})
			}
			if p.EndDate != nil && !p.EndDate.Before(today) && !p.EndDate.After(today.AddDate(0, 0, 30)) {
				out = append(out, models.Notification{
					Type:          "policy_expiry",
					Title:         "Policy Ending Soon",
					Message:       fmt.Sprintf("%s cover ends on %s. Renew it or arrange a replacement.", label, p.EndDate.Format("02 Jan 2006")),
					Date:          *p.EndDate,
					ReferenceType: "insurance",
					ReferenceID:   p.ID.Hex(),
					DedupeKey:     dedupeKey("policy_expiry", p.ID.Hex(), dayKey(*p.EndDate)),
				})
			}
		}
	}

	if s.finance.budgets != nil {
		if sum, err := s.finance.budgets.Summary(ctx, userID, 1); err == nil {
			month := now.Format("2006-01")
			for _, b := range sum.Budgets {
				if b.Status != "over" {
					continue
				}
				out = append(out, models.Notification{
					Type:          "budget_over",
					Title:         "Budget Exceeded",
					Message:       fmt.Sprintf("%s spending is ₹%.0f this month, over the ₹%.0f budget.", b.Category, b.Spent, b.Limit),
					Date:          now,
					ReferenceType: "budget",
					ReferenceID:   b.Category,
					DedupeKey:     dedupeKey("budget_over", b.Category, month),
				})
			}
		}
	}

	if s.finance.documentRepo != nil {
		docs, _ := s.finance.documentRepo.GetAll(ctx, userID)
		for _, d := range docs {
			if d.ExpiryDate == nil || d.ExpiryDate.Before(today) || d.ExpiryDate.After(today.AddDate(0, 0, 30)) {
				continue
			}
			out = append(out, models.Notification{
				Type:          "document_expiry",
				Title:         "Document Expiring",
				Message:       fmt.Sprintf("%s expires on %s.", d.Title, d.ExpiryDate.Format("02 Jan 2006")),
				Date:          *d.ExpiryDate,
				ReferenceType: "document",
				ReferenceID:   d.ID.Hex(),
				DedupeKey:     dedupeKey("document_expiry", d.ID.Hex(), dayKey(*d.ExpiryDate)),
			})
		}
	}
	return out
}
