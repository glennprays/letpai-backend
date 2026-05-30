BEGIN;
ALTER TABLE session_participants DROP CONSTRAINT IF EXISTS uq_session_participants_public_slug;
ALTER TABLE session_participants DROP COLUMN IF EXISTS public_slug;
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS uq_sessions_public_slug;
ALTER TABLE sessions DROP COLUMN IF EXISTS public_slug;
COMMIT;
