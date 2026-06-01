-- Composite index for the per-participant "latest log" query that
-- powers the session-detail status chip:
--
--   SELECT DISTINCT ON (nl.participant_id)
--          nl.* FROM notification_logs nl
--     JOIN session_participants sp ON sp.participant_id = nl.participant_id
--    WHERE sp.session_id = $1
--    ORDER BY nl.participant_id, nl.sent_at DESC;
--
-- The existing idx_logs_sent indexes sent_at globally; adding
-- (participant_id, sent_at DESC) makes the DISTINCT ON scan a single
-- index range per participant.
CREATE INDEX IF NOT EXISTS idx_notification_logs_participant_sent
    ON notification_logs(participant_id, sent_at DESC);
