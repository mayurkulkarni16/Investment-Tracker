package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BenchmarkService struct{}

func NewBenchmarkService() *BenchmarkService {
	return &BenchmarkService{}
}

type BenchmarkPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type BenchmarkData struct {
	IndexName string           `json:"index_name"`
	Symbol    string           `json:"symbol"`
	Points    []BenchmarkPoint `json:"points"`
	Current   float64          `json:"current"`
	Change1Y  float64          `json:"change_1y"` // %
	Change3Y  float64          `json:"change_3y"` // %
	Change5Y  float64          `json:"change_5y"` // %
}

type BenchmarkResponse struct {
	Indices []BenchmarkData `json:"indices"`
}

func (s *BenchmarkService) GetBenchmarks(ctx context.Context, period string) (*BenchmarkResponse, error) {
	periodMap := map[string]string{
		"1m": "1mo", "3m": "3mo", "6m": "6mo",
		"1y": "1y", "3y": "3y", "5y": "5y", "10y": "10y",
	}

	yahooRange, ok := periodMap[period]
	if !ok {
		yahooRange = "1y"
	}

	indices := []struct {
		symbol string
		name   string
	}{
		{"^NSEI", "Nifty 50"},
		{"^BSESN", "Sensex"},
	}

	resp := &BenchmarkResponse{Indices: []BenchmarkData{}}

	for _, idx := range indices {
		data, err := s.fetchYahooChart(idx.symbol, yahooRange)
		if err != nil {
			continue
		}
		data.IndexName = idx.name
		data.Symbol = idx.symbol
		resp.Indices = append(resp.Indices, *data)
	}

	return resp, nil
}

func (s *BenchmarkService) fetchYahooChart(symbol string, yahooRange string) (*BenchmarkData, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=1d", symbol, yahooRange)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data for %s", symbol)
	}

	r := result.Chart.Result[0]
	data := &BenchmarkData{Points: []BenchmarkPoint{}}

	closes := r.Indicators.Quote[0].Close
	for i, ts := range r.Timestamp {
		if i < len(closes) && closes[i] > 0 {
			t := time.Unix(ts, 0)
			data.Points = append(data.Points, BenchmarkPoint{
				Date:  t.Format("2006-01-02"),
				Value: closes[i],
			})
		}
	}

	if len(data.Points) > 0 {
		data.Current = data.Points[len(data.Points)-1].Value
		first := data.Points[0].Value
		if first > 0 {
			data.Change1Y = (data.Current - first) / first * 100
		}
	}

	return data, nil
}
