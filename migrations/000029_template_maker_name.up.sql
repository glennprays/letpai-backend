-- Add MakerName to session_notification and payment_reminder templates.
-- Update body to include maker name and Letpai URL in signature.

UPDATE message_templates
SET
  body = E'Hello, {{.ParticipantName}} 👋\n\n{{.MakerName}} has added you to a bill split: {{.SessionName}}\n\nBill Summary\n• Total Amount: Rp{{.Total}}\n• Your Share: Rp{{.Share}}\n\nReview the bill and upload your payment proof:\n{{.URL}}\n\nIf you have any questions, contact {{.MakerName}} directly.\n\n— Letpai · Bill Split\nhttps://letpai.app',
  variables = '["ParticipantName","SessionName","MakerName","Total","Share","URL"]'::jsonb,
  updated_at = CURRENT_TIMESTAMP
WHERE key = 'session_notification';

UPDATE message_templates
SET
  body = E'Hey {{.ParticipantName}}, just a reminder! 👋\n\n{{.MakerName}} is waiting for your payment on: {{.SessionName}}\n\nAmount Due: Rp{{.Share}}\n\nUpload your payment proof here:\n{{.URL}}\n\nThanks!\n— Letpai\nhttps://letpai.app',
  variables = '["ParticipantName","SessionName","MakerName","Share","URL"]'::jsonb,
  updated_at = CURRENT_TIMESTAMP
WHERE key = 'payment_reminder';

-- Also add Letpai URL to the OTP template signature
UPDATE message_templates
SET
  body = E'*Letpai - Verification Code*\n\nYour verification code is: *{{.Code}}*\n\nThis code expires in {{.ExpiresMinutes}} minutes. Don''t share it with anyone.\n\n— Letpai\nhttps://letpai.app',
  updated_at = CURRENT_TIMESTAMP
WHERE key = 'otp_code';
