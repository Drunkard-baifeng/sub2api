-- Private attachments are included in database backups.
CREATE TABLE IF NOT EXISTS support_tickets (
 id BIGSERIAL PRIMARY KEY,
 user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 subject VARCHAR(120) NOT NULL,
 category VARCHAR(20) NOT NULL CHECK (category IN ('recharge','redemption','api','other')),
 status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','waiting_user','resolved')),
 last_message_id BIGINT NOT NULL DEFAULT 0,
 last_user_message_id BIGINT NOT NULL DEFAULT 0,
 last_admin_message_id BIGINT NOT NULL DEFAULT 0,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS support_tickets_user_updated_idx ON support_tickets(user_id, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS support_tickets_status_updated_idx ON support_tickets(status, updated_at DESC, id DESC);
CREATE TABLE IF NOT EXISTS support_messages (
 id BIGSERIAL PRIMARY KEY,
 ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
 author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
 is_admin BOOLEAN NOT NULL DEFAULT FALSE,
 body TEXT NOT NULL DEFAULT '',
 status_event VARCHAR(20) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS support_messages_ticket_idx ON support_messages(ticket_id, id DESC);
CREATE TABLE IF NOT EXISTS support_attachments (
 id BIGSERIAL PRIMARY KEY,
 message_id BIGINT NOT NULL REFERENCES support_messages(id) ON DELETE CASCADE,
 mime VARCHAR(20) NOT NULL CHECK (mime IN ('image/png','image/jpeg')),
 data BYTEA NOT NULL CHECK (octet_length(data) <= 2097152)
);
CREATE INDEX IF NOT EXISTS support_attachments_message_idx ON support_attachments(message_id);
CREATE TABLE IF NOT EXISTS support_ticket_reads (
 ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
 reader_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 admin_view BOOLEAN NOT NULL,
 message_id BIGINT NOT NULL DEFAULT 0,
 PRIMARY KEY(ticket_id, reader_id, admin_view)
);
