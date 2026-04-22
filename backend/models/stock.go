package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StockTransactionType string

const (
	StockBuy  StockTransactionType = "buy"
	StockSell StockTransactionType = "sell"
)

type StockExchange string

const (
	ExchangeNSE StockExchange = "NSE"
	ExchangeBSE StockExchange = "BSE"
)

type StockTransaction struct {
	TransactionID string               `json:"transaction_id" bson:"transaction_id"`
	Date          time.Time            `json:"date" bson:"date"`
	Type          StockTransactionType `json:"type" bson:"type"`
	Quantity      int                  `json:"quantity" bson:"quantity"`
	PricePerShare float64              `json:"price_per_share" bson:"price_per_share"`
	Amount        float64              `json:"amount" bson:"amount"`
}

type Stock struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	StockName        string             `json:"stock_name" bson:"stock_name"`
	Symbol           string             `json:"symbol" bson:"symbol"`
	Exchange         StockExchange      `json:"exchange" bson:"exchange"`
	Transactions     []StockTransaction `json:"transactions" bson:"transactions"`
	TotalQuantity    int                `json:"total_quantity" bson:"total_quantity"`
	TotalInvested    float64            `json:"total_invested" bson:"total_invested"`
	AvgBuyPrice      float64            `json:"avg_buy_price" bson:"avg_buy_price"`
	CurrentPrice     float64            `json:"current_price" bson:"current_price"`
	CurrentValue     float64            `json:"current_value" bson:"current_value"`
	DayChange        float64            `json:"day_change" bson:"day_change"`
	DayChangePct     float64            `json:"day_change_percent" bson:"day_change_percent"`
	GainLoss         float64            `json:"gain_loss" bson:"gain_loss"`
	GainLossPercent  float64            `json:"gain_loss_percent" bson:"gain_loss_percent"`
	PriceLastUpdated *time.Time         `json:"price_last_updated,omitempty" bson:"price_last_updated,omitempty"`
	IsMarketOpen     bool               `json:"is_market_open" bson:"-"`
	Notes            string             `json:"notes" bson:"notes"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreateStockRequest struct {
	StockName string        `json:"stock_name" binding:"required"`
	Symbol    string        `json:"symbol" binding:"required"`
	Exchange  StockExchange `json:"exchange" binding:"required"`
	Notes     string        `json:"notes"`
}

type UpdateStockRequest struct {
	StockName string        `json:"stock_name"`
	Symbol    string        `json:"symbol"`
	Exchange  StockExchange `json:"exchange"`
	Notes     string        `json:"notes"`
}

type AddStockTransactionRequest struct {
	Date          string               `json:"date" binding:"required"`
	Type          StockTransactionType `json:"type" binding:"required"`
	Quantity      int                  `json:"quantity" binding:"required"`
	PricePerShare float64              `json:"price_per_share" binding:"required"`
}
