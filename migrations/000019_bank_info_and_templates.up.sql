-- Session-level bank info that the host shares with participants on the
-- public payment page so they know where to transfer to.
ALTER TABLE sessions
    ADD COLUMN bank_name TEXT,
    ADD COLUMN bank_account_number TEXT,
    ADD COLUMN bank_account_holder TEXT;

-- Admin-managed notification template store. The default rows cover the
-- three send paths the backend currently issues (initial notify, payment
-- reminder, OTP). The `body` column carries Go text/template syntax —
-- e.g. {{.ParticipantName}}, {{.SessionName}}, {{.Share}}, {{.URL}}.
-- key is unique so the renderer can look up by string key without
-- ambiguity.
CREATE TABLE IF NOT EXISTS message_templates (
    template_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key          TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    description  TEXT,
    body         TEXT NOT NULL,
    variables    JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO message_templates (key, name, description, body, variables) VALUES
(
    'session_notification',
    'Bill split – initial notification',
    'Sent when the host kicks off a session and notifies participants for the first time.',
    E'*Letpai - Bill Split*\n\nHi {{.ParticipantName}}!\n\nYou''ve been added to a bill split session: *{{.SessionName}}*\n\nTotal Amount: *Rp{{.Total}}*\nYour Share: *Rp{{.Share}}*\n\nView your bill and upload proof here:\n{{.URL}}\n\nThank you for using Letpai!',
    '["ParticipantName","SessionName","Total","Share","URL"]'::jsonb
),
(
    'payment_reminder',
    'Payment reminder',
    'Sent when the host reminds a single participant or fires the bulk-reminder action.',
    E'*Letpai - Payment Reminder*\n\nHi {{.ParticipantName}}!\n\nFriendly reminder about your pending payment for: *{{.SessionName}}*\n\nAmount Due: *Rp{{.Share}}*\n\nUpload your proof here:\n{{.URL}}\n\nThank you for using Letpai!',
    '["ParticipantName","SessionName","Share","URL"]'::jsonb
),
(
    'otp_code',
    'OTP verification code',
    'Sent for login / registration OTP verification.',
    E'*Letpai - Verification Code*\n\nYour verification code is: *{{.Code}}*\n\nThis code expires in {{.ExpiresMinutes}} minutes. Don''t share it with anyone.',
    '["Code","ExpiresMinutes"]'::jsonb
)
ON CONFLICT (key) DO NOTHING;
