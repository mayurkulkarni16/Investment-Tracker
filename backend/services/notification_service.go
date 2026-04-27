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
}

func NewNotificationService(
	repo *repository.NotificationRepo,
	homeLoanRepo *repository.HomeLoanRepo,
	personalLoanRepo *repository.PersonalLoanRepo,
	fdRepo *repository.FixedDepositRepo,
	bondRepo *repository.CorporateBondRepo,
	sipRepo *repository.SIPRepo,
	creditCardRepo *repository.CreditCardRepo,
) *NotificationService {
	return &NotificationService{
		repo: repo, homeLoanRepo: homeLoanRepo, personalLoanRepo: personalLoanRepo,
		fdRepo: fdRepo, bondRepo: bondRepo, sipRepo: sipRepo, creditCardRepo: creditCardRepo,
	}
}

func (s *NotificationService) GetAll(ctx context.Context) ([]models.Notification, error) {
	return s.repo.GetAll(ctx)
}

func (s *NotificationService) GetUnread(ctx context.Context) ([]models.Notification, error) {
	return s.repo.GetUnread(ctx)
}

func (s *NotificationService) MarkRead(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, objID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context) error {
	return s.repo.MarkAllRead(ctx)
}

// GenerateNotifications scans all modules and creates upcoming reminders
func (s *NotificationService) GenerateNotifications(ctx context.Context) ([]models.Notification, error) {
	now := time.Now()
	upcoming := now.AddDate(0, 0, 7) // 7 days ahead
	var generated []models.Notification

	// Home Loan EMIs
	homeLoans, _ := s.homeLoanRepo.GetAll(ctx)
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
	pLoans, _ := s.personalLoanRepo.GetAll(ctx)
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
	fds, _ := s.fdRepo.GetAll(ctx)
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
	bonds, _ := s.bondRepo.GetAll(ctx)
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
	sips, _ := s.sipRepo.GetAll(ctx)
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
	cards, _ := s.creditCardRepo.GetAll(ctx)
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

	// Store generated notifications
	for i := range generated {
		generated[i].ID = [12]byte{} // reset for insert
		n := generated[i]
		n.CreatedAt = time.Now()
		uid := uuid.New().String()
		_ = uid
		_ = s.repo.Create(ctx, &n)
	}

	return generated, nil
}
