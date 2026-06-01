BEGIN;
DROP TRIGGER IF EXISTS trg_bill_items_touch_session ON bill_items;
DROP TRIGGER IF EXISTS trg_session_participants_touch_session ON session_participants;
DROP FUNCTION IF EXISTS touch_session_updated_at();
ALTER TABLE sessions DROP COLUMN IF EXISTS last_notified_at;
COMMIT;
