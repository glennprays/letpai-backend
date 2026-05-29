BEGIN;
DROP INDEX IF EXISTS idx_session_bank_accounts_session;
DROP INDEX IF EXISTS uq_session_bank_accounts_ordinal;
DROP TABLE IF EXISTS session_bank_accounts;
COMMIT;
