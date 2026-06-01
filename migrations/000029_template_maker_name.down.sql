-- Revert MakerName changes: restore original templates.

UPDATE message_templates
SET
  body = E'*Letpai - Bill Split*\n\nHi {{.ParticipantName}}!\n\nYou''ve been added to a bill split session: *{{.SessionName}}*\n\nTotal Amount: *Rp{{.Total}}*\nYour Share: *Rp{{.Share}}*\n\nView your bill and upload proof here:\n{{.URL}}\n\nThank you for using Letpai!',
  variables = '["ParticipantName","SessionName","Total","Share","URL"]'::jsonb,
  updated_at = CURRENT_TIMESTAMP
WHERE key = 'session_notification';

UPDATE message_templates
SET
  body = E'*Letpai - Payment Reminder*\n\nHi {{.ParticipantName}}!\n\nFriendly reminder about your pending payment for: *{{.SessionName}}*\n\nAmount Due: *Rp{{.Share}}*\n\nUpload your proof here:\n{{.URL}}\n\nThank you for using Letpai!',
  variables = '["ParticipantName","SessionName","Share","URL"]'::jsonb,
  updated_at = CURRENT_TIMESTAMP
WHERE key = 'payment_reminder';

UPDATE message_templates
SET
  body = E'*Letpai - Verification Code*\n\nYour verification code is: *{{.Code}}*\n\nThis code expires in {{.ExpiresMinutes}} minutes. Don''t share it with anyone.',
  updated_at = CURRENT_TIMESTAMP
WHERE key = 'otp_code';
