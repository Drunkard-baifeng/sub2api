// Package support implements the personal Fork's isolated support-ticket module.
package support

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

var ErrNotFound = errors.New("ticket not found")
var ErrClosed = errors.New("ticket is resolved; reopen before replying")
var ErrOpenLimit = errors.New("at most five open tickets are allowed")
var ErrTransition = errors.New("invalid status transition")
var ErrForbidden = errors.New("admin role required")

type Actor struct {
	ID    int64
	Admin bool
}
type Ticket struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Email         string    `json:"email,omitempty"`
	Subject       string    `json:"subject"`
	Category      string    `json:"category"`
	Status        string    `json:"status"`
	LastMessageID int64     `json:"last_message_id"`
	Unread        bool      `json:"unread"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}
type Attachment struct {
	ID   int64  `json:"id"`
	MIME string `json:"mime"`
	Size int    `json:"size"`
}
type Image struct {
	MIME string
	Data []byte
}
type Message struct {
	ID          int64        `json:"id"`
	IsAdmin     bool         `json:"is_admin"`
	Body        string       `json:"body"`
	StatusEvent string       `json:"status_event"`
	CreatedAt   time.Time    `json:"created_at"`
	Attachments []Attachment `json:"attachments"`
}
type Detail struct {
	Ticket   Ticket    `json:"ticket"`
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
}

const defaultRetentionDays = 7

type Store struct {
	db            *sql.DB
	retentionDays int
}

func NewStore(db *sql.DB) *Store { return &Store{db: db, retentionDays: defaultRetentionDays} }

// The interval is produced only from a bounded integer, never request input.
func (s *Store) days() int {
	if s.retentionDays < 1 || s.retentionDays > 365 {
		return defaultRetentionDays
	}
	return s.retentionDays
}
func (s *Store) intervalSQL() string { return "INTERVAL '" + strconv.Itoa(s.days()) + " days'" }
func (s *Store) activeSQL(alias string) string {
	return alias + ".created_at > NOW() - " + s.intervalSQL()
}
func (s *Store) columns() string { return ticketColumns + ",t.created_at + " + s.intervalSQL() }

const ticketColumns = `t.id,t.user_id,CASE WHEN $2 THEN u.email ELSE '' END,t.subject,t.category,t.status,t.last_message_id,
(CASE WHEN $2 THEN t.last_user_message_id ELSE t.last_admin_message_id END > COALESCE(r.message_id,0)),t.created_at,t.updated_at`
const ticketJoins = ` FROM support_tickets t JOIN users u ON u.id=t.user_id
 LEFT JOIN support_ticket_reads r ON r.ticket_id=t.id AND r.reader_id=$1 AND r.admin_view=$2 `

func scanTicket(row interface{ Scan(...any) error }) (Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.UserID, &t.Email, &t.Subject, &t.Category, &t.Status, &t.LastMessageID, &t.Unread, &t.CreatedAt, &t.UpdatedAt, &t.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return t, err
}
func (s *Store) List(ctx context.Context, a Actor, status string, unread bool, page int) ([]Ticket, int, error) {
	where := ` WHERE ` + s.activeSQL("t") + ` AND ($2 OR t.user_id=$1) AND ($3='' OR t.status=$3) AND (NOT $4 OR CASE WHEN $2 THEN t.last_user_message_id ELSE t.last_admin_message_id END>COALESCE(r.message_id,0))`
	args := []any{a.ID, a.Admin, status, unread}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+ticketJoins+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+s.columns()+ticketJoins+where+` ORDER BY t.updated_at DESC,t.id DESC LIMIT 20 OFFSET $5`, append(args, (page-1)*20)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := []Ticket{}
	for rows.Next() {
		t, e := scanTicket(rows)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}
func (s *Store) Unread(ctx context.Context, a Actor) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+ticketJoins+` WHERE `+s.activeSQL("t")+` AND ($2 OR t.user_id=$1) AND CASE WHEN $2 THEN t.last_user_message_id ELSE t.last_admin_message_id END>COALESCE(r.message_id,0)`, a.ID, a.Admin).Scan(&n)
	return n, err
}
func (s *Store) Get(ctx context.Context, a Actor, id, before int64) (Detail, error) {
	var d Detail
	t, err := scanTicket(s.db.QueryRowContext(ctx, `SELECT `+s.columns()+ticketJoins+` WHERE `+s.activeSQL("t")+` AND t.id=$3 AND ($2 OR t.user_id=$1)`, a.ID, a.Admin, id))
	if err != nil {
		return d, err
	}
	d.Ticket = t
	rows, err := s.db.QueryContext(ctx, `SELECT id,is_admin,body,status_event,created_at FROM support_messages WHERE ticket_id=$1 AND id<=$2 AND ($3::bigint=0 OR id<$3) AND EXISTS (SELECT 1 FROM support_tickets t WHERE t.id=$1 AND `+s.activeSQL("t")+` AND ($4 OR t.user_id=$5)) ORDER BY id DESC LIMIT 51`, id, t.LastMessageID, before, a.Admin, a.ID)
	if err != nil {
		return d, err
	}
	d.Messages = []Message{}
	for rows.Next() {
		var m Message
		m.Attachments = []Attachment{}
		if err = rows.Scan(&m.ID, &m.IsAdmin, &m.Body, &m.StatusEvent, &m.CreatedAt); err != nil {
			_ = rows.Close()
			return d, err
		}
		d.Messages = append(d.Messages, m)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return d, err
	}
	if len(d.Messages) > 50 {
		d.HasMore = true
		d.Messages = d.Messages[:50]
	}
	for i, j := 0, len(d.Messages)-1; i < j; i, j = i+1, j-1 {
		d.Messages[i], d.Messages[j] = d.Messages[j], d.Messages[i]
	}
	if len(d.Messages) == 0 {
		return d, nil
	}
	ar, err := s.db.QueryContext(ctx, `SELECT a.id,a.message_id,a.mime,octet_length(a.data) FROM support_attachments a JOIN support_messages m ON m.id=a.message_id JOIN support_tickets t ON t.id=m.ticket_id WHERE `+s.activeSQL("t")+` AND m.ticket_id=$1 AND m.id BETWEEN $2 AND $3 AND ($4 OR t.user_id=$5) ORDER BY a.id`, id, d.Messages[0].ID, d.Messages[len(d.Messages)-1].ID, a.Admin, a.ID)
	if err != nil {
		return d, err
	}
	defer func() { _ = ar.Close() }()
	positions := map[int64]int{}
	for i, m := range d.Messages {
		positions[m.ID] = i
	}
	for ar.Next() {
		var at Attachment
		var mid int64
		if err = ar.Scan(&at.ID, &mid, &at.MIME, &at.Size); err != nil {
			return d, err
		}
		if i, ok := positions[mid]; ok {
			d.Messages[i].Attachments = append(d.Messages[i].Attachments, at)
		}
	}
	return d, ar.Err()
}
func (s *Store) lockTicket(ctx context.Context, tx *sql.Tx, a Actor, id int64) (string, error) {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM support_tickets t WHERE `+s.activeSQL("t")+` AND id=$1 AND ($2 OR user_id=$3) FOR UPDATE`, id, a.Admin, a.ID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return status, err
}
func addMessage(ctx context.Context, tx *sql.Tx, a Actor, id int64, body, status, event string, images []Image) error {
	var mid int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO support_messages(ticket_id,author_id,is_admin,body,status_event) VALUES($1,$2,$3,$4,$5) RETURNING id`, id, a.ID, a.Admin, body, event).Scan(&mid); err != nil {
		return err
	}
	for _, im := range images {
		if _, err := tx.ExecContext(ctx, `INSERT INTO support_attachments(message_id,mime,data) VALUES($1,$2,$3)`, mid, im.MIME, im.Data); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `UPDATE support_tickets SET status=$2,updated_at=NOW(),last_message_id=$3,last_user_message_id=CASE WHEN NOT $4 THEN $3 ELSE last_user_message_id END,last_admin_message_id=CASE WHEN $4 THEN $3 ELSE last_admin_message_id END WHERE id=$1`, id, status, mid, a.Admin)
	return err
}
func (s *Store) Create(ctx context.Context, a Actor, subject, category, body string, images []Image) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	// Lock the owner row so concurrent requests cannot bypass the open-ticket quota.
	var uid int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 FOR NO KEY UPDATE`, a.ID).Scan(&uid); err != nil {
		return 0, err
	}
	var n int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM support_tickets t WHERE `+s.activeSQL("t")+` AND user_id=$1 AND status<>'resolved'`, a.ID).Scan(&n); err != nil {
		return 0, err
	}
	if n >= 5 {
		return 0, ErrOpenLimit
	}
	var id int64
	if err = tx.QueryRowContext(ctx, `INSERT INTO support_tickets(user_id,subject,category) VALUES($1,$2,$3) RETURNING id`, a.ID, subject, category).Scan(&id); err != nil {
		return 0, err
	}
	if err = addMessage(ctx, tx, a, id, body, "pending", "", images); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
func (s *Store) Reply(ctx context.Context, a Actor, id int64, body string, images []Image) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	status, err := s.lockTicket(ctx, tx, a, id)
	if err != nil {
		return err
	}
	if status == "resolved" {
		return ErrClosed
	}
	status = "pending"
	if a.Admin {
		status = "waiting_user"
	}
	if err = addMessage(ctx, tx, a, id, body, status, "", images); err != nil {
		return err
	}
	return tx.Commit()
}
func validTransition(current, next string, admin bool) bool {
	if current == next {
		return false
	}
	if current == "resolved" {
		return next == "pending"
	}
	if admin {
		return next == "processing" || next == "waiting_user" || next == "resolved" || next == "pending"
	}
	return next == "resolved"
}
func (s *Store) ChangeStatus(ctx context.Context, a Actor, id int64, next string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Same lock order as Create; reopening also observes the open-ticket quota.
	if next == "pending" {
		var owner int64
		err = tx.QueryRowContext(ctx, `SELECT u.id FROM users u JOIN support_tickets t ON t.user_id=u.id WHERE `+s.activeSQL("t")+` AND t.id=$1 AND ($2 OR t.user_id=$3) FOR NO KEY UPDATE OF u`, id, a.Admin, a.ID).Scan(&owner)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM support_tickets t WHERE `+s.activeSQL("t")+` AND user_id=$1 AND status<>'resolved' AND id<>$2`, owner, id).Scan(&count); err != nil {
			return err
		}
		if count >= 5 {
			return ErrOpenLimit
		}
	}
	current, err := s.lockTicket(ctx, tx, a, id)
	if err != nil {
		return err
	}
	if !validTransition(current, next, a.Admin) {
		return ErrTransition
	}
	if err = addMessage(ctx, tx, a, id, "", next, next, nil); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) MarkRead(ctx context.Context, a Actor, id, mid int64) error {
	result, err := s.db.ExecContext(ctx, `INSERT INTO support_ticket_reads(ticket_id,reader_id,admin_view,message_id) SELECT t.id,$2,$3,m.id FROM support_tickets t JOIN support_messages m ON m.ticket_id=t.id WHERE `+s.activeSQL("t")+` AND t.id=$1 AND ($3 OR t.user_id=$2) AND m.id=$4 ON CONFLICT(ticket_id,reader_id,admin_view) DO UPDATE SET message_id=GREATEST(support_ticket_reads.message_id,EXCLUDED.message_id)`, id, a.ID, a.Admin, mid)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}
func (s *Store) Attachment(ctx context.Context, a Actor, id, attachmentID int64) (Image, error) {
	var im Image
	err := s.db.QueryRowContext(ctx, `SELECT a.mime,a.data FROM support_attachments a JOIN support_messages m ON m.id=a.message_id JOIN support_tickets t ON t.id=m.ticket_id WHERE `+s.activeSQL("t")+` AND t.id=$1 AND a.id=$2 AND ($3 OR t.user_id=$4)`, id, attachmentID, a.Admin, a.ID).Scan(&im.MIME, &im.Data)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return im, err
}

// Delete is admin-only. Foreign keys cascade to messages, screenshots and read cursors.
func (s *Store) Delete(ctx context.Context, a Actor, id int64) error {
	if !a.Admin {
		return ErrForbidden
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM support_tickets WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// DeleteExpiredBatch is bounded and safe for multiple running application instances.
func (s *Store) DeleteExpiredBatch(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `WITH expired AS (
 SELECT id FROM support_tickets
 WHERE created_at <= NOW() - `+s.intervalSQL()+`
 ORDER BY created_at,id LIMIT 100 FOR UPDATE SKIP LOCKED
 ) DELETE FROM support_tickets t USING expired e WHERE t.id=e.id`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
