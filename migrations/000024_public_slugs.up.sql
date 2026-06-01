BEGIN;

-- Short URL-safe slugs for the entities exposed in user-visible URLs.
-- Sessions and session_participants. UUIDs stay as internal PKs;
-- slugs are the public handle.
--
-- Alphabet: [A-Za-z0-9_-]  (nanoid / URL-safe base64, 64 chars)
-- Length:   10
-- Keyspace: 64^10 ≈ 1.15e18
-- Collision math at 1M rows: P ≈ 1M^2 / (2 * 1.15e18) ≈ 4e-7
-- Collision math at 1B rows: P ≈ 1B^2 / (2 * 1.15e18) ≈ 0.4
--
-- We pair this with a UNIQUE constraint + a retry-on-conflict insert
-- path in Go so even a one-in-ten-million collision degrades to a
-- second random pick rather than a 500.

ALTER TABLE sessions
    ADD COLUMN public_slug TEXT;

ALTER TABLE session_participants
    ADD COLUMN public_slug TEXT;

-- Backfill: generate a 10-char slug for every existing row using
-- Postgres's built-in random(). This DB doesn't have pgcrypto's
-- gen_random_bytes() — fine for a one-shot backfill, the Go app
-- uses crypto/rand for new rows. Retry on the vanishingly-rare
-- in-batch collision via the inner LOOP.
DO $$
DECLARE
    chars TEXT := 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-';
    slug  TEXT;
    rec   RECORD;
    i     INTEGER;
BEGIN
    FOR rec IN SELECT session_id FROM sessions WHERE public_slug IS NULL LOOP
        LOOP
            slug := '';
            FOR i IN 1..10 LOOP
                slug := slug || substr(chars, floor(random() * 64)::int + 1, 1);
            END LOOP;
            BEGIN
                UPDATE sessions SET public_slug = slug WHERE session_id = rec.session_id;
                EXIT;
            EXCEPTION WHEN unique_violation THEN
                -- regenerate
            END;
        END LOOP;
    END LOOP;

    FOR rec IN SELECT participant_id FROM session_participants WHERE public_slug IS NULL LOOP
        LOOP
            slug := '';
            FOR i IN 1..10 LOOP
                slug := slug || substr(chars, floor(random() * 64)::int + 1, 1);
            END LOOP;
            BEGIN
                UPDATE session_participants SET public_slug = slug WHERE participant_id = rec.participant_id;
                EXIT;
            EXCEPTION WHEN unique_violation THEN
                -- regenerate
            END;
        END LOOP;
    END LOOP;
END $$;

-- Now that every row has a slug, lock the column NOT NULL + UNIQUE.
ALTER TABLE sessions
    ALTER COLUMN public_slug SET NOT NULL,
    ADD CONSTRAINT uq_sessions_public_slug UNIQUE (public_slug);

ALTER TABLE session_participants
    ALTER COLUMN public_slug SET NOT NULL,
    ADD CONSTRAINT uq_session_participants_public_slug UNIQUE (public_slug);

COMMIT;
