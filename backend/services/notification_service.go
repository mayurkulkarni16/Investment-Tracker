package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type NotificationService struct {
	repo              *repository.NotificationRepo
	homeLoanRepo      *repository.HomeLoanRepo
	personalLoanRepo  *repository.PersonalLoanRepo
	fdRepo            *repository.FixedDepositRepo
	bondRepo          *repository.CorporateBondRepo
	sipRepo           *repository.SIPRepo
	creditCardRepo    *repository.CreditCardRepo
	vehicleRentalRepo *repository.VehicleRentalRepo
	finance           *financeReminderSources
	goalService       *GoalService
}

func NewNotificationService(
	repo *repository.NotificationRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	fdRepo *repository.FixedDepositRepo,
	bondRepo *repository.CorporateBondRepo,
	sipRepo *repository.SIPRepo,
	creditCardRepo *repository.CreditCardRepo,
	vehicleRentalRepo *repository.VehicleRentalRepo,
	goalService *GoalService,
) *NotificationService {
	return &NotificationService{
		repo: repo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		fdRepo: fdRepo, bondRepo: bondRepo, sipRepo: sipRepo, creditCardRepo: creditCardRepo,
		vehicleRentalRepo: vehicleRentalRepo,
		goalService:       goalService,
	}
}

func (s *NotificationService) GetAll(ctx context.Context, userID string) ([]models.Notification, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *NotificationService) GetUnread(ctx context.Context, userID string) ([]models.Notification, error) {
	return s.repo.GetUnread(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, objID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// PurgeOld deletes read notifications older than the given age (all users).
func (s *NotificationService) PurgeOld(ctx context.Context, age time.Duration) error {
	return s.repo.DeleteOlderThan(ctx, time.Now().Add(-age))
}

// GenerateNotifications scans all modules and creates upcoming reminders
func (s *NotificationService) GenerateNotifications(ctx context.Context, userID string) ([]models.Notification, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	upcoming := today.AddDate(0, 0, 7) // 7 days ahead
	var generated []models.Notification

	// Home Loan EMIs
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx, userID)
	for _, loan := range homeLoans {
		if loan.Status != "active" {
			continue
		}
		nextEMI := loan.EMIStartDate
		if len(loan.EMIsPaid) > 0 {
			last := loan.EMIsPaid[len(loan.EMIsPaid)-1]
			nextEMI = last.DueDate.AddDate(0, 1, 0)
		}
		if !nextEMI.Before(now) && !nextEMI.After(upcoming) {
			generated = append(generated, models.Notification{
				Type:          "emi_due",
				Title:         "Home Loan EMI Due",
				Message:       fmt.Sprintf("%s EMI of ₹%.0f due on %s", loan.BankName, loan.EMIAmount, nextEMI.Format("02 Jan 2006")),
				Date:          nextEMI,
				ReferenceType: "home_loan",
				ReferenceID:   loan.ID.Hex(),
				DedupeKey:     dedupeKey("emi_due", "home_loan", loan.ID.Hex(), dayKey(nextEMI)),
			})
		}
	}

	// Personal Loan EMIs
	pLoans, _ := s.personalLoanRepo.GetAll(ctx, userID)
	for _, loan := range pLoans {
		if loan.Status != "active" {
			continue
		}
		nextEMI := loan.EMIStartDate
		if len(loan.EMIsPaid) > 0 {
			last := loan.EMIsPaid[len(loan.EMIsPaid)-1]
			nextEMI = last.DueDate.AddDate(0, 1, 0)
		}
		if !nextEMI.Before(now) && !nextEMI.After(upcoming) {
			generated = append(generated, models.Notification{
				Type:          "emi_due",
				Title:         "Personal Loan EMI Due",
				Message:       fmt.Sprintf("%s EMI of ₹%.0f due on %s", loan.LenderName, loan.EMIAmount, nextEMI.Format("02 Jan 2006")),
				Date:          nextEMI,
				ReferenceType: "personal_loan",
				ReferenceID:   loan.ID.Hex(),
				DedupeKey:     dedupeKey("emi_due", "personal_loan", loan.ID.Hex(), dayKey(nextEMI)),
			})
		}
	}

	// FD Maturities
	fds, _ := s.fdRepo.GetAll(ctx, userID)
	for _, fd := range fds {
		if fd.Status == "active" && !fd.MaturityDate.Before(now) && !fd.MaturityDate.After(now.AddDate(0, 0, 30)) {
			generated = append(generated, models.Notification{
				Type:          "fd_maturity",
				Title:         "FD Maturing Soon",
				Message:       fmt.Sprintf("%s FD maturing on %s (₹%.0f)", fd.BankName, fd.MaturityDate.Format("02 Jan 2006"), fd.MaturityAmount),
				Date:          fd.MaturityDate,
				ReferenceType: "fixed_deposit",
				ReferenceID:   fd.ID.Hex(),
				DedupeKey:     dedupeKey("fd_maturity", fd.ID.Hex(), dayKey(fd.MaturityDate)),
			})
		}
	}

	// Bond Coupons
	bonds, _ := s.bondRepo.GetAll(ctx, userID)
	for _, bond := range bonds {
		for _, p := range bond.InterestPayouts {
			if p.Status == "pending" && !p.ScheduledDate.Before(now) && !p.ScheduledDate.After(now.AddDate(0, 0, 30)) {
				generated = append(generated, models.Notification{
					Type:          "bond_coupon",
					Title:         "Bond Interest Payout",
					Message:       fmt.Sprintf("%s interest of ₹%.0f expected on %s", bond.BondName, p.Amount, p.ScheduledDate.Format("02 Jan 2006")),
					Date:          p.ScheduledDate,
					ReferenceType: "corporate_bond",
					ReferenceID:   bond.ID.Hex(),
					DedupeKey:     dedupeKey("bond_coupon", bond.ID.Hex(), p.PayoutID),
				})
			}
		}
	}

	// Vehicle rental payouts: alert on overdue payouts and an upcoming buyback
	rentals, _ := s.vehicleRentalRepo.GetAll(ctx, userID)
	for i := range rentals {
		vr := &rentals[i]
		ComputeVehicleRental(vr, now)
		name := vr.Provider
		if vr.VehicleModel != "" {
			name += " " + vr.VehicleModel
		}
		if vr.OverdueCount > 0 {
			// Keyed on the latest overdue payout: each newly missed month alerts once.
			latestOverdue := ""
			for _, p := range vr.Payouts {
				if p.IsOverdue {
					latestOverdue = p.PayoutID
				}
			}
			generated = append(generated, models.Notification{
				Type:          "rental_overdue",
				Title:         "Rental Payout Overdue",
				Message:       fmt.Sprintf("%s: %d payout(s) totalling ₹%.0f not received. Follow up with the provider.", name, vr.OverdueCount, vr.OverdueAmount),
				Date:          now,
				ReferenceType: "vehicle_rental",
				ReferenceID:   vr.ID.Hex(),
				DedupeKey:     dedupeKey("rental_overdue", vr.ID.Hex(), latestOverdue),
			})
		}
		if vr.BuybackAmount > 0 && vr.BuybackStatus == models.PayoutPending && !vr.AgreementEndDate.Before(now) && !vr.AgreementEndDate.After(now.AddDate(0, 0, 30)) {
			generated = append(generated, models.Notification{
				Type:          "rental_buyback",
				Title:         "Rental Agreement Ending",
				Message:       fmt.Sprintf("%s agreement ends on %s. Buyback of ₹%.0f is due.", name, vr.AgreementEndDate.Format("02 Jan 2006"), vr.BuybackAmount),
				Date:          vr.AgreementEndDate,
				ReferenceType: "vehicle_rental",
				ReferenceID:   vr.ID.Hex(),
				DedupeKey:     dedupeKey("rental_buyback", vr.ID.Hex(), dayKey(vr.AgreementEndDate)),
			})
		}
	}

	// SIP Due Dates
	sips, _ := s.sipRepo.GetAll(ctx, userID)
	for _, sip := range sips {
		if sip.Status != "active" || sip.SIPDate < 1 {
			continue
		}
		nextSIP := addMonthsClamped(time.Date(now.Year(), now.Month(), sip.SIPDate, 0, 0, 0, 0, now.Location()), 0)
		if nextSIP.Before(today) {
			nextSIP = addMonthsClamped(time.Date(now.Year(), now.Month(), sip.SIPDate, 0, 0, 0, 0, now.Location()), 1)
		}
		if sip.EndDate != nil && nextSIP.After(*sip.EndDate) {
			continue
		}
		if !nextSIP.After(upcoming) {
			generated = append(generated, models.Notification{
				Type:          "sip_due",
				Title:         "SIP Due",
				Message:       fmt.Sprintf("%s SIP of ₹%.0f due on %s", sip.FundName, sip.Amount, nextSIP.Format("02 Jan 2006")),
				Date:          nextSIP,
				ReferenceType: "sip",
				ReferenceID:   sip.ID.Hex(),
				DedupeKey:     dedupeKey("sip_due", sip.ID.Hex(), dayKey(nextSIP)),
			})
		}
	}

	// Credit Card Due Dates
	cards, _ := s.creditCardRepo.GetAll(ctx, userID)
	for _, card := range cards {
		if card.Status != "active" {
			continue
		}
		for _, stmt := range card.Statements {
			if !stmt.IsPaid && !stmt.DueDate.Before(today) && !stmt.DueDate.After(now.AddDate(0, 0, 15)) {
				generated = append(generated, models.Notification{
					Type:          "credit_card_due",
					Title:         "Credit Card Payment Due",
					Message:       fmt.Sprintf("%s %s statement of ₹%.0f due on %s", card.BankName, stmt.Month, stmt.TotalAmount, stmt.DueDate.Format("02 Jan 2006")),
					Date:          stmt.DueDate,
					ReferenceType: "credit_card",
					ReferenceID:   card.ID.Hex(),
					DedupeKey:     dedupeKey("credit_card_due", card.ID.Hex(), stmt.StatementID),
				})
			}
		}
	}

	// Goal Milestones
	if s.goalService != nil {
		goals, _ := s.goalService.GetAll(ctx, userID)
		for _, g := range goals {
			if g.Status != "active" {
				continue
			}
			// Alert (at most monthly) if the target date is within 90 days and the goal is not on track
			if !g.OnTrack && !g.TargetDate.Before(now) && !g.TargetDate.After(now.AddDate(0, 0, 90)) {
				generated = append(generated, models.Notification{
					Type:          "goal_milestone",
					Title:         "Goal At Risk",
					Message:       fmt.Sprintf("%s is %.0f%% complete but target date %s is approaching. Projected shortfall: ₹%.0f", g.Name, g.ProgressPct, g.TargetDate.Format("02 Jan 2006"), g.Shortfall),
					Date:          now,
					ReferenceType: "goal",
					ReferenceID:   g.ID.Hex(),
					DedupeKey:     dedupeKey("goal_at_risk", g.ID.Hex(), now.Format("2006-01")),
				})
			}
			// Alert once for the highest milestone reached (25%, 50%, 75%, 90%)
			for _, pct := range []float64{90, 75, 50, 25} {
				if g.ProgressPct >= pct && g.ProgressPct < 100 {
					generated = append(generated, models.Notification{
						Type:          "goal_milestone",
						Title:         fmt.Sprintf("Goal %.0f%% Reached!", pct),
						Message:       fmt.Sprintf("%s has reached %.0f%% of its ₹%.0f target", g.Name, g.ProgressPct, g.TargetAmount),
						Date:          now,
						ReferenceType: "goal",
						ReferenceID:   g.ID.Hex(),
						DedupeKey:     dedupeKey("goal_milestone", g.ID.Hex(), fmt.Sprintf("%.0f", pct)),
					})
					break
				}
			}
		}
	}

	generated = append(generated, s.financeNotifications(ctx, userID, now)...)

	for i := range generated {
		generated[i].UserID = userID
		if _, err := s.repo.Upsert(ctx, &generated[i]); err != nil {
			return nil, err
		}
	}
	if generated == nil {
		generated = []models.Notification{}
	}
	return generated, nil
}

func dedupeKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}
