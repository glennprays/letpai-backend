-- Reverse: bare key back to the (former) CDN-prefixed https form.
UPDATE session_participants
   SET payment_proof_url = 'https://s3.rylo.lol/letpai/' || payment_proof_url
 WHERE payment_proof_url LIKE 'payments/%';
