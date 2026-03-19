CREATE TABLE notification_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID REFERENCES session_participants(participant_id) ON DELETE CASCADE,
    notification_type VARCHAR(20) NOT NULL CHECK (notification_type IN ('initial', 'reminder', 'rejection')),
    whatsapp_message_id VARCHAR(255),
    message_content TEXT NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'sent' CHECK (status IN ('queued', 'sent', 'failed')),
    error_message TEXT
);

CREATE INDEX idx_logs_participant ON notification_logs(participant_id);
CREATE INDEX idx_logs_sent ON notification_logs(sent_at DESC);
CREATE INDEX idx_logs_status ON notification_logs(status);
