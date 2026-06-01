BEGIN;

-- Per-session 1..N bank accounts. The host can list multiple transfer
-- destinations (e.g. BCA + GoPay + an e-wallet) and the participant
-- payment page renders each as a copy-able card.
--
-- Replaces the single sessions.bank_name / bank_account_number /
-- bank_account_holder columns. Those legacy columns stay through one
-- release as a read-compat shim — see migration 000024 (separate
-- release) for the drop.
--
-- Why a separate table and not a JSONB column on sessions:
--   * Lets us index session_id alone for the typical "load all
--     accounts for one session" query.
--   * The CHECK constraint is enforceable per-row instead of
--     hand-validated in the use case.
--   * Soft-delete via deleted_at gives BulkReplace a clean
--     "swap-in-place" path that doesn't lose history.

CREATE TABLE session_bank_accounts (
    account_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    ordinal        SMALLINT NOT NULL,            -- 0-based display order
    bank_name      TEXT,
    account_number TEXT,
    account_holder TEXT,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at     TIMESTAMP,
    CONSTRAINT session_bank_accounts_at_least_one_field
        CHECK (
            COALESCE(NULLIF(TRIM(bank_name), ''),
                     NULLIF(TRIM(account_number), ''),
                     NULLIF(TRIM(account_holder), '')) IS NOT NULL
        )
);

-- ordinal uniqueness is per-session, scoped to live rows only so a
-- soft-deleted row at ordinal 0 doesn't block re-using that slot.
CREATE UNIQUE INDEX uq_session_bank_accounts_ordinal
    ON session_bank_accounts(session_id, ordinal)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_session_bank_accounts_session
    ON session_bank_accounts(session_id)
    WHERE deleted_at IS NULL;

-- Backfill: copy the single-account legacy row into ordinal 0 if any
-- bank field has a non-empty value. Sessions where all three legacy
-- fields are NULL or empty are skipped (the host never filled it in).
INSERT INTO session_bank_accounts (session_id, ordinal, bank_name, account_number, account_holder)
SELECT s.session_id,
       0,
       NULLIF(TRIM(s.bank_name), ''),
       NULLIF(TRIM(s.bank_account_number), ''),
       NULLIF(TRIM(s.bank_account_holder), '')
  FROM sessions s
 WHERE s.deleted_at IS NULL
   AND COALESCE(NULLIF(TRIM(s.bank_name), ''),
                NULLIF(TRIM(s.bank_account_number), ''),
                NULLIF(TRIM(s.bank_account_holder), '')) IS NOT NULL;

COMMIT;
