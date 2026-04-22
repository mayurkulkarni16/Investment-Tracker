package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type NAVResponse struct {
	Meta map[string]interface{}   `json:"meta"`
	Data []map[string]interface{} `json:"data"`
}

type NAVFetcher struct {
	client *http.Client
}

func NewNAVFetcher() *NAVFetcher {
	return &NAVFetcher{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *NAVFetcher) FetchLatestNAV(schemeCode string) (float64, error) {
	url := fmt.Sprintf("https://api.mfapi.in/mf/%s/latest", schemeCode)

	resp, err := f.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch NAV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("NAV API returned status %d", resp.StatusCode)
	}

	var navResp NAVResponse
	if err := json.NewDecoder(resp.Body).Decode(&navResp); err != nil {
		return 0, fmt.Errorf("failed to decode NAV response: %w", err)
	}

	if len(navResp.Data) == 0 {
		return 0, fmt.Errorf("no NAV data found for scheme %s", schemeCode)
	}

	navStr, ok := navResp.Data[0]["nav"].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected NAV format")
	}

	nav, err := strconv.ParseFloat(navStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse NAV value: %w", err)
	}

	return nav, nil
}
