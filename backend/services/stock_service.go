package services

import (
	"context"
	"fmt"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"github.com/google/uuid"
)

type StockService struct {
	repo         *repository.StockRepo
	priceFetcher *StockPriceFetcher
}

func NewStockService(repo *repository.StockRepo, priceFetcher *StockPriceFetcher) *StockService {
	return &StockService{repo: repo, priceFetcher: priceFetcher}
}

func (s *StockService) Create(ctx context.Context, userID string, req models.CreateStockRequest) (*models.Stock, error) {
	stock := &models.Stock{
		UserID:       userID,
		StockName:    req.StockName,
		Symbol:       req.Symbol,
		Exchange:     req.Exchange,
		Transactions: []models.StockTransaction{},
		Notes:        req.Notes,
	}

	if err := s.repo.Create(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) GetAll(ctx context.Context, userID string) ([]models.Stock, error) {
	stocks, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	marketOpen := IsMarketOpen()
	for i := range stocks {
		s.recalculate(&stocks[i])
		stocks[i].IsMarketOpen = marketOpen
		stocks[i].XIRR = ComputeStockXIRR(&stocks[i])
		stocks[i].DataSource = "Yahoo Finance"
	}
	return stocks, nil
}

func (s *StockService) GetByID(ctx context.Context, id string) (*models.Stock, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	s.recalculate(stock)
	stock.IsMarketOpen = IsMarketOpen()
	stock.XIRR = ComputeStockXIRR(stock)
	return stock, nil
}

func (s *StockService) AddTransaction(ctx context.Context, id string, req models.AddStockTransactionRequest) (*models.Stock, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	txDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}

	txn := models.StockTransaction{
		TransactionID: uuid.New().String(),
		Date:          txDate,
		Type:          req.Type,
		Quantity:      req.Quantity,
		PricePerShare: req.PricePerShare,
		Amount:        float64(req.Quantity) * req.PricePerShare,
	}

	stock.Transactions = append(stock.Transactions, txn)
	s.recalculate(stock)

	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) RefreshPrice(ctx context.Context, id string) (*models.Stock, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	quote, err := s.priceFetcher.FetchQuote(stock.Symbol, string(stock.Exchange))
	if err != nil {
		return nil, err
	}

	stock.CurrentPrice = quote.Price
	stock.DayChange = quote.DayChange
	stock.DayChangePct = quote.DayChangePct
	now := time.Now()
	stock.PriceLastUpdated = &now
	s.recalculate(stock)
	stock.IsMarketOpen = IsMarketOpen()

	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) RefreshAllPrices(ctx context.Context) ([]models.Stock, error) {
	stocks, err := s.repo.GetAll(ctx, "")
	if err != nil {
		return nil, err
	}

	for i := range stocks {
		quote, err := s.priceFetcher.FetchQuote(stocks[i].Symbol, string(stocks[i].Exchange))
		if err != nil {
			continue
		}
		stocks[i].CurrentPrice = quote.Price
		stocks[i].DayChange = quote.DayChange
		stocks[i].DayChangePct = quote.DayChangePct
		now := time.Now()
		stocks[i].PriceLastUpdated = &now
		s.recalculate(&stocks[i])
		_ = s.repo.Update(ctx, &stocks[i])
	}

	marketOpen := IsMarketOpen()
	for i := range stocks {
		stocks[i].IsMarketOpen = marketOpen
	}
	return stocks, nil
}

func (s *StockService) Update(ctx context.Context, id string, req models.UpdateStockRequest) (*models.Stock, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}

	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.StockName != "" {
		stock.StockName = req.StockName
	}
	if req.Symbol != "" {
		stock.Symbol = req.Symbol
	}
	if req.Exchange != "" {
		stock.Exchange = req.Exchange
	}
	if req.Notes != "" {
		stock.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *StockService) DeleteTransaction(ctx context.Context, stockID, txnID string) (*models.Stock, error) {
	objID, err := parseObjectID(stockID)
	if err != nil {
		return nil, err
	}
	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	found := false
	newTxns := make([]models.StockTransaction, 0, len(stock.Transactions))
	for _, t := range stock.Transactions {
		if t.TransactionID == txnID {
			found = true
			continue
		}
		newTxns = append(newTxns, t)
	}
	if !found {
		return nil, fmt.Errorf("transaction not found")
	}
	stock.Transactions = newTxns
	s.recalculate(stock)
	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) UpdateTransaction(ctx context.Context, stockID, txnID string, req models.AddStockTransactionRequest) (*models.Stock, error) {
	objID, err := parseObjectID(stockID)
	if err != nil {
		return nil, err
	}
	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	txDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	found := false
	for i, t := range stock.Transactions {
		if t.TransactionID == txnID {
			stock.Transactions[i].Date = txDate
			stock.Transactions[i].Type = req.Type
			stock.Transactions[i].Quantity = req.Quantity
			stock.Transactions[i].PricePerShare = req.PricePerShare
			stock.Transactions[i].Amount = float64(req.Quantity) * req.PricePerShare
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("transaction not found")
	}
	s.recalculate(stock)
	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) AddDividend(ctx context.Context, stockID string, req models.AddStockDividendRequest) (*models.Stock, error) {
	objID, err := parseObjectID(stockID)
	if err != nil {
		return nil, err
	}
	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	dvDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	// Calculate quantity held at dividend date
	qtyAtDate := 0
	for _, tx := range stock.Transactions {
		if !tx.Date.After(dvDate) {
			if tx.Type == models.StockBuy {
				qtyAtDate += tx.Quantity
			} else {
				qtyAtDate -= tx.Quantity
			}
		}
	}
	div := models.StockDividend{
		DividendID:     uuid.New().String(),
		Date:           dvDate,
		AmountPerShare: req.AmountPerShare,
		TotalAmount:    req.AmountPerShare * float64(qtyAtDate),
	}
	stock.Dividends = append(stock.Dividends, div)
	s.recalculate(stock)
	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

func (s *StockService) DeleteDividend(ctx context.Context, stockID, divID string) (*models.Stock, error) {
	objID, err := parseObjectID(stockID)
	if err != nil {
		return nil, err
	}
	stock, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	found := false
	newDivs := make([]models.StockDividend, 0, len(stock.Dividends))
	for _, d := range stock.Dividends {
		if d.DividendID == divID {
			found = true
			continue
		}
		newDivs = append(newDivs, d)
	}
	if !found {
		return nil, fmt.Errorf("dividend not found")
	}
	stock.Dividends = newDivs
	s.recalculate(stock)
	if err := s.repo.Update(ctx, stock); err != nil {
		return nil, err
	}
	return stock, nil
}

// recalculate recomputes derived fields from transactions
func (s *StockService) recalculate(stock *models.Stock) {
	totalQty := 0
	totalInvested := 0.0

	for _, txn := range stock.Transactions {
		if txn.Type == models.StockBuy {
			totalQty += txn.Quantity
			totalInvested += txn.Amount
		} else {
			// For sells, deduct using average cost
			avgCost := 0.0
			if totalQty > 0 {
				avgCost = totalInvested / float64(totalQty)
			}
			totalQty -= txn.Quantity
			totalInvested -= float64(txn.Quantity) * avgCost
		}
	}

	if totalQty < 0 {
		totalQty = 0
	}
	if totalInvested < 0 {
		totalInvested = 0
	}

	stock.TotalQuantity = totalQty
	stock.TotalInvested = totalInvested

	if totalQty > 0 {
		stock.AvgBuyPrice = totalInvested / float64(totalQty)
	} else {
		stock.AvgBuyPrice = 0
	}

	stock.CurrentValue = float64(totalQty) * stock.CurrentPrice
	stock.GainLoss = stock.CurrentValue - stock.TotalInvested
	if stock.TotalInvested > 0 {
		stock.GainLossPercent = (stock.GainLoss / stock.TotalInvested) * 100
	} else {
		stock.GainLossPercent = 0
	}

	// Sum dividends
	totalDiv := 0.0
	for _, d := range stock.Dividends {
		totalDiv += d.TotalAmount
	}
	stock.TotalDividends = totalDiv
}
