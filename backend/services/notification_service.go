package services

import (
	"context"
	"fmt"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type NotificationService struct {
	repo             *repository.NotificationRepo
	homeLoanRepo     *repository.HomeLoanRepo
	personalLoanRepo *repository.PersonalLoanRepo
	fdRepo           *repository.FixedDepositRepo
	bondRepo         *repository.CorporateBondRepo
	sipRepo          *repository.SIPRepo
	creditCardRepo   *repository.CreditCardRepo
	goalService      *GoalService
}

func NewNotificationService(
	repo *repository.NotificationRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	fdRepo *repository.FixedDepositRepo,
	bondRepo *repository.CorporateBondRepo,
	sipRepo *repository.SIPRepo,
	creditCardRepo *repository.CreditCardRepo,
	goalService *GoalService,
) *NotificationService {
	return &NotificationService{
		repo: repo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		fdRepo: fdRepo, bondRepo: bondRepo, sipRepo: sipRepo, creditCardRepo: creditCardRepo,
		goalService: goalService,
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

// GenerateNotifications scans all modules and creates upcoming reminders
func (s *NotificationService) GenerateNotifications(ctx context.Context, userID string) ([]models.Notification, error) {
	now := time.Now()
	upcoming := now.AddDate(0, 0, 7) // 7 days ahead
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
			n := models.Notification{
				Type:          "emi_due",
				Title:         "Home Loan EMI Due",
				Message:       fmt.Sprintf("%s EMI of ₹%.0f due on %s", loan.BankName, loan.EMIAmount, nextEMI.Format("02 Jan 2006")),
				Date:          nextEMI,
				ReferenceType: "home_loan",
				ReferenceID:   loan.ID.Hex(),
			}
			n.ID = [12]byte{}
			generated = append(generated, n)
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
				})
			}
		}
	}

	// SIP Due Dates
	sips, _ := s.sipRepo.GetAll(ctx, userID)
	for _, sip := range sips {
		if sip.Status != "active" {
			continue
		}
		nextSIP := time.Date(now.Year(), now.Month(), sip.SIPDate, 0, 0, 0, 0, time.UTC)
		if nextSIP.Before(now) {
			nextSIP = nextSIP.AddDate(0, 1, 0)
		}
		if !nextSIP.After(upcoming) {
			generated = append(generated, models.Notification{
				Type:          "sip_due",
				Title:         "SIP Due",
				Message:       fmt.Sprintf("%s SIP of ₹%.0f due on %s", sip.FundName, sip.Amount, nextSIP.Format("02 Jan 2006")),
				Date:          nextSIP,
				ReferenceType: "sip",
				ReferenceID:   sip.ID.Hex(),
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
			if !stmt.IsPaid && !stmt.DueDate.Before(now) && !stmt.DueDate.After(now.AddDate(0, 0, 15)) {
				generated = append(generated, models.Notification{
					Type:          "credit_card_due",
					Title:         "Credit Card Payment Due",
					Message:       fmt.Sprintf("%s %s statement of ₹%.0f due on %s", card.BankName, stmt.Month, stmt.TotalAmount, stmt.DueDate.Format("02 Jan 2006")),
					Date:          stmt.DueDate,
					ReferenceType: "credit_card",
					ReferenceID:   card.ID.Hex(),
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
			// Alert if goal target date is within 90 days and not on track
			if !g.OnTrack && !g.TargetDate.Before(now) && !g.TargetDate.After(now.AddDate(0, 0, 90)) {
				generated = append(generated, models.Notification{
					Type:          "goal_milestone",
					Title:         "Goal At Risk",
					Message:       fmt.Sprintf("%s is %.0f%% complete but target date %s is approaching. Shortfall: ₹%.0f", g.Name, g.ProgressPct, g.TargetDate.Format("02 Jan 2006"), g.Shortfall),
					Date:          g.TargetDate,
					ReferenceType: "goal",
					ReferenceID:   g.ID.Hex(),
				})
			}
			// Alert on milestone reached (25%, 50%, 75%, 90%)
			for _, pct := range []float64{25, 50, 75, 90} {
				if g.ProgressPct >= pct && g.ProgressPct < pct+5 {
					generated = append(generated, models.Notification{
						Type:          "goal_milestone",
						Title:         fmt.Sprintf("Goal %.0f%% Reached!", pct),
						Message:       fmt.Sprintf("%s has reached %.0f%% of its ₹%.0f target", g.Name, g.ProgressPct, g.TargetAmount),
						Date:          now,
						ReferenceType: "goal",
						ReferenceID:   g.ID.Hex(),
					})
					break
				}
			}
		}
	}

	// Store generated notifications
	for i := range generated {
		generated[i].ID = [12]byte{} // reset for insert
		generated[i].UserID = userID
		n := generated[i]
		n.CreatedAt = time.Now()
		uid := uuid.New().String()
		_ = uid
		_ = s.repo.Create(ctx, &n)
	}

	return generated, nil
}
