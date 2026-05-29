-- Backfill payment_proof_url rows that were stored as raw s3:// URIs
-- before submit_payment.go was switched to UploadResult.PublicURL.
--
-- This rewrites any existing `s3://<bucket>/<key>` into the
-- CDN_URL-prefixed https form that getPublicURL composes at upload
-- time. The CDN host is hardcoded here because env vars aren't
-- available inside migrations; if you re-host MinIO under a
-- different domain in the future this file should be replaced with a
-- one-off script, not re-run.
--
-- Safe to re-run: the substring filter and REPLACE both no-op on rows
-- that already have an https:// URL.
UPDATE session_participants
   SET payment_proof_url = REPLACE(
       payment_proof_url,
       's3://lunatica-store/',
       'https://staging-lunatica-minio.kaskur.site/lunatica-store/')
 WHERE payment_proof_url LIKE 's3://lunatica-store/%';
