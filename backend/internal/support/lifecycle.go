package support

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const cleanupInterval = 15 * time.Minute

func retentionDaysFromEnv() (int, error) {
	raw := strings.TrimSpace(os.Getenv("SUPPORT_TICKET_RETENTION_DAYS"))
	if raw == "" {
		return defaultRetentionDays, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 || days > 365 {
		return 0, fmt.Errorf("SUPPORT_TICKET_RETENTION_DAYS must be an integer between 1 and 365")
	}
	return days, nil
}

// ProvideHandler validates policy without starting background work.
func ProvideHandler(db *sql.DB, rdb *redis.Client) (*Handler, error) {
	days, err := retentionDaysFromEnv()
	if err != nil {
		return nil, err
	}
	h := NewHandler(db, rdb)
	h.store.retentionDays = days
	return h, nil
}

// StartCleanup is called after successful dependency initialization. Its returned
// stop function joins the worker before the application's database is closed.
func (h *Handler) StartCleanup() func() {
	days := h.store.days()
	slog.Info("support ticket retention enabled", "retention_days", days, "basis", "created_at")
	return h.store.startCleanup(cleanupInterval)
}

func (s *Store) startCleanup(interval time.Duration) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		run := func() {
			// At most 1,000 tickets per sweep; each transaction has a deadline.
			for batch := 0; batch < 10; batch++ {
				if ctx.Err() != nil {
					return
				}
				batchCtx, finish := context.WithTimeout(ctx, 30*time.Second)
				n, err := s.DeleteExpiredBatch(batchCtx)
				finish()
				if err != nil {
					if ctx.Err() == nil {
						slog.Error("support ticket cleanup failed", "error", err)
					}
					return
				}
				if n > 0 {
					slog.Info("expired support tickets deleted", "count", n)
				}
				if n < 100 {
					return
				}
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { cancel(); <-done }) }
}
