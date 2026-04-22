package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type StockQuote struct {
	Price        float64
	PrevClose    float64
	DayChange    float64
	DayChangePct float64
}

type StockPriceFetcher struct {
	client *http.Client
}

func NewStockPriceFetcher() *StockPriceFetcher {
	return &StockPriceFetcher{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// yahooSymbol converts our symbol+exchange to a Yahoo Finance ticker
func yahooSymbol(symbol string, exchange string) string {
	switch exchange {
	case "NSE":
		return symbol + ".NS"
	case "BSE":
		return symbol + ".BO"
	default:
		return symbol + ".NS"
	}
}

func (f *StockPriceFetcher) FetchQuote(symbol string, exchange string) (*StockQuote, error) {
	ySym := yahooSymbol(symbol, exchange)
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", url.PathEscape(ySym))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stock price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stock API returned status %d", resp.StatusCode)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice json.Number `json:"regularMarketPrice"`
					ChartPreviousClose json.Number `json:"chartPreviousClose"`
					PreviousClose      json.Number `json:"previousClose"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode stock response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("stock API error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data found for symbol %s", symbol)
	}

	meta := result.Chart.Result[0].Meta
	price, err := strconv.ParseFloat(meta.RegularMarketPrice.String(), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	prevCloseStr := meta.ChartPreviousClose.String()
	if prevCloseStr == "" {
		prevCloseStr = meta.PreviousClose.String()
	}
	prevClose, _ := strconv.ParseFloat(prevCloseStr, 64)

	dayChange := price - prevClose
	dayChangePct := 0.0
	if prevClose > 0 {
		dayChangePct = (dayChange / prevClose) * 100
	}

	return &StockQuote{
		Price:        price,
		PrevClose:    prevClose,
		DayChange:    dayChange,
		DayChangePct: dayChangePct,
	}, nil
}

// IsMarketOpen checks if the Indian stock market (NSE/BSE) is currently open.
// Market hours: Mon-Fri, 9:15 AM to 3:30 PM IST.
func IsMarketOpen() bool {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return false
	}
	now := time.Now().In(ist)

	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour, min, _ := now.Clock()
	minutesSinceMidnight := hour*60 + min

	marketOpen := 9*60 + 15   // 9:15 AM
	marketClose := 15*60 + 30 // 3:30 PM

	return minutesSinceMidnight >= marketOpen && minutesSinceMidnight <= marketClose
}
