-- whatsapp_configs was created in migration 11 without a deleted_at
-- column, but the repository (internal/repository/whatsapp_config_postgres.go)
-- has been issuing `WHERE deleted_at IS NULL` against it on every
-- read/write. That meant every /admin/status call has been returning
-- 500 silently — the FE catches the failure on the dashboard and the
-- /admin/whatsapp page falls back to an "Unknown" badge.
--
-- Add the column so the existing predicates work. NULL means "live row".
ALTER TABLE whatsapp_configs ADD COLUMN deleted_at TIMESTAMP;
