package services

import (
	"context"
	"log"
	"sync"
	"time"

	"investment-tracker/repository"
)

// startupDelay lets the server finish booting before the catch-up run.
const startupDelay = 30 * time.Second

type Scheduler struct {
	userRepo            *repository.UserRepo
	mfService           *MutualFundService
	stockService        *StockService
	netWorthService     *NetWorthService
	notificationService *NotificationService
	messaging           *MessagingService
	stopCh              chan struct{}
	stopOnce            sync.Once
	runMu               sync.Mutex
}

func NewScheduler(userRepo *repository.UserRepo, mfService *MutualFundService, stockService *StockService, netWorthService *NetWorthService, notificationService *NotificationService) *Scheduler {
	return &Scheduler{
		userRepo:            userRepo,
		mfService:           mfService,
		stockService:        stockService,
		netWorthService:     netWorthService,
		notificationService: notificationService,
		stopCh:              make(chan struct{}),
	}
}

// SetMessaging enables email/Telegram digests after each notification scan.
func (s *Scheduler) SetMessaging(m *MessagingService) { s.messaging = m }

// Start begins the daily scheduler (20:00 IST). It also runs the per-user tasks
// shortly after startup, so hosts that sleep or restart (e.g. free tiers) still
// get this month's net worth snapshot and fresh notifications.
func (s *Scheduler) Start() {
	go s.run()
	go func() {
		select {
		case <-time.After(startupDelay):
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			s.runPerUserTasks(ctx)
		case <-s.stopCh:
		}
	}()
	log.Println("[Scheduler] Started — daily refresh at 20:00 IST")
}

func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		log.Println("[Scheduler] Stopped")
	})
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Println("[Scheduler] Running daily tasks...")

	// 1. Refresh MF NAVs (all users)
	if funds, err := s.mfService.RefreshAllNAVs(ctx, ""); err != nil {
		log.Printf("[Scheduler] NAV refresh error: %v", err)
	} else {
		log.Printf("[Scheduler] Refreshed NAV for %d funds", len(funds))
	}

	// 2. Refresh stock prices (all users)
	if stocks, err := s.stockService.RefreshAllPrices(ctx, ""); err != nil {
		log.Printf("[Scheduler] Stock price refresh error: %v", err)
	} else {
		log.Printf("[Scheduler] Refreshed prices for %d stocks", len(stocks))
	}

	// 3. Per-user net worth snapshot and notifications
	s.runPerUserTasks(ctx)

	// 4. Housekeeping: drop read notifications older than a year
	if s.notificationService != nil {
		if err := s.notificationService.PurgeOld(ctx, 365*24*time.Hour); err != nil {
			log.Printf("[Scheduler] Notification cleanup error: %v", err)
		}
	}

	log.Println("[Scheduler] Daily tasks complete")
}

// runPerUserTasks takes each user's net worth snapshot (one per month, updated
// in place) and generates their notifications. Each user is processed with an
// owner-scoped context so data can never mix between users.
func (s *Scheduler) runPerUserTasks(ctx context.Context) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	if removed, err := s.netWorthService.DeleteOrphanSnapshots(ctx); err != nil {
		log.Printf("[Scheduler] Orphan snapshot cleanup error: %v", err)
	} else if removed > 0 {
		log.Printf("[Scheduler] Removed %d net worth snapshots without a user", removed)
	}

	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to list users: %v", err)
		return
	}
	snapshots, notified := 0, 0
	for _, u := range users {
		if ctx.Err() != nil {
			log.Printf("[Scheduler] Stopped per-user tasks: %v", ctx.Err())
			break
		}
		uid := u.ID.Hex()
		uctx := repository.WithOwner(ctx, uid)
		if _, err := s.netWorthService.TakeSnapshot(uctx, uid); err != nil {
			log.Printf("[Scheduler] Net worth snapshot failed for user %s: %v", uid, err)
		} else {
			snapshots++
		}
		if s.notificationService != nil {
			if _, err := s.notificationService.GenerateNotifications(uctx, uid); err != nil {
				log.Printf("[Scheduler] Notifications failed for user %s: %v", uid, err)
			} else {
				notified++
			}
			if s.messaging != nil {
				if _, err := s.messaging.DispatchPending(uctx, uid); err != nil {
					log.Printf("[Scheduler] Reminder delivery failed for user %s: %v", uid, err)
				}
			}
		}
	}
	log.Printf("[Scheduler] Net worth snapshots: %d, notification scans: %d (users: %d)", snapshots, notified, len(users))
}

// RunNow triggers an immediate run of all daily tasks (for manual trigger via API).
func (s *Scheduler) RunNow(ctx context.Context) error {
	s.runDailyTasks()
	return nil
}
