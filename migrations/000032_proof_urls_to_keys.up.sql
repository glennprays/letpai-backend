-- payment_proof_url now stores the bare S3 object key; reads presign it
-- (the letpai bucket is private and the old CDN host is unrouted).
UPDATE session_participants
   SET payment_proof_url = regexp_replace(payment_proof_url, '^.*/payments/', 'payments/')
 WHERE payment_proof_url IS NOT NULL
   AND payment_proof_url LIKE '%/payments/%';
