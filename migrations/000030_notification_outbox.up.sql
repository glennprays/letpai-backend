-- Evolve notification_logs into a durable outbox: widen the status set to
-- cover the worker lifecycle (pending/sending/dead) and add bookkeeping
-- columns for retry + claim. Existing rows keep status 'queued'/'sent'/
-- 'failed'; the app treats 'queued' as 'pending'. All adds are nullable or
-- defaulted so the current INSERT keeps working before code is updated.
ALTER TABLE notification_logs DROP CONSTRAINT IF EXISTS notification_logs_status_check;
ALTER TABLE notification_logs
    ADD CONSTRAINT notification_logs_status_check
    CHECK (status IN ('queued','pending','sending','sent','failed','dead'));

ALTER TABLE notification_logs
    ADD COLUMN IF NOT EXISTS attempts        INT          NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_attempts    INT          NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS locked_at       TIMESTAMP,
    ADD COLUMN IF NOT EXISTS locked_by       VARCHAR(64),
    ADD COLUMN IF NOT EXISTS phone           VARCHAR(32),
    ADD COLUMN IF NOT EXISTS updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- One gateway message id => one log row, so the webhook compare-and-swap is
-- unambiguous. Partial unique to allow many NULLs (rows never sent).
CREATE UNIQUE INDEX IF NOT EXISTS uq_notification_logs_wamid
    ON notification_logs(whatsapp_message_id)
    WHERE whatsapp_message_id IS NOT NULL;

-- Claim query: WHERE status IN ('pending','queued','failed') AND next_attempt_at <= now()
CREATE INDEX IF NOT EXISTS idx_notification_logs_claim
    ON notification_logs(next_attempt_at)
    WHERE status IN ('pending','queued','failed');
