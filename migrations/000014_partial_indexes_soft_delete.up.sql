-- Partial indexes for soft-delete tables.
--
-- Common query shape: `WHERE user_id = $1 AND deleted_at IS NULL`. Without
-- a partial index Postgres falls back to a sequential scan once tables grow,
-- making list endpoints (sessions, contacts, contact_groups) slow under load.
-- Partial indexes are also smaller and faster to maintain than full indexes
-- because they exclude soft-deleted rows.

CREATE INDEX IF NOT EXISTS idx_sessions_user_active
	ON sessions (user_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_user_active
	ON contacts (user_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_contact_groups_user_active
	ON contact_groups (user_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_admins_active
	ON admins (whatsapp_number) WHERE deleted_at IS NULL;

-- Bill items: SumBySessionID hits session_id heavily; existing
-- idx_bill_items_session covers it but make it covering for the sum.
CREATE INDEX IF NOT EXISTS idx_bill_items_session_amount
	ON bill_items (session_id) INCLUDE (amount);
