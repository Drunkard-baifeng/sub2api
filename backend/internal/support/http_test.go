package support

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func pngFixture(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 8, 8))
	im.Set(0, 0, color.RGBA{R: 20, G: 200, A: 255})
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, im))
	return b.Bytes()
}
func TestScreenshotValidation(t *testing.T) {
	original := pngFixture(t)
	clean, err := sanitizeImage(append(append([]byte{}, original...), []byte("<script>trailing</script>")...))
	require.NoError(t, err)
	require.Equal(t, "image/png", clean.MIME)
	require.NotContains(t, string(clean.Data), "<script>")
	for _, data := range [][]byte{[]byte("<svg onload='alert(1)'></svg>"), []byte("not an image"), original[:30]} {
		_, err = sanitizeImage(data)
		require.Error(t, err)
	}
	huge := append([]byte{}, original...)
	binary.BigEndian.PutUint32(huge[16:20], 8193)
	binary.BigEndian.PutUint32(huge[29:33], crc32.ChecksumIEEE(huge[12:29]))
	_, err = sanitizeImage(huge)
	require.ErrorContains(t, err, "尺寸")
}
func formRequest(t *testing.T, body string, images [][]byte) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	require.NoError(t, w.WriteField("body", body))
	for _, data := range images {
		part, err := w.CreateFormFile("images", "screen.png")
		require.NoError(t, err)
		_, err = part.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	req := httptest.NewRequest("POST", "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}
func TestMessageLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, body string
		images     [][]byte
		ok         bool
	}{
		{"empty", "", nil, false}, {"text", "hello", nil, true}, {"image only", "", [][]byte{pngFixture(t)}, true},
		{"too many", "", [][]byte{pngFixture(t), pngFixture(t), pngFixture(t), pngFixture(t)}, false},
		{"oversized", "", [][]byte{make([]byte, maxImageBytes+1)}, false},
		{"spoofed", "text", [][]byte{[]byte("<svg/>")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = formRequest(t, tt.body, tt.images)
			_, _, err := messageForm(c)
			if tt.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
func TestStatusTransitions(t *testing.T) {
	require.True(t, validTransition("pending", "processing", true))
	require.True(t, validTransition("processing", "waiting_user", true))
	require.True(t, validTransition("waiting_user", "resolved", false))
	require.True(t, validTransition("resolved", "pending", false))
	require.False(t, validTransition("pending", "processing", false))
	require.False(t, validTransition("resolved", "waiting_user", true))
	require.False(t, validTransition("pending", "unknown", true))
	require.False(t, validTransition("pending", "pending", true))
}
func TestRateLimitSharedUserAndExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	h := NewHandler(nil, rdb)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.GetHeader("User"), 10, 64)
		c.Set("support_actor", Actor{ID: id})
	})
	router.POST("/", h.limit("create", 5, 3600), func(c *gin.Context) { c.Status(204) })
	call := func(id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("User", id)
		router.ServeHTTP(w, req)
		return w
	}
	for i := 0; i < 5; i++ {
		require.Equal(t, 204, call("1").Code)
	}
	denied := call("1")
	require.Equal(t, 429, denied.Code)
	require.Equal(t, "3600", denied.Header().Get("Retry-After"))
	require.Equal(t, 204, call("2").Code)
	mr.FastForward(time.Hour)
	require.Equal(t, 204, call("1").Code)
	mr.SetError("offline")
	require.Equal(t, 503, call("3").Code)
}
func TestUnauthenticatedAndAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, nil)
	router := gin.New()
	h.Register(router.Group("/tickets"), false)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/tickets", nil))
	require.Equal(t, 401, w.Code)
	router = gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 12})
		c.Set(string(middleware.ContextKeyUserRole), "user")
	})
	h.Register(router.Group("/admin/tickets"), true)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/admin/tickets", nil))
	require.Equal(t, 403, w.Code)
}
func TestForeignTicketStopsBeforeImageParsing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT").WithArgs(int64(2), false, int64(10)).WillReturnError(sql.ErrNoRows)
	h := NewHandler(db, nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("support_actor", Actor{ID: 2})
	c.Params = gin.Params{{Key: "id", Value: "10"}}
	c.Request = httptest.NewRequest("POST", "/tickets/10/messages", bytes.NewBufferString("invalid multipart"))
	h.reply(c)
	require.Equal(t, 404, c.Writer.Status())
	require.NoError(t, mock.ExpectationsWereMet())
}
