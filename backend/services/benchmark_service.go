package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

const benchmarkCacheTTL = time.Hour

type BenchmarkService struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]benchmarkCacheEntry
}

type benchmarkCacheEntry struct {
	points  []BenchmarkPoint
	fetched time.Time
}

func NewBenchmarkService() *BenchmarkService {
	return &BenchmarkService{
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  map[string]benchmarkCacheEntry{},
	}
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
	Change1Y  float64          `json:"change_1y"` // % absolute
	Change3Y  float64          `json:"change_3y"` // % absolute
	Change5Y  float64          `json:"change_5y"` // % absolute
	CAGR3Y    float64          `json:"cagr_3y"`   // % per year
	CAGR5Y    float64          `json:"cagr_5y"`   // % per year
	AsOf      string           `json:"as_of"`     // date of the latest close
	Available []string         `json:"available"` // which of 1y/3y/5y have enough history
}

type BenchmarkResponse struct {
	Indices []BenchmarkData `json:"indices"`
}

var benchmarkIndices = []struct{ symbol, name string }{
	{"^NSEI", "Nifty 50"},
	{"^BSESN", "Sensex"},
}

// periodMonths maps the supported chart periods to months of history.
var periodMonths = map[string]int{"1m": 1, "3m": 3, "6m": 6, "1y": 12, "3y": 36, "5y": 60}

// GetBenchmarks returns index levels for the requested chart period along with
// 1Y/3Y/5Y returns, which are always computed from 5 years of daily closes.
func (s *BenchmarkService) GetBenchmarks(ctx context.Context, period string) (*BenchmarkResponse, error) {
	months, ok := periodMonths[period]
	if !ok {
		months = 12
	}

	resp := &BenchmarkResponse{Indices: []BenchmarkData{}}
	for _, idx := range benchmarkIndices {
		points, err := s.history(ctx, idx.symbol)
		if err != nil || len(points) == 0 {
			continue
		}
		data := summarizeBenchmark(points, months, time.Now())
		data.IndexName = idx.name
		data.Symbol = idx.symbol
		resp.Indices = append(resp.Indices, data)
	}
	return resp, nil
}

func (s *BenchmarkService) history(ctx context.Context, symbol string) ([]BenchmarkPoint, error) {
	s.mu.Lock()
	entry, ok := s.cache[symbol]
	s.mu.Unlock()
	if ok && time.Since(entry.fetched) < benchmarkCacheTTL {
		return entry.points, nil
	}

	points, err := s.fetchYahooChart(ctx, symbol, "5y")
	if err != nil {
		if ok {
			return entry.points, nil // serve stale data rather than nothing
		}
		return nil, err
	}
	s.mu.Lock()
	s.cache[symbol] = benchmarkCacheEntry{points: points, fetched: time.Now()}
	s.mu.Unlock()
	return points, nil
}

// summarizeBenchmark computes trailing returns from daily closes (oldest first)
// and trims the chart points to the last `months` months.
func summarizeBenchmark(points []BenchmarkPoint, months int, now time.Time) BenchmarkData {
	last := points[len(points)-1]
	data := BenchmarkData{Current: last.Value, AsOf: last.Date, Available: []string{}}
	lastDate, _ := time.Parse("2006-01-02", last.Date)
	if lastDate.IsZero() {
		lastDate = now
	}

	type horizon struct {
		key    string
		years  int
		change *float64
		cagr   *float64
	}
	for _, h := range []horizon{{"1y", 1, &data.Change1Y, nil}, {"3y", 3, &data.Change3Y, &data.CAGR3Y}, {"5y", 5, &data.Change5Y, &data.CAGR5Y}} {
		base, ok := closeOnOrBefore(points, lastDate.AddDate(-h.years, 0, 0))
		if !ok || base <= 0 {
			continue
		}
		*h.change = (last.Value/base - 1) * 100
		if h.cagr != nil {
			*h.cagr = (math.Pow(last.Value/base, 1/float64(h.years)) - 1) * 100
		}
		data.Available = append(data.Available, h.key)
	}

	cutoff := lastDate.AddDate(0, -months, 0).Format("2006-01-02")
	data.Points = []BenchmarkPoint{}
	for _, p := range points {
		if p.Date >= cutoff {
			data.Points = append(data.Points, p)
		}
	}
	return data
}

// closeOnOrBefore returns the last close on or before target. It requires the
// history to actually reach back to within a week of target.
func closeOnOrBefore(points []BenchmarkPoint, target time.Time) (float64, bool) {
	t := target.Format("2006-01-02")
	if len(points) == 0 || points[0].Date > target.AddDate(0, 0, 7).Format("2006-01-02") {
		return 0, false
	}
	val, found := 0.0, false
	for _, p := range points {
		if p.Date > t {
			break
		}
		val, found = p.Value, true
	}
	if !found {
		return points[0].Value, true // history starts within a week after target
	}
	return val, true
}

func (s *BenchmarkService) fetchYahooChart(ctx context.Context, symbol, yahooRange string) ([]BenchmarkPoint, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=1d", symbol, yahooRange)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	httpResp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo chart %s: HTTP %d", symbol, httpResp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	var result struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if len(result.Chart.Result) == 0 || len(result.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no data for %s", symbol)
	}

	r := result.Chart.Result[0]
	closes := r.Indicators.Quote[0].Close
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.UTC
	}
	points := make([]BenchmarkPoint, 0, len(r.Timestamp))
	for i, ts := range r.Timestamp {
		if i < len(closes) && closes[i] != nil && *closes[i] > 0 {
			points = append(points, BenchmarkPoint{
				Date:  time.Unix(ts, 0).In(ist).Format("2006-01-02"),
				Value: *closes[i],
			})
		}
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("no closes for %s", symbol)
	}
	return points, nil
}
