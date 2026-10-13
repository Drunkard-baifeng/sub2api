package support

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetentionPolicyConfiguration(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		days  int
		valid bool
	}{
		{"", 7, true}, {"  ", 7, true}, {"7", 7, true}, {"1", 1, true}, {"365", 365, true},
		{"0", 0, false}, {"-1", 0, false}, {"366", 0, false}, {"7.5", 0, false}, {"invalid", 0, false},
	} {
		t.Run(fmt.Sprintf("%q", tc.raw), func(t *testing.T) {
			t.Setenv("SUPPORT_TICKET_RETENTION_DAYS", tc.raw)
			days, err := retentionDaysFromEnv()
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, tc.days, days)
			} else {
				require.Error(t, err)
				_, err = ProvideHandler(nil, nil)
				require.Error(t, err)
			}
		})
	}
}

func TestAdminDeleteAndCascadeIntegration(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	owner := Actor{ID: 1}
	staff := Actor{ID: 3, Admin: true}
	im, err := sanitizeImage(pngFixture(t))
	require.NoError(t, err)
	for _, resolved := range []bool{false, true} {
		id, err := s.Create(ctx, owner, "Delete test", "other", "hello", []Image{im})
		require.NoError(t, err)
		if resolved {
			require.NoError(t, s.ChangeStatus(ctx, owner, id, "resolved"))
		}
		d, err := s.Get(ctx, owner, id, 0)
		require.NoError(t, err)
		require.NoError(t, s.MarkRead(ctx, staff, id, d.Ticket.LastMessageID))
		require.ErrorIs(t, s.Delete(ctx, owner, id), ErrForbidden)
		require.ErrorIs(t, s.Delete(ctx, Actor{ID: 2}, id), ErrForbidden)
		require.ErrorIs(t, s.Delete(ctx, Actor{ID: 3}, id), ErrForbidden)
		_, err = s.Get(ctx, owner, id, 0)
		require.NoError(t, err)
		require.NoError(t, s.Delete(ctx, staff, id))
		require.ErrorIs(t, s.Delete(ctx, staff, id), ErrNotFound)
		for _, table := range []string{"support_tickets", "support_messages", "support_attachments", "support_ticket_reads"} {
			var n int
			require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&n))
			require.Zero(t, n, table)
		}
	}
}

func TestExpiryHiddenAndPurgedIntegration(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	owner := Actor{ID: 1}
	staff := Actor{ID: 3, Admin: true}
	im, err := sanitizeImage(pngFixture(t))
	require.NoError(t, err)
	for _, status := range []string{"pending", "processing", "waiting_user", "resolved"} {
		id, err := s.Create(ctx, owner, "Expired ticket", "api", "original", []Image{im})
		require.NoError(t, err)
		require.NoError(t, s.Reply(ctx, staff, id, "recent reply does not extend retention", nil))
		d, err := s.Get(ctx, owner, id, 0)
		require.NoError(t, err)
		aid := d.Messages[0].Attachments[0].ID
		require.NoError(t, s.MarkRead(ctx, staff, id, d.Ticket.LastMessageID))
		_, err = s.db.Exec(`UPDATE support_tickets SET status=$2,created_at=NOW()-INTERVAL '7 days',updated_at=NOW() WHERE id=$1`, id, status)
		require.NoError(t, err)
		for _, a := range []Actor{owner, staff} {
			_, err = s.Get(ctx, a, id, 0)
			require.ErrorIs(t, err, ErrNotFound)
			_, err = s.Attachment(ctx, a, id, aid)
			require.ErrorIs(t, err, ErrNotFound)
			require.ErrorIs(t, s.MarkRead(ctx, a, id, d.Ticket.LastMessageID), ErrNotFound)
			require.ErrorIs(t, s.Reply(ctx, a, id, "late", nil), ErrNotFound)
			require.ErrorIs(t, s.ChangeStatus(ctx, a, id, "pending"), ErrNotFound)
			require.ErrorIs(t, s.ChangeStatus(ctx, a, id, "resolved"), ErrNotFound)
			items, total, err := s.List(ctx, a, "", false, 1)
			require.NoError(t, err)
			require.Zero(t, total)
			require.Empty(t, items)
			unread, err := s.Unread(ctx, a)
			require.NoError(t, err)
			require.Zero(t, unread)
		}
	}
	fresh, err := s.Create(ctx, owner, "Fresh ticket", "other", "still here", nil)
	require.NoError(t, err)
	_, err = s.db.Exec(`UPDATE support_tickets SET created_at=NOW()-INTERVAL '6 days 23 hours' WHERE id=$1`, fresh)
	require.NoError(t, err)
	d, err := s.Get(ctx, owner, fresh, 0)
	require.NoError(t, err)
	require.WithinDuration(t, d.Ticket.CreatedAt.Add(7*24*time.Hour), d.Ticket.ExpiresAt, time.Second)
	n, err := s.DeleteExpiredBatch(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 4, n)
	for table, want := range map[string]int{"support_tickets": 1, "support_messages": 1, "support_attachments": 0, "support_ticket_reads": 0} {
		var count int
		require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Equal(t, want, count, table)
	}
	_, err = s.Get(ctx, owner, fresh, 0)
	require.NoError(t, err)
}

func TestExpiredTicketsDoNotConsumeOpenQuotaIntegration(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	a := Actor{ID: 1}
	for i := 0; i < 5; i++ {
		_, err := s.Create(ctx, a, "old ticket", "other", "hello", nil)
		require.NoError(t, err)
	}
	_, err := s.Create(ctx, a, "over quota", "other", "hello", nil)
	require.ErrorIs(t, err, ErrOpenLimit)
	_, err = s.db.Exec(`UPDATE support_tickets SET created_at=NOW()-INTERVAL '8 days'`)
	require.NoError(t, err)
	id, err := s.Create(ctx, a, "new ticket", "other", "hello", nil)
	require.NoError(t, err)
	require.NoError(t, s.ChangeStatus(ctx, a, id, "resolved"))
	require.NoError(t, s.ChangeStatus(ctx, a, id, "pending"))
}

func TestCleanupBatchConcurrencyAndShutdownIntegration(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	_, err := s.db.Exec(`INSERT INTO support_tickets(user_id,subject,category,created_at) SELECT 1,'old','other',NOW()-INTERVAL '8 days' FROM generate_series(1,205)`)
	require.NoError(t, err)
	n, err := s.DeleteExpiredBatch(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 100, n)
	var wg sync.WaitGroup
	counts := make(chan int64, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); n, e := s.DeleteExpiredBatch(ctx); counts <- n; errs <- e }()
	}
	wg.Wait()
	close(counts)
	close(errs)
	sum := int64(0)
	for e := range errs {
		require.NoError(t, e)
	}
	for n := range counts {
		require.LessOrEqual(t, n, int64(100))
		sum += n
	}
	require.EqualValues(t, 105, sum)
	_, err = s.db.Exec(`INSERT INTO support_tickets(user_id,subject,category,created_at) VALUES(1,'old','other',NOW()-INTERVAL '8 days')`)
	require.NoError(t, err)
	stop := s.startCleanup(20 * time.Millisecond)
	t.Cleanup(stop)
	require.Eventually(t, func() bool {
		var n int
		return s.db.QueryRow("SELECT COUNT(*) FROM support_tickets").Scan(&n) == nil && n == 0
	}, 3*time.Second, 10*time.Millisecond)
	stop()
	stop()
	_, err = s.db.Exec(`INSERT INTO support_tickets(user_id,subject,category,created_at) VALUES(1,'old','other',NOW()-INTERVAL '8 days')`)
	require.NoError(t, err)
	time.Sleep(60 * time.Millisecond)
	var left int
	require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM support_tickets").Scan(&left))
	require.Equal(t, 1, left)
}
