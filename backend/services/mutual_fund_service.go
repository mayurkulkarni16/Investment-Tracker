package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type MutualFundService struct {
	repo       *repository.MutualFundRepo
	navFetcher *NAVFetcher
}

func NewMutualFundService(repo *repository.MutualFundRepo, navFetcher *NAVFetcher) *MutualFundService {
	return &MutualFundService{repo: repo, navFetcher: navFetcher}
}

func (s *MutualFundService) Create(ctx context.Context, userID string, req models.CreateMutualFundRequest) (*models.MutualFund, error) {
	mf := &models.MutualFund{
		UserID:       userID,
		FundName:     req.FundName,
		AMC:          req.AMC,
		FundType:     req.FundType,
		SchemeCode:   req.SchemeCode,
		FolioNumber:  req.FolioNumber,
		IsELSS:       req.IsELSS,
		Transactions: []models.MFTransaction{},
		Notes:        req.Notes,
	}

	if err := s.repo.Create(ctx, mf); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *MutualFundService) GetAll(ctx context.Context, userID string) ([]models.MutualFund, error) {
	funds, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range funds {
		s.recalculate(&funds[i])
		funds[i].XIRR = ComputeMutualFundXIRR(&funds[i])
		funds[i].DataSource = "mfapi.in"
	}
	return funds, nil
}

func (s *MutualFundService) GetByID(ctx context.Context, id string) (*models.MutualFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	mf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.recalculate(mf)
	mf.XIRR = ComputeMutualFundXIRR(mf)
	return mf, nil
}

func (s *MutualFundService) RecalculateAll(ctx context.Context, userID string) (int, error) {
	funds, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return 0, err
	}
	for i := range funds {
		s.recalculate(&funds[i])
		_ = s.repo.Update(ctx, &funds[i])
	}
	return len(funds), nil
}

func (s *MutualFundService) AddTransaction(ctx context.Context, id string, req models.AddMFTransactionRequest) (*models.MutualFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	mf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	txDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	units := req.Units
	if units == 0 && req.NAVAtPurchase > 0 {
		units = req.Amount / req.NAVAtPurchase
	}

	txn := models.MFTransaction{
		TransactionID: uuid.New().String(),
		Date:          txDate,
		Type:          req.Type,
		Amount:        req.Amount,
		NAVAtPurchase: req.NAVAtPurchase,
		Units:         units,
	}

	if mf.IsELSS && (req.Type == models.TransactionPurchase || req.Type == models.TransactionSIP) {
		lockIn := txDate.AddDate(3, 0, 0)
		txn.LockInEnd = &lockIn
	}

	mf.Transactions = append(mf.Transactions, txn)

	if req.Type == models.TransactionRedemption || req.Type == models.TransactionSwitchOut {
		// Deduct cost basis using average cost per unit, not redemption amount
		avgCost := 0.0
		if mf.TotalUnits > 0 {
			avgCost = mf.TotalInvested / mf.TotalUnits
		}
		mf.TotalUnits -= units
		mf.TotalInvested -= units * avgCost
	} else {
		mf.TotalUnits += units
		mf.TotalInvested += req.Amount
	}

	s.recalculate(mf)

	if err := s.repo.Update(ctx, mf); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *MutualFundService) RefreshNAV(ctx context.Context, id string) (*models.MutualFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	mf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	nav, err := s.navFetcher.FetchLatestNAV(mf.SchemeCode)
	if err != nil {
		return nil, err
	}

	mf.CurrentNAV = nav
	now := time.Now()
	mf.NAVLastUpdated = &now
	s.recalculate(mf)

	if err := s.repo.Update(ctx, mf); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *MutualFundService) RefreshAllNAVs(ctx context.Context) ([]models.MutualFund, error) {
	funds, err := s.repo.GetAll(ctx, "")
	if err != nil {
		return nil, err
	}

	for i := range funds {
		nav, err := s.navFetcher.FetchLatestNAV(funds[i].SchemeCode)
		if err != nil {
			continue
		}
		funds[i].CurrentNAV = nav
		now := time.Now()
		funds[i].NAVLastUpdated = &now
		s.recalculate(&funds[i])
		_ = s.repo.Update(ctx, &funds[i])
	}
	return funds, nil
}

func (s *MutualFundService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *MutualFundService) DeleteTransaction(ctx context.Context, fundID, txnID string) (*models.MutualFund, error) {
	objID, err := parseObjectID(fundID)
	if err != nil {
		return nil, err
	}
	mf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	found := false
	newTxns := make([]models.MFTransaction, 0, len(mf.Transactions))
	for _, t := range mf.Transactions {
		if t.TransactionID == txnID {
			found = true
			continue
		}
		newTxns = append(newTxns, t)
	}
	if !found {
		return nil, fmt.Errorf("transaction not found")
	}
	mf.Transactions = newTxns
	s.recalculate(mf)
	if err := s.repo.Update(ctx, mf); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *MutualFundService) UpdateTransaction(ctx context.Context, fundID, txnID string, req models.AddMFTransactionRequest) (*models.MutualFund, error) {
	objID, err := parseObjectID(fundID)
	if err != nil {
		return nil, err
	}
	mf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	txDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	units := req.Units
	if units == 0 && req.NAVAtPurchase > 0 {
		units = req.Amount / req.NAVAtPurchase
	}
	found := false
	for i, t := range mf.Transactions {
		if t.TransactionID == txnID {
			mf.Transactions[i].Date = txDate
			mf.Transactions[i].Type = req.Type
			mf.Transactions[i].Amount = req.Amount
			mf.Transactions[i].NAVAtPurchase = req.NAVAtPurchase
			mf.Transactions[i].Units = units
			if mf.IsELSS && (req.Type == models.TransactionPurchase || req.Type == models.TransactionSIP) {
				lockIn := txDate.AddDate(3, 0, 0)
				mf.Transactions[i].LockInEnd = &lockIn
			} else {
				mf.Transactions[i].LockInEnd = nil
			}
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("transaction not found")
	}
	s.recalculate(mf)
	if err := s.repo.Update(ctx, mf); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *MutualFundService) Update(ctx context.Context, id string, req models.UpdateMutualFundRequest) (*models.MutualFund, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	mf, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.FundName != "" {
		mf.FundName = req.FundName
	}
	if req.AMC != "" {
		mf.AMC = req.AMC
	}
	if req.FundType != "" {
		mf.FundType = req.FundType
	}
	if req.SchemeCode != "" {
		mf.SchemeCode = req.SchemeCode
	}
	if req.FolioNumber != "" {
		mf.FolioNumber = req.FolioNumber
	}
	mf.IsELSS = req.IsELSS
	if req.Notes != "" {
		mf.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, mf); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *MutualFundService) ImportFromCAS(ctx context.Context, userID string, req models.ImportCASRequest) (*models.ImportResult, error) {
	result := &models.ImportResult{}

	for _, fund := range req.Funds {
		// Sort transactions chronologically for correct average cost calculation
		sort.Slice(fund.Transactions, func(i, j int) bool {
			return fund.Transactions[i].Date < fund.Transactions[j].Date
		})

		// Try to find existing fund by folio number
		existingMF, err := s.repo.GetByFolioNumber(ctx, userID, fund.FolioNumber)

		if err != nil {
			// Fund doesn't exist, create it
			fundType := mapCASCategory(fund.Category)
			isELSS := fundType == models.FundTypeELSS
			amc := extractAMC(fund.FundName)

			newMF := &models.MutualFund{
				UserID:       userID,
				FundName:     fund.FundName,
				AMC:          amc,
				FundType:     fundType,
				FolioNumber:  fund.FolioNumber,
				IsELSS:       isELSS,
				Transactions: []models.MFTransaction{},
			}

			for _, tx := range fund.Transactions {
				txDate, parseErr := parseDate(tx.Date)
				if parseErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Invalid date %s for %s: %v", tx.Date, fund.FundName, parseErr))
					continue
				}

				txType := mapCASTransactionType(tx.Type)
				txn := models.MFTransaction{
					TransactionID: uuid.New().String(),
					Date:          txDate,
					Type:          txType,
					Amount:        tx.Amount,
					NAVAtPurchase: tx.NAV,
					Units:         tx.Units,
				}

				if isELSS && (txType == models.TransactionPurchase || txType == models.TransactionSIP) {
					lockIn := txDate.AddDate(3, 0, 0)
					txn.LockInEnd = &lockIn
				}

				newMF.Transactions = append(newMF.Transactions, txn)

				if txType == models.TransactionRedemption || txType == models.TransactionSwitchOut {
					// Deduct cost basis using average cost per unit
					avgCost := 0.0
					if newMF.TotalUnits > 0 {
						avgCost = newMF.TotalInvested / newMF.TotalUnits
					}
					newMF.TotalUnits -= tx.Units
					newMF.TotalInvested -= tx.Units * avgCost
				} else {
					newMF.TotalUnits += tx.Units
					newMF.TotalInvested += tx.Amount
				}
				result.TransactionsAdded++
			}

			s.recalculate(newMF)
			if err := s.repo.Create(ctx, newMF); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create fund %s: %v", fund.FundName, err))
				continue
			}
			result.FundsCreated++
		} else {
			// Fund exists, add new transactions (avoid duplicates)
			added := 0
			for _, tx := range fund.Transactions {
				txDate, parseErr := parseDate(tx.Date)
				if parseErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Invalid date %s: %v", tx.Date, parseErr))
					continue
				}

				// Check for duplicate (4-field composite: date + amount + units + type)
				isDuplicate := false
				txType := mapCASTransactionType(tx.Type)
				for _, existingTx := range existingMF.Transactions {
					if existingTx.Date.Equal(txDate) &&
						existingTx.Amount == tx.Amount &&
						existingTx.Units == tx.Units &&
						existingTx.Type == txType {
						isDuplicate = true
						break
					}
				}
				if isDuplicate {
					result.TransactionsSkipped++
					continue
				}

				txn := models.MFTransaction{
					TransactionID: uuid.New().String(),
					Date:          txDate,
					Type:          txType,
					Amount:        tx.Amount,
					NAVAtPurchase: tx.NAV,
					Units:         tx.Units,
				}

				if existingMF.IsELSS && (txType == models.TransactionPurchase || txType == models.TransactionSIP) {
					lockIn := txDate.AddDate(3, 0, 0)
					txn.LockInEnd = &lockIn
				}

				existingMF.Transactions = append(existingMF.Transactions, txn)

				if txType == models.TransactionRedemption || txType == models.TransactionSwitchOut {
					// Deduct cost basis using average cost per unit
					avgCost := 0.0
					if existingMF.TotalUnits > 0 {
						avgCost = existingMF.TotalInvested / existingMF.TotalUnits
					}
					existingMF.TotalUnits -= tx.Units
					existingMF.TotalInvested -= tx.Units * avgCost
				} else {
					existingMF.TotalUnits += tx.Units
					existingMF.TotalInvested += tx.Amount
				}
				added++
				result.TransactionsAdded++
			}

			if added > 0 {
				s.recalculate(existingMF)
				if err := s.repo.Update(ctx, existingMF); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to update fund %s: %v", fund.FundName, err))
					continue
				}
				result.FundsUpdated++
			}
		}
	}

	return result, nil
}

func mapCASCategory(category string) models.FundType {
	switch {
	// Tax-saving
	case contains(category, "ELSS") || contains(category, "Tax Saver") || contains(category, "Tax Saving"):
		return models.FundTypeELSS
	// Index / ETF
	case contains(category, "Index") || contains(category, "Nifty") || contains(category, "Sensex") || contains(category, "ETF"):
		return models.FundTypeIndex
	// Overnight
	case contains(category, "Overnight"):
		return models.FundTypeOvernight
	// Liquid
	case contains(category, "Liquid"):
		return models.FundTypeLiquid
	// Money Market
	case contains(category, "Money Market"):
		return models.FundTypeMoneyMarket
	// Gilt
	case contains(category, "Gilt") || contains(category, "Government Securities"):
		return models.FundTypeGilt
	// Corporate Bond (debt)
	case contains(category, "Corporate Bond") || contains(category, "Credit Risk"):
		return models.FundTypeCorporateBond
	// Dynamic Bond
	case contains(category, "Dynamic Bond") || contains(category, "Dynamic Duration"):
		return models.FundTypeDynamicBond
	// Small Cap
	case contains(category, "Small Cap") || contains(category, "Smallcap"):
		return models.FundTypeSmallCap
	// Mid Cap
	case contains(category, "Mid Cap") || contains(category, "Midcap"):
		return models.FundTypeMidCap
	// Large Cap
	case contains(category, "Large Cap") || contains(category, "Largecap") || contains(category, "Bluechip"):
		return models.FundTypeLargeCap
	// Large & Mid Cap
	case contains(category, "Large & Mid"):
		return models.FundTypeLargeCap
	// Multi Cap
	case contains(category, "Multi Cap") || contains(category, "Multicap"):
		return models.FundTypeMultiCap
	// Flexi Cap
	case contains(category, "Flexi Cap") || contains(category, "Flexicap"):
		return models.FundTypeFlexiCap
	// Sectoral / Thematic
	case contains(category, "Sectoral") || contains(category, "Sector"):
		return models.FundTypeSectoral
	case contains(category, "Thematic"):
		return models.FundTypeThematic
	// Hybrid
	case contains(category, "Hybrid") || contains(category, "Balanced") || contains(category, "Aggressive") || contains(category, "Conservative") || contains(category, "Equity Savings") || contains(category, "Arbitrage"):
		return models.FundTypeHybrid
	// Debt (catch-all for debt variants)
	case contains(category, "Debt") || contains(category, "Short Duration") || contains(category, "Medium Duration") || contains(category, "Long Duration") || contains(category, "Ultra Short") || contains(category, "Low Duration") || contains(category, "Banking") || contains(category, "Floater"):
		return models.FundTypeDebt
	default:
		return models.FundTypeEquity
	}
}

func mapCASTransactionType(t string) models.TransactionType {
	switch {
	case contains(t, "SIP"):
		return models.TransactionSIP
	case contains(t, "Withdraw") || contains(t, "Redemption"):
		return models.TransactionRedemption
	case contains(t, "Switch In"):
		return models.TransactionSwitchIn
	case contains(t, "Switch Out"):
		return models.TransactionSwitchOut
	default:
		return models.TransactionPurchase
	}
}

func extractAMC(fundName string) string {
	amcMap := map[string]string{
		"Nippon India":     "Nippon India",
		"UTI":              "UTI",
		"ICICI Prudential": "ICICI Prudential",
		"Quant":            "Quant",
		"Axis":             "Axis",
		"Canara Robeco":    "Canara Robeco",
		"PGIM India":       "PGIM India",
		"HDFC":             "HDFC",
		"SBI":              "SBI",
		"Kotak":            "Kotak",
		"Aditya Birla":     "Aditya Birla Sun Life",
		"Mirae Asset":      "Mirae Asset",
		"Tata":             "Tata",
		"DSP":              "DSP",
		"Motilal Oswal":    "Motilal Oswal",
		"Parag Parikh":     "PPFAS",
		"Franklin":         "Franklin Templeton",
		"Bandhan":          "Bandhan",
		"Edelweiss":        "Edelweiss",
	}
	for prefix, amc := range amcMap {
		if contains(fundName, prefix) {
			return amc
		}
	}
	// Fallback: use first two words
	parts := strings.Fields(fundName)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return fundName
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func (s *MutualFundService) recalculate(mf *models.MutualFund) {
	// Recompute TotalUnits and TotalInvested from all transactions
	// using average cost method for redemptions
	sort.Slice(mf.Transactions, func(i, j int) bool {
		return mf.Transactions[i].Date.Before(mf.Transactions[j].Date)
	})

	mf.TotalUnits = 0
	mf.TotalInvested = 0

	for _, tx := range mf.Transactions {
		if tx.Type == models.TransactionRedemption || tx.Type == models.TransactionSwitchOut {
			avgCost := 0.0
			if mf.TotalUnits > 0 {
				avgCost = mf.TotalInvested / mf.TotalUnits
			}
			mf.TotalUnits -= tx.Units
			mf.TotalInvested -= tx.Units * avgCost
		} else {
			mf.TotalUnits += tx.Units
			mf.TotalInvested += tx.Amount
		}
	}

	// Prevent floating point drift below zero
	if mf.TotalInvested < 0 {
		mf.TotalInvested = 0
	}
	if mf.TotalUnits < 0 {
		mf.TotalUnits = 0
	}

	if mf.CurrentNAV > 0 {
		mf.CurrentValue = mf.TotalUnits * mf.CurrentNAV
		mf.GainLoss = mf.CurrentValue - mf.TotalInvested
		if mf.TotalInvested > 0 {
			mf.GainLossPercent = (mf.GainLoss / mf.TotalInvested) * 100
		}
	}
}
