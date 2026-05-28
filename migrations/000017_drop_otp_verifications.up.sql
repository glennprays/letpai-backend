-- OTP storage moved to Redis. The Postgres tables are no longer read or
-- written by the live code path; drop them.
--
-- ROLLBACK NOTE: the down migration recreates the table structure but
-- not the data — in-flight OTPs are short-lived (≤ a few minutes) and
-- not worth migrating between backends.
DROP TABLE IF EXISTS otp_verifications;
DROP TABLE IF EXISTS admin_otp_verifications;
