package support

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const maxImageBytes = 2 << 20
const maxRequestBytes = 7 << 20
const maxImageDailyBytes = 32 << 20
const uploadTimeout = 30 * time.Second

var screenshotSlots = make(chan struct{}, 2)
var submissionSlots = make(chan struct{}, 4)
var submissionUsers sync.Map
var errUploadTimeout = errors.New("上传超时，请在 30 秒内完成上传后重试")

type Handler struct {
	store *Store
	redis *redis.Client
}

func NewHandler(db *sql.DB, rdb *redis.Client) *Handler { return &Handler{NewStore(db), rdb} }
func (h *Handler) Register(g *gin.RouterGroup, admin bool) {
	g.Use(func(c *gin.Context) {
		subject, ok := middleware.GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}
		if admin {
			role, _ := middleware.GetUserRoleFromContext(c)
			if role != "admin" {
				response.Forbidden(c, "需要管理员身份")
				c.Abort()
				return
			}
		}
		c.Set("support_actor", Actor{subject.UserID, admin})
		c.Header("Cache-Control", "no-store")
	})
	g.GET("", h.list)
	g.GET("/unread", h.unread)
	g.GET("/:id", h.detail)
	if !admin {
		g.POST("", h.limit("create", 5, 3600), submissionLimit, h.create)
	} else {
		g.DELETE("/:id", h.limit("delete", 20, 60), h.delete)
	}
	g.POST("/:id/messages", h.limit("reply", 20, 60), submissionLimit, h.reply)
	g.PATCH("/:id/status", h.limit("status", 20, 60), h.status)
	g.POST("/:id/read", h.read)
	g.GET("/:id/attachments/:attachment", h.attachment)
}
func actor(c *gin.Context) Actor { a, _ := c.MustGet("support_actor").(Actor); return a }
func ticketID(c *gin.Context) int64 {
	n, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if n <= 0 {
		response.BadRequest(c, "工单编号无效")
		return 0
	}
	return n
}

var rateScript = redis.NewScript(`local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],ARGV[1]) end; return {n,redis.call('TTL',KEYS[1])}`)

func (h *Handler) limit(action string, max, seconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.redis == nil {
			response.Error(c, 503, "提交服务暂不可用，请稍后重试")
			c.Abort()
			return
		}
		result, err := rateScript.Run(c.Request.Context(), h.redis, []string{fmt.Sprintf("support:rate:%d:%s", actor(c).ID, action)}, seconds).Int64Slice()
		if err != nil || len(result) != 2 {
			response.Error(c, 503, "提交服务暂不可用，请稍后重试")
			c.Abort()
			return
		}
		if result[0] > int64(max) {
			ttl := result[1]
			if ttl < 1 {
				ttl = 1
			}
			c.Header("Retry-After", strconv.FormatInt(ttl, 10))
			response.Error(c, 429, "提交过于频繁，请稍后重试")
			c.Abort()
			return
		}
		c.Next()
	}
}
func fail(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c, "需要管理员身份")
	case errors.Is(err, ErrNotFound):
		response.Error(c, 404, "工单或附件不存在")
	case errors.Is(err, ErrClosed):
		response.Error(c, 409, "工单已解决，请先重新打开")
	case errors.Is(err, ErrOpenLimit):
		response.Error(c, 409, "最多保留 5 个未解决工单")
	case errors.Is(err, ErrTransition):
		response.Error(c, 409, "状态已变化，请刷新后重试")
	default:
		response.ErrorFrom(c, err)
	}
	return true
}
func validStatus(s string) bool {
	return s == "pending" || s == "processing" || s == "waiting_user" || s == "resolved"
}
func (h *Handler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 || page > 100000 {
		response.BadRequest(c, "页码无效")
		return
	}
	status := c.Query("status")
	if status != "" && !validStatus(status) {
		response.BadRequest(c, "状态无效")
		return
	}
	items, total, err := h.store.List(c.Request.Context(), actor(c), status, c.Query("unread") == "true", page)
	if fail(c, err) {
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": 20, "retention_days": h.store.days()})
}
func (h *Handler) unread(c *gin.Context) {
	n, err := h.store.Unread(c.Request.Context(), actor(c))
	if !fail(c, err) {
		response.Success(c, gin.H{"count": n})
	}
}
func (h *Handler) detail(c *gin.Context) {
	id := ticketID(c)
	if id == 0 {
		return
	}
	before, _ := strconv.ParseInt(c.DefaultQuery("before", "0"), 10, 64)
	if before < 0 {
		response.BadRequest(c, "游标无效")
		return
	}
	d, err := h.store.Get(c.Request.Context(), actor(c), id, before)
	if !fail(c, err) {
		response.Success(c, d)
	}
}

// Decode and re-encode images to reject spoofed files and remove metadata/trailing payloads.
func sanitizeImage(data []byte) (Image, error) {
	var result Image
	if len(data) == 0 || len(data) > maxImageBytes {
		return result, errors.New("每张截图最多 2 MB")
	}
	mime := http.DetectContentType(data)
	if mime != "image/png" && mime != "image/jpeg" {
		return result, errors.New("仅支持有效的 PNG 或 JPEG 截图")
	}
	select {
	case screenshotSlots <- struct{}{}:
		defer func() { <-screenshotSlots }()
	default:
		return result, errors.New("截图处理繁忙，请稍后重试")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") || "image/"+format != mime {
		return result, errors.New("仅支持有效的 PNG 或 JPEG 截图")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
		return result, errors.New("截图尺寸过大，最多 1600 万像素")
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || decodedFormat != format {
		return result, errors.New("截图内容损坏")
	}
	if decoded.Bounds().Dx() != cfg.Width || decoded.Bounds().Dy() != cfg.Height {
		return result, errors.New("截图尺寸与内容不一致")
	}
	out := boundedImageBuffer{remaining: maxImageBytes}
	if format == "png" {
		err = png.Encode(&out, decoded)
		result.MIME = "image/png"
	} else {
		err = jpeg.Encode(&out, decoded, &jpeg.Options{Quality: 90})
		result.MIME = "image/jpeg"
	}
	if err != nil || out.Len() > maxImageBytes {
		return Image{}, errors.New("截图处理后超过 2 MB，请压缩后上传")
	}
	result.Data = out.Bytes()
	return result, nil
}

// Cap encoded output as it is produced, not after an unbounded allocation.
type boundedImageBuffer struct {
	bytes.Buffer
	remaining int
}

func (b *boundedImageBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, errors.New("encoded image exceeds limit")
	}
	n, err := b.Buffer.Write(p)
	b.remaining -= n
	return n, err
}

func validImageFilename(name string) bool {
	if len(name) == 0 || len(name) > 255 || !utf8.ValidString(name) || strings.TrimSpace(name) != name || strings.ContainsAny(name, "/\\:%") {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

func submissionLimit(c *gin.Context) {
	limitSubmission(c, uploadTimeout)
}

func limitSubmission(c *gin.Context, timeout time.Duration) {
	select {
	case submissionSlots <- struct{}{}:
		defer func() { <-submissionSlots }()
	default:
		c.Header("Connection", "close")
		c.Header("Retry-After", "2")
		response.Error(c, 503, "提交服务繁忙，请稍后重试")
		c.Abort()
		return
	}
	id := actor(c).ID
	if _, loaded := submissionUsers.LoadOrStore(id, struct{}{}); loaded {
		c.Header("Connection", "close")
		c.Header("Retry-After", "2")
		response.Error(c, 429, "已有提交正在上传，请稍后重试")
		c.Abort()
		return
	}
	defer submissionUsers.Delete(id)

	// A context timeout alone cannot interrupt a blocked socket read. Set a
	// fixed deadline on the native request stream, including ownership checks.
	controller := http.NewResponseController(c.Writer)
	if err := controller.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		c.Header("Connection", "close")
		response.Error(c, 503, "上传服务暂不可用，请稍后重试")
		c.Abort()
		return
	}
	body := &submissionBody{ReadCloser: c.Request.Body}
	c.Request.Body = body
	defer func() {
		// Keep the deadline for unread/error bodies: net/http may drain them
		// after the handler returns. Only fully read requests can reuse it safely.
		if body.complete {
			_ = controller.SetReadDeadline(time.Time{})
		}
	}()
	c.Next()
}

type submissionBody struct {
	io.ReadCloser
	complete bool
}

func (b *submissionBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF {
		b.complete = true
	}
	return n, err
}

func formReadError(err error) error {
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return errUploadTimeout
	}
	return errors.New("上传内容过大或格式无效")
}

func failForm(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errUploadTimeout) {
		response.Error(c, http.StatusRequestTimeout, err.Error())
	} else {
		response.BadRequest(c, err.Error())
	}
	return true
}

var imageBudgetScript = redis.NewScript(`local n=tonumber(redis.call('GET',KEYS[1]) or '0'); if n+tonumber(ARGV[1])>tonumber(ARGV[2]) then return {0,redis.call('TTL',KEYS[1])} end; n=redis.call('INCRBY',KEYS[1],ARGV[1]); if redis.call('TTL',KEYS[1])<0 then redis.call('EXPIRE',KEYS[1],86400) end; return {1,redis.call('TTL',KEYS[1])}`)

// A shared 24-hour byte budget bounds image storage per account across all tickets.
// Failed writes still consume the budget so retries do not create an abuse bypass.
func (h *Handler) chargeImages(c *gin.Context, images []Image) bool {
	n := 0
	for _, im := range images {
		n += len(im.Data)
	}
	if n == 0 {
		return true
	}
	if h.redis == nil {
		response.Error(c, 503, "提交服务暂不可用，请稍后重试")
		return false
	}
	result, err := imageBudgetScript.Run(c.Request.Context(), h.redis, []string{fmt.Sprintf("support:image-bytes:%d", actor(c).ID)}, n, maxImageDailyBytes).Int64Slice()
	if err != nil || len(result) != 2 {
		response.Error(c, 503, "提交服务暂不可用，请稍后重试")
		return false
	}
	if result[0] != 1 {
		c.Header("Retry-After", strconv.FormatInt(max(result[1], 1), 10))
		response.Error(c, 429, "24 小时内截图上传总量最多 32 MB，请稍后重试")
		return false
	}
	return true
}
func messageForm(c *gin.Context) (string, []Image, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)
	if err := c.Request.ParseMultipartForm(maxRequestBytes); err != nil {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
		return "", nil, formReadError(err)
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	// Multipart parsing may stop at the closing boundary before HTTP body EOF.
	// Bound and drain the epilogue under the same deadline, too.
	if _, err := io.Copy(io.Discard, c.Request.Body); err != nil {
		return "", nil, formReadError(err)
	}
	body := strings.TrimSpace(c.PostForm("body"))
	if !utf8.ValidString(body) || utf8.RuneCountInString(body) > 5000 {
		return "", nil, errors.New("文字最多 5000 字")
	}
	files := c.Request.MultipartForm.File["images"]
	if len(files) > 3 {
		return "", nil, errors.New("每条消息最多 3 张截图")
	}
	for key := range c.Request.MultipartForm.File {
		if key != "images" {
			return "", nil, errors.New("附件字段无效")
		}
	}
	if body == "" && len(files) == 0 {
		return "", nil, errors.New("请输入文字或添加截图")
	}
	images := []Image{}
	for _, file := range files {
		if !validImageFilename(file.Filename) {
			return "", nil, errors.New("截图文件名无效，请使用 PNG 或 JPEG 文件")
		}
		if file.Size > maxImageBytes {
			return "", nil, errors.New("每张截图最多 2 MB")
		}
		f, err := file.Open()
		if err != nil {
			return "", nil, err
		}
		data, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
		_ = f.Close()
		if err != nil || len(data) > maxImageBytes {
			return "", nil, errors.New("截图读取失败或超过 2 MB")
		}
		im, err := sanitizeImage(data)
		if err != nil {
			return "", nil, err
		}
		images = append(images, im)
	}
	return body, images, nil
}
func (h *Handler) create(c *gin.Context) {
	body, images, err := messageForm(c)
	if failForm(c, err) {
		return
	}
	subject := strings.TrimSpace(c.PostForm("subject"))
	category := c.PostForm("category")
	if !utf8.ValidString(subject) || utf8.RuneCountInString(subject) < 2 || utf8.RuneCountInString(subject) > 120 {
		response.BadRequest(c, "标题需要 2–120 字")
		return
	}
	if category != "recharge" && category != "redemption" && category != "api" && category != "other" {
		response.BadRequest(c, "请选择问题类型")
		return
	}
	if !h.chargeImages(c, images) {
		return
	}
	id, err := h.store.Create(c.Request.Context(), actor(c), subject, category, body, images)
	if !fail(c, err) {
		response.Created(c, gin.H{"id": id})
	}
}
func (h *Handler) reply(c *gin.Context) {
	id := ticketID(c)
	if id == 0 {
		return
	}
	// Verify ownership before parsing expensive image data; transactional write checks again.
	_, err := scanTicket(h.store.db.QueryRowContext(c.Request.Context(), `SELECT `+h.store.columns()+ticketJoins+` WHERE `+h.store.activeSQL("t")+` AND t.id=$3 AND ($2 OR t.user_id=$1)`, actor(c).ID, actor(c).Admin, id))
	if fail(c, err) {
		return
	}
	body, images, err := messageForm(c)
	if failForm(c, err) {
		return
	}
	if !h.chargeImages(c, images) {
		return
	}
	if !fail(c, h.store.Reply(c.Request.Context(), actor(c), id, body, images)) {
		response.Success(c, gin.H{"ok": true})
	}
}
func (h *Handler) status(c *gin.Context) {
	id := ticketID(c)
	if id == 0 {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var req struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&req) != nil || !validStatus(req.Status) {
		response.BadRequest(c, "状态无效")
		return
	}
	if !fail(c, h.store.ChangeStatus(c.Request.Context(), actor(c), id, req.Status)) {
		response.Success(c, gin.H{"ok": true})
	}
}
func (h *Handler) read(c *gin.Context) {
	id := ticketID(c)
	if id == 0 {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var req struct {
		MessageID int64 `json:"message_id"`
	}
	if c.ShouldBindJSON(&req) != nil || req.MessageID <= 0 {
		response.BadRequest(c, "已读位置无效")
		return
	}
	if !fail(c, h.store.MarkRead(c.Request.Context(), actor(c), id, req.MessageID)) {
		response.Success(c, gin.H{"ok": true})
	}
}
func (h *Handler) attachment(c *gin.Context) {
	id := ticketID(c)
	if id == 0 {
		return
	}
	aid, _ := strconv.ParseInt(c.Param("attachment"), 10, 64)
	if aid <= 0 {
		response.BadRequest(c, "附件编号无效")
		return
	}
	im, err := h.store.Attachment(c.Request.Context(), actor(c), id, aid)
	if fail(c, err) {
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	if im.MIME != "image/png" && im.MIME != "image/jpeg" {
		response.Error(c, 404, "附件不存在")
		return
	}
	ext := "png"
	if im.MIME == "image/jpeg" {
		ext = "jpg"
	}
	c.Header("Content-Disposition", `inline; filename="screenshot.`+ext+`"`)
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Data(200, im.MIME, im.Data)
}

func (h *Handler) delete(c *gin.Context) {
	if !actor(c).Admin {
		fail(c, ErrForbidden)
		return
	}
	id := ticketID(c)
	if id == 0 {
		return
	}
	if !fail(c, h.store.Delete(c.Request.Context(), actor(c), id)) {
		response.Success(c, gin.H{"ok": true})
	}
}
