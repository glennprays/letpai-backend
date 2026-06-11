DROP INDEX IF EXISTS idx_notification_logs_claim;
DROP INDEX IF EXISTS uq_notification_logs_wamid;
ALTER TABLE notification_logs
    DROP COLUMN IF EXISTS attempts,
    DROP COLUMN IF EXISTS max_attempts,
    DROP COLUMN IF EXISTS next_attempt_at,
    DROP COLUMN IF EXISTS locked_at,
    DROP COLUMN IF EXISTS locked_by,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS updated_at;
ALTER TABLE notification_logs DROP CONSTRAINT IF EXISTS notification_logs_status_check;
ALTER TABLE notification_logs
    ADD CONSTRAINT notification_logs_status_check
    CHECK (status IN ('queued','sent','failed'));
