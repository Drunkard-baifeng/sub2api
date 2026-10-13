package support

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRenamedExecutableAndScriptUploads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for name, data := range map[string][]byte{
		"executable": []byte("MZ\x90\x00renamed executable"), "elf": []byte("\x7fELFexecutable"),
		"script": []byte("<?php echo 'not a picture'; ?>"), "html": []byte("<!DOCTYPE html><script>alert(1)</script>"),
		"svg": []byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>"), "archive": []byte("PK\x03\x04zip"),
		"oversized": make([]byte, maxImageBytes+1), "truncated": pngFixture(t)[:40],
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = formRequest(t, "", [][]byte{data})
			_, _, err := messageForm(c)
			require.Error(t, err)
		})
	}
}

func TestImageMetadataAndTrailingDataRemoved(t *testing.T) {
	payload := []byte("Comment\x00<?php metadata should be removed ?>")
	original := pngFixture(t)
	chunk := make([]byte, 12+len(payload))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(payload)))
	copy(chunk[4:8], "tEXt")
	copy(chunk[8:], payload)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	combined := append(append(append([]byte{}, original[:33]...), chunk...), original[33:]...)
	combined = append(combined, []byte("MZ-extra-executable-data")...)
	clean, err := sanitizeImage(combined)
	require.NoError(t, err)
	require.NotContains(t, string(clean.Data), "metadata should")
	require.NotContains(t, string(clean.Data), "MZ-extra")
	_, format, err := image.Decode(bytes.NewReader(clean.Data))
	require.NoError(t, err)
	require.Equal(t, "png", format)
	var jpg bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil))
	// JPEG COM segment immediately after SOI.
	comment := []byte("<?php jpeg metadata ?>")
	segment := append([]byte{0xff, 0xfe, 0, byte(len(comment) + 2)}, comment...)
	jpegData := append(append(append([]byte{}, jpg.Bytes()[:2]...), segment...), jpg.Bytes()[2:]...)
	jpegData = append(jpegData, []byte("MZ-extra-executable-data")...)
	clean, err = sanitizeImage(jpegData)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", clean.MIME)
	require.NotContains(t, string(clean.Data), "jpeg metadata")
	require.NotContains(t, string(clean.Data), "MZ-extra")
	_, format, err = image.Decode(bytes.NewReader(clean.Data))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
}

func TestImageFilenameAndEncoderBounds(t *testing.T) {
	for _, name := range []string{"screen.png", "屏幕截图.JPG", "image.jpeg"} {
		require.True(t, validImageFilename(name), name)
	}
	for _, name := range []string{"", "screen.exe", "screen.png.exe", "../screen.png", "..\\screen.png", "screen.png:payload", "screen%00.png", "screen\x00.png", "screen\n.png", "screen.png ", strings.Repeat("a", 256) + ".png"} {
		require.False(t, validImageFilename(name), name)
	}
	b := boundedImageBuffer{remaining: 4}
	n, err := b.Write([]byte("1234"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
	n, err = b.Write([]byte("5678"))
	require.Error(t, err)
	require.Zero(t, n)
	require.Equal(t, "1234", b.String())
	var raw bytes.Buffer
	w := multipart.NewWriter(&raw)
	part, err := w.CreateFormFile("images", "screen.exe")
	require.NoError(t, err)
	_, err = part.Write(pngFixture(t))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", &raw)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	_, _, err = messageForm(c)
	require.ErrorContains(t, err, "文件名")
}

func TestDeleteRoutesRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	h := NewHandler(db, rdb)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		c.Set(string(middleware.ContextKeyUserRole), c.GetHeader("Role"))
	})
	h.Register(router.Group("/tickets"), false)
	h.Register(router.Group("/admin/tickets"), true)
	call := func(path, role string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", path, nil)
		r.Header.Set("Role", role)
		router.ServeHTTP(w, r)
		return w
	}
	for _, role := range []string{"user", "admin"} {
		require.Equal(t, 404, call("/tickets/10", role).Code)
	}
	require.Equal(t, 403, call("/admin/tickets/10", "user").Code)
	mock.ExpectExec("DELETE FROM support_tickets").WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 1))
	require.Equal(t, 200, call("/admin/tickets/10", "admin").Code)
	mock.ExpectExec("DELETE FROM support_tickets").WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 0))
	require.Equal(t, 404, call("/admin/tickets/10", "admin").Code)
	// Direct handler invocation still checks the actor; no database query is reached.
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("support_actor", Actor{ID: 1})
	h.delete(c)
	require.Equal(t, 403, c.Writer.Status())
	require.ErrorIs(t, NewStore(nil).Delete(context.Background(), Actor{ID: 1}, 10), ErrForbidden)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImageBudgetAtomicAndFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	h := NewHandler(nil, rdb)
	call := func(id int64, images []Image) (bool, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", nil)
		c.Set("support_actor", Actor{ID: id})
		return h.chargeImages(c, images), w
	}
	key := "support:image-bytes:1"
	require.NoError(t, mr.Set(key, strconv.Itoa(maxImageDailyBytes-32)))
	mr.SetTTL(key, time.Hour)
	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := call(1, []Image{{Data: make([]byte, 16)}})
			if ok {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 2, successes.Load())
	ok, w := call(1, []Image{{Data: []byte("x")}})
	require.False(t, ok)
	require.Equal(t, 429, w.Code)
	require.Equal(t, "3600", w.Header().Get("Retry-After"))
	ok, _ = call(2, []Image{{Data: []byte("x")}})
	require.True(t, ok)
	require.Equal(t, 24*time.Hour, mr.TTL("support:image-bytes:2"))
	mr.FastForward(time.Hour)
	ok, _ = call(1, []Image{{Data: []byte("x")}})
	require.True(t, ok)
	mr.SetError("offline")
	ok, w = call(3, []Image{{Data: []byte("x")}})
	require.False(t, ok)
	require.Equal(t, 503, w.Code)
	ok, _ = call(3, nil)
	require.True(t, ok)
}

func TestSubmissionConcurrencyBound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for i := 0; i < cap(submissionSlots); i++ {
		submissionSlots <- struct{}{}
	}
	defer func() {
		for len(submissionSlots) > 0 {
			<-submissionSlots
		}
	}()
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("support_actor", Actor{ID: 1}) })
	router.POST("/", submissionLimit, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	require.Equal(t, 503, w.Code)
	<-submissionSlots
	w = httptest.NewRecorder()
	router.ServeHTTP(deadlineRecorder{w}, httptest.NewRequest("POST", "/", nil))
	require.Equal(t, 204, w.Code)
	require.Len(t, submissionSlots, cap(submissionSlots)-1)
}

// Production uses net/http's native deadlines; the concurrency unit test has no socket.
type deadlineRecorder struct{ *httptest.ResponseRecorder }

func (deadlineRecorder) SetReadDeadline(time.Time) error { return nil }
