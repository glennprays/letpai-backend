-- Reverse the URL rewrite. Largely cosmetic — keeping it here so
-- `migrate down` stays consistent. Live rows that have been visited
-- since 000020 will already round-trip through the new code path,
-- so this is best-effort.
UPDATE session_participants
   SET payment_proof_url = REPLACE(
       payment_proof_url,
       'https://staging-lunatica-minio.kaskur.site/lunatica-store/',
       's3://lunatica-store/')
 WHERE payment_proof_url LIKE 'https://staging-lunatica-minio.kaskur.site/lunatica-store/%';
