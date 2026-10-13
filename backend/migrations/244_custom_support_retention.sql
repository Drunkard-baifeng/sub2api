-- Support retention is based on creation time, independent of replies/status.
CREATE INDEX IF NOT EXISTS support_tickets_created_idx ON support_tickets(created_at, id);
