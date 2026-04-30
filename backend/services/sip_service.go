package services

import (
	"context"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type SIPService struct {
	repo   *repository.SIPRepo
	mfRepo *repository.MutualFundRepo
}

func NewSIPService(repo *repository.SIPRepo, mfRepo *repository.MutualFundRepo) *SIPService {
	return &SIPService{repo: repo, mfRepo: mfRepo}
}

func (s *SIPService) Create(ctx context.Context, userID string, req models.CreateSIPRequest) (*models.SIP, error) {
	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}

	var endDate *time.Time
	if req.EndDate != "" {
		ed, err := parseDate(req.EndDate)
		if err != nil {
			return nil, err
		}
		endDate = &ed
	}

	sip := &models.SIP{
		UserID:       userID,
		FundName:     req.FundName,
		FundID:       req.FundID,
		AMCCode:      req.AMCCode,
		SchemeCode:   req.SchemeCode,
		Amount:       req.Amount,
		Frequency:    req.Frequency,
		SIPDate:      req.SIPDate,
		StartDate:    startDate,
		EndDate:      endDate,
		Installments: []models.SIPInstallment{},
		Status:       "active",
		Notes:        req.Notes,
	}

	if err := s.repo.Create(ctx, sip); err != nil {
		return nil, err
	}
	s.computeDerived(sip)
	return sip, nil
}

func (s *SIPService) GetAll(ctx context.Context, userID string) ([]models.SIP, error) {
	sips, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range sips {
		s.computeDerived(&sips[i])
	}
	return sips, nil
}

func (s *SIPService) GetByID(ctx context.Context, id string) (*models.SIP, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	sip, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.computeDerived(sip)
	return sip, nil
}

func (s *SIPService) Update(ctx context.Context, id string, req models.UpdateSIPRequest) (*models.SIP, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	sip, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.Amount > 0 {
		sip.Amount = req.Amount
	}
	if req.SIPDate > 0 {
		sip.SIPDate = req.SIPDate
	}
	if req.EndDate != "" {
		ed, err := parseDate(req.EndDate)
		if err != nil {
			return nil, err
		}
		sip.EndDate = &ed
	}
	if req.Status != "" {
		sip.Status = req.Status
	}
	if req.Notes != "" {
		sip.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, sip); err != nil {
		return nil, err
	}
	s.computeDerived(sip)
	return sip, nil
}

func (s *SIPService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *SIPService) RecordInstallment(ctx context.Context, id string, req models.RecordSIPInstallmentRequest) (*models.SIP, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	sip, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	installment := models.SIPInstallment{
		InstallmentID: uuid.New().String(),
		Date:          date,
		Amount:        req.Amount,
		NAV:           req.NAV,
		Units:         req.Units,
		Status:        req.Status,
	}

	sip.Installments = append(sip.Installments, installment)
	sip.TotalInstallments++

	switch req.Status {
	case "success":
		sip.CompletedInstallments++
		sip.TotalInvested += req.Amount

		// Also create a transaction in the linked mutual fund
		if sip.FundID != "" {
			fundObjID, err := parseObjectID(sip.FundID)
			if err == nil {
				mf, err := s.mfRepo.GetByID(ctx, fundObjID)
				if err == nil {
					units := req.Units
					if units == 0 && req.NAV > 0 {
						units = req.Amount / req.NAV
					}

					txn := models.MFTransaction{
						TransactionID: uuid.New().String(),
						Date:          date,
						Type:          models.TransactionSIP,
						Amount:        req.Amount,
						NAVAtPurchase: req.NAV,
						Units:         units,
					}

					if mf.IsELSS {
						lockIn := date.AddDate(3, 0, 0)
						txn.LockInEnd = &lockIn
					}

					mf.Transactions = append(mf.Transactions, txn)
					mf.TotalUnits += units
					mf.TotalInvested += req.Amount
					mf.CurrentValue = mf.TotalUnits * mf.CurrentNAV
					mf.GainLoss = mf.CurrentValue - mf.TotalInvested
					if mf.TotalInvested > 0 {
						mf.GainLossPercent = (mf.GainLoss / mf.TotalInvested) * 100
					}
					_ = s.mfRepo.Update(ctx, mf)
				}
			}
		}
	case "failed", "skipped":
		sip.MissedInstallments++
	}

	if err := s.repo.Update(ctx, sip); err != nil {
		return nil, err
	}
	s.computeDerived(sip)
	return sip, nil
}

func (s *SIPService) computeDerived(sip *models.SIP) {
	now := time.Now()
	monthsSinceStart := int(now.Sub(sip.StartDate).Hours() / 24 / 30)
	if monthsSinceStart < 0 {
		monthsSinceStart = 0
	}
	sip.MonthsActive = monthsSinceStart

	// Next SIP date
	next := time.Date(now.Year(), now.Month(), sip.SIPDate, 0, 0, 0, 0, time.UTC)
	if next.Before(now) {
		next = next.AddDate(0, 1, 0)
	}
	sip.NextSIPDate = next
}
