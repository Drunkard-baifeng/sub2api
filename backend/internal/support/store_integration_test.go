package support

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func integrationStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("SUPPORT_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUPPORT_TEST_DSN to an isolated test database")
	}

	bootstrap, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	schema := fmt.Sprintf("support_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	_, err = bootstrap.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	scopedDSN := dsn + " search_path=" + schema
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, e := url.Parse(dsn)
		require.NoError(t, e)
		q := parsed.Query()
		q.Set("search_path", schema)
		parsed.RawQuery = q.Encode()
		scopedDSN = parsed.String()
	}
	db, err := sql.Open("postgres", scopedDSN)
	require.NoError(t, err)
	db.SetMaxOpenConns(8)
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = bootstrap.Exec("DROP SCHEMA " + schema + " CASCADE")
		_ = bootstrap.Close()
	})
	_, err = db.Exec("CREATE TABLE users (id BIGINT PRIMARY KEY, email TEXT NOT NULL); INSERT INTO users VALUES (1,'alice@example.test'),(2,'bob@example.test'),(3,'staff@example.test'),(4,'staff2@example.test')")
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("243_custom_support_tickets.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	// Verify migration is reentrant.
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	retention, err := migrations.FS.ReadFile("244_custom_support_retention.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = db.Exec(string(retention))
		require.NoError(t, err)
	}
	return NewStore(db)
}
func TestTicketLifecycleIntegration(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	alice := Actor{ID: 1}
	bob := Actor{ID: 2}
	staff := Actor{ID: 3, Admin: true}
	staff2 := Actor{ID: 4, Admin: true}
	im, err := sanitizeImage(pngFixture(t))
	require.NoError(t, err)
	id, err := s.Create(ctx, alice, "API request failed", "api", "first", []Image{im})
	require.NoError(t, err)
	d, err := s.Get(ctx, alice, id, 0)
	require.NoError(t, err)
	require.Len(t, d.Messages, 1)
	require.Len(t, d.Messages[0].Attachments, 1)
	require.Equal(t, "", d.Ticket.Email)
	require.False(t, d.Ticket.Unread)
	attachmentID := d.Messages[0].Attachments[0].ID
	// A storage failure rolls back both the message and ticket state.
	require.Error(t, s.Reply(ctx, alice, id, "must roll back", []Image{{MIME: "invalid", Data: []byte("x")}}))
	unchanged, e := s.Get(ctx, alice, id, 0)
	require.NoError(t, e)
	require.Len(t, unchanged.Messages, 1)
	require.Equal(t, d.Ticket.LastMessageID, unchanged.Ticket.LastMessageID)

	_, err = s.Get(ctx, bob, id, 0)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = s.Attachment(ctx, bob, id, attachmentID)
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorIs(t, s.Reply(ctx, bob, id, "intrusion", nil), ErrNotFound)
	require.ErrorIs(t, s.ChangeStatus(ctx, bob, id, "resolved"), ErrNotFound)
	require.ErrorIs(t, s.ChangeStatus(ctx, bob, id, "pending"), ErrNotFound)
	require.ErrorIs(t, s.MarkRead(ctx, bob, id, d.Ticket.LastMessageID), ErrNotFound)
	list, total, err := s.List(ctx, bob, "", false, 1)
	require.NoError(t, err)
	require.Empty(t, list)
	require.Zero(t, total)
	got, err := s.Attachment(ctx, staff, id, attachmentID)
	require.NoError(t, err)
	require.Equal(t, im.Data, got.Data)
	count, err := s.Unread(ctx, staff)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, s.MarkRead(ctx, staff, id, d.Ticket.LastMessageID))
	count, err = s.Unread(ctx, staff)
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = s.Unread(ctx, staff2)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, s.ChangeStatus(ctx, staff, id, "processing"))
	require.NoError(t, s.Reply(ctx, staff, id, "please provide details", nil))
	d, err = s.Get(ctx, alice, id, 0)
	require.NoError(t, err)
	require.Equal(t, "waiting_user", d.Ticket.Status)
	require.True(t, d.Ticket.Unread)
	seen := d.Ticket.LastMessageID
	require.NoError(t, s.Reply(ctx, staff, id, "one more question", nil))
	// A read acknowledgement for an older snapshot must not swallow the newer message.
	require.NoError(t, s.MarkRead(ctx, alice, id, seen))
	count, err = s.Unread(ctx, alice)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	d, err = s.Get(ctx, alice, id, 0)
	require.NoError(t, err)
	require.NoError(t, s.MarkRead(ctx, alice, id, d.Ticket.LastMessageID))
	require.NoError(t, s.MarkRead(ctx, alice, id, seen))
	count, err = s.Unread(ctx, alice)
	require.NoError(t, err)
	require.Zero(t, count)
	require.ErrorIs(t, s.MarkRead(ctx, alice, id, 999999), ErrNotFound)
	require.ErrorIs(t, s.ChangeStatus(ctx, alice, id, "processing"), ErrTransition)
	require.NoError(t, s.Reply(ctx, alice, id, "details", nil))
	d, err = s.Get(ctx, staff, id, 0)
	require.NoError(t, err)
	require.Equal(t, "pending", d.Ticket.Status)
	require.Equal(t, "alice@example.test", d.Ticket.Email)
	require.NoError(t, s.ChangeStatus(ctx, alice, id, "resolved"))
	require.ErrorIs(t, s.Reply(ctx, staff, id, "late", nil), ErrClosed)
	require.NoError(t, s.ChangeStatus(ctx, alice, id, "pending"))

	for i := 0; i < 60; i++ {
		require.NoError(t, s.Reply(ctx, staff, id, fmt.Sprintf("reply %d", i), nil))
	}
	d, err = s.Get(ctx, alice, id, 0)
	require.NoError(t, err)
	require.Len(t, d.Messages, 50)
	require.True(t, d.HasMore)
	older, err := s.Get(ctx, alice, id, d.Messages[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, older.Messages)
	require.Less(t, older.Messages[len(older.Messages)-1].ID, d.Messages[0].ID)
	require.False(t, older.HasMore)

	// HTTP layer: authenticated attachment routes never expose another user's images.
	gin.SetMode(gin.TestMode)
	h := &Handler{store: s}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 2}) })
	h.Register(router.Group("/tickets"), false)
	for _, path := range []string{fmt.Sprintf("/tickets/%d", id), fmt.Sprintf("/tickets/%d/attachments/%d", id, attachmentID)} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 404, w.Code)
		require.NotContains(t, w.Body.String(), "first")
	}
	router = gin.New()
	router.Use(func(c *gin.Context) { c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1}) })
	h.Register(router.Group("/tickets"), false)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", fmt.Sprintf("/tickets/%d/attachments/%d", id, attachmentID), nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "default-src 'none'; sandbox", w.Header().Get("Content-Security-Policy"))
	require.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Resource-Policy"))
	require.Equal(t, `inline; filename="screenshot.png"`, w.Header().Get("Content-Disposition"))
	require.Equal(t, "image/png", w.Header().Get("Content-Type"))
	req := httptest.NewRequest("POST", fmt.Sprintf("/tickets/%d/read", id), strings.NewReader(fmt.Sprintf("{\"message_id\":%d}", d.Ticket.LastMessageID)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	// Concurrent creates stay within the open quota. Each successful operation is atomic.
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Create(ctx, bob, "new issue", "other", "hello", nil)
			mu.Lock()
			defer mu.Unlock()
			if e == nil {
				success++
			} else {
				require.ErrorIs(t, e, ErrOpenLimit)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 5, success)
	_, total, err = s.List(ctx, bob, "", false, 1)
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.NoError(t, s.ChangeStatus(ctx, alice, id, "resolved"))
	for i := 0; i < 5; i++ {
		_, err = s.Create(ctx, alice, "another issue", "other", "hello", nil)
		require.NoError(t, err)
	}
	require.ErrorIs(t, s.ChangeStatus(ctx, alice, id, "pending"), ErrOpenLimit)
}

func TestHTTPReplyPreflightAndExpiredAccessIntegration(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	id, err := s.Create(ctx, Actor{ID: 1}, "HTTP reply", "other", "original", nil)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	h := &Handler{store: s, redis: rdb}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1}) })
	h.Register(router.Group("/tickets"), false)
	call := func() *httptest.ResponseRecorder {
		r := formRequest(t, "reply with screenshot", [][]byte{pngFixture(t)})
		r.URL.Path = fmt.Sprintf("/tickets/%d/messages", id)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	w := call()
	require.Equal(t, 200, w.Code, w.Body.String())
	d, err := s.Get(ctx, Actor{ID: 1}, id, 0)
	require.NoError(t, err)
	require.Len(t, d.Messages, 2)
	require.Len(t, d.Messages[1].Attachments, 1)
	_, err = s.db.Exec(`UPDATE support_tickets SET created_at=NOW()-INTERVAL '7 days' WHERE id=$1`, id)
	require.NoError(t, err)
	require.Equal(t, 404, call().Code)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/tickets", nil))
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"retention_days":7`)
	require.Contains(t, w.Body.String(), `"items":[]`)
}
