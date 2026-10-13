package support

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func uploadTestRouter(timeout time.Duration, entered chan<- int64) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), middleware.ServerTiming(true), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.GetHeader("User"), 10, 64)
		c.Set("support_actor", Actor{ID: id, Admin: strings.HasPrefix(c.Request.URL.Path, "/admin/")})
	})
	for _, route := range []string{"/tickets", "/tickets/:id/messages", "/admin/tickets/:id/messages"} {
		router.POST(route, func(c *gin.Context) { limitSubmission(c, timeout) }, func(c *gin.Context) {
			if entered != nil {
				entered <- actor(c).ID
			}
			_, _, err := messageForm(c)
			if !failForm(c, err) {
				c.Status(http.StatusNoContent)
			}
		})
	}
	return router
}

func startIncompleteUpload(t *testing.T, server *httptest.Server, prefix, contentType string, length int, user int64) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	_, err = fmt.Fprintf(conn, "POST /tickets HTTP/1.1\r\nHost: test\r\nUser: %d\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s", user, contentType, length, prefix)
	require.NoError(t, err)
	return conn
}

func TestUploadFixedDeadlineAndReleasedSlots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, epilogue := range []bool{false, true} {
		t.Run(fmt.Sprintf("epilogue=%v", epilogue), func(t *testing.T) {
			entered := make(chan int64, 4)
			server := httptest.NewServer(uploadTestRouter(400*time.Millisecond, entered))
			defer server.Close()
			prefix, contentType := "--slow\r\nContent-Disposition: form-data; name=\"body\"\r\n\r\nhello", "multipart/form-data; boundary=slow"
			if epilogue {
				req := formRequest(t, "hello", nil)
				data, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				prefix, contentType = string(data), req.Header.Get("Content-Type")
			}
			started := time.Now()
			conn := startIncompleteUpload(t, server, prefix, contentType, len(prefix)+1024, 101)
			require.EqualValues(t, 101, <-entered)
			// Keep sending bytes: the total deadline must not reset on activity.
			writesDone := make(chan struct{})
			go func() {
				defer close(writesDone)
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()
				for range ticker.C {
					if _, err := conn.Write([]byte("x")); err != nil {
						return
					}
				}
			}()
			resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			_ = resp.Body.Close()
			require.Equal(t, http.StatusRequestTimeout, resp.StatusCode, string(body))
			require.Contains(t, string(body), "上传超时")
			require.Less(t, time.Since(started), 2*time.Second)
			_ = conn.Close()
			<-writesDone
			require.Eventually(t, func() bool {
				_, active := submissionUsers.Load(int64(101))
				return !active && len(submissionSlots) == 0
			}, time.Second, time.Millisecond)
			// The same account can upload again after timeout.
			req := formRequest(t, "hello", [][]byte{pngFixture(t)})
			req.URL.Scheme, req.URL.Host = "http", server.Listener.Addr().String()
			req.RequestURI = ""
			req.URL.Path = "/tickets"
			req.Header.Set("User", "101")
			resp, err = server.Client().Do(req)
			require.NoError(t, err)
			_ = resp.Body.Close()
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	}
}

func TestUploadAccountIsolationAcrossRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	entered := make(chan int64, 8)
	server := httptest.NewServer(uploadTestRouter(3*time.Second, entered))
	defer server.Close()
	conn := startIncompleteUpload(t, server, "--slow\r\n", "multipart/form-data; boundary=slow", 1024, 201)
	require.EqualValues(t, 201, <-entered)
	for _, route := range []string{"/tickets", "/tickets/1/messages", "/admin/tickets/1/messages"} {
		req, err := http.NewRequest("POST", server.URL+route, bytes.NewBufferString("incomplete"))
		require.NoError(t, err)
		req.Header.Set("User", "201")
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	}
	// One slow account cannot occupy the remaining three shared slots.
	require.Len(t, submissionSlots, 1)
	req := formRequest(t, "another user", nil)
	req.RequestURI = ""
	req.URL.Scheme, req.URL.Host, req.URL.Path = "http", server.Listener.Addr().String(), "/tickets"
	req.Header.Set("User", "202")
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	_ = conn.Close()
	require.Eventually(t, func() bool {
		_, active := submissionUsers.Load(int64(201))
		return !active && len(submissionSlots) == 0
	}, time.Second, time.Millisecond)
}

func TestUploadDeadlineClearedForKeepAlive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	timeout := 100 * time.Millisecond
	router := uploadTestRouter(timeout, nil)
	router.POST("/echo", func(c *gin.Context) {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Data(http.StatusOK, "text/plain", data)
	})
	server := httptest.NewServer(router)
	defer server.Close()
	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	req := formRequest(t, "hello", nil)
	req.URL.Path = "/tickets"
	req.Header.Set("User", "301")
	require.NoError(t, req.Write(conn))
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	_, err = io.WriteString(conn, "POST /echo HTTP/1.1\r\nHost: test\r\nContent-Length: 5\r\n\r\n")
	require.NoError(t, err)
	// Body of an unrelated request remains readable beyond the upload deadline.
	time.Sleep(2 * timeout)
	_, err = io.WriteString(conn, "hello")
	require.NoError(t, err)
	resp, err = http.ReadResponse(reader, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "hello", string(data))
}

func TestUploadRejectsUnsupportedDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	called := false
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("support_actor", Actor{ID: 401}) })
	router.POST("/", submissionLimit, func(c *gin.Context) { called = true })
	router.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.False(t, called)
	require.Empty(t, submissionSlots)
	_, active := submissionUsers.Load(int64(401))
	require.False(t, active)
}

func TestUploadHTTP2Deadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := httptest.NewUnstartedServer(uploadTestRouter(100*time.Millisecond, nil))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, err := http.NewRequest("POST", server.URL+"/tickets", reader)
	require.NoError(t, err)
	req.Header.Set("User", "501")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=slow")
	server.Client().Timeout = 2 * time.Second
	go func() {
		_, _ = io.WriteString(writer, "--slow\r\nContent-Disposition: form-data; name=\"body\"\r\n\r\nhello")
	}()
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 2, resp.ProtoMajor)
	require.Equal(t, http.StatusRequestTimeout, resp.StatusCode, string(data))
	require.Eventually(t, func() bool {
		_, active := submissionUsers.Load(int64(501))
		return !active && len(submissionSlots) == 0
	}, time.Second, time.Millisecond)
}
