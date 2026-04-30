package services

import (
	"context"
	"log"
	"time"
)

type Scheduler struct {
	mfService       *MutualFundService
	stockService    *StockService
	netWorthService *NetWorthService
	stopCh          chan struct{}
}

func NewScheduler(mfService *MutualFundService, stockService *StockService, netWorthService *NetWorthService) *Scheduler {
	return &Scheduler{
		mfService:       mfService,
		stockService:    stockService,
		netWorthService: netWorthService,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the daily scheduler. It runs tasks at the configured hour (default 20:00 IST).
func (s *Scheduler) Start() {
	go s.run()
	log.Println("[Scheduler] Started — daily refresh at 20:00 IST")
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	log.Println("[Scheduler] Stopped")
}

func (s *Scheduler) run() {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("[Scheduler] Failed to load IST timezone, using UTC: %v", err)
		ist = time.UTC
	}

	for {
		now := time.Now().In(ist)
		// Schedule for 20:00 today, or tomorrow if past 20:00
		next := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, ist)
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		wait := time.Until(next)
		log.Printf("[Scheduler] Next run at %s (in %s)", next.Format("2006-01-02 15:04 MST"), wait.Round(time.Minute))

		select {
		case <-time.After(wait):
			s.runDailyTasks()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) runDailyTasks() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("[Scheduler] Running daily tasks...")

	// 1. Refresh MF NAVs
	if funds, err := s.mfService.RefreshAllNAVs(ctx); err != nil {
		log.Printf("[Scheduler] NAV refresh error: %v", err)
	} else {
		log.Printf("[Scheduler] Refreshed NAV for %d funds", len(funds))
	}

	// 2. Refresh stock prices
	if stocks, err := s.stockService.RefreshAllPrices(ctx); err != nil {
		log.Printf("[Scheduler] Stock price refresh error: %v", err)
	} else {
		log.Printf("[Scheduler] Refreshed prices for %d stocks", len(stocks))
	}

	// 3. Take net worth snapshot (for all users — pass empty string)
	if _, err := s.netWorthService.TakeSnapshot(ctx, ""); err != nil {
		log.Printf("[Scheduler] Net worth snapshot error: %v", err)
	} else {
		log.Println("[Scheduler] Net worth snapshot taken")
	}

	log.Println("[Scheduler] Daily tasks complete")
}

// RunNow triggers an immediate run of all daily tasks (for manual trigger via API).
func (s *Scheduler) RunNow(ctx context.Context) error {
	s.runDailyTasks()
	return nil
}
