BEGIN;

-- Per-session "last sent" timestamp. The dirty predicate (
--     last_notified_at IS NULL
--     OR sessions.updated_at > last_notified_at
-- ) gates the send-notifications endpoint so a host can't fire the
-- batch repeatedly without making a change first. NULL by default
-- so every existing session is considered first-send-allowed.
ALTER TABLE sessions
    ADD COLUMN last_notified_at TIMESTAMP NULL;

-- Child mutations (participants, bills) must bump sessions.updated_at
-- so the predicate above is a one-row check. Doing this via triggers
-- rather than in every Go use-case is the design call from the
-- planning doc — Add/Remove/Update participant, Add/Update/Delete
-- bill, MarkPaidManually all touch session_participants or
-- bill_items, so a single trigger pair covers every existing AND
-- future write path. The trade-off (invisible to code review) is
-- accepted because the alternative (per-use-case bumps) is a
-- chronic miss-the-spot bug source.
--
-- Important: there is intentionally no trigger on the `sessions`
-- table itself. sessionRepo.MarkNotified updates last_notified_at
-- directly; if a session-self trigger touched updated_at on every
-- write, the dirty gate would re-open the moment we closed it.
CREATE OR REPLACE FUNCTION touch_session_updated_at() RETURNS TRIGGER AS $$
BEGIN
    UPDATE sessions
       SET updated_at = CURRENT_TIMESTAMP
     WHERE session_id = COALESCE(NEW.session_id, OLD.session_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_session_participants_touch_session
AFTER INSERT OR UPDATE OR DELETE ON session_participants
FOR EACH ROW EXECUTE FUNCTION touch_session_updated_at();

CREATE TRIGGER trg_bill_items_touch_session
AFTER INSERT OR UPDATE OR DELETE ON bill_items
FOR EACH ROW EXECUTE FUNCTION touch_session_updated_at();

COMMIT;
