CREATE TABLE session_participants (
    participant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES sessions(session_id) ON DELETE CASCADE,
    contact_id UUID REFERENCES contacts(contact_id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    whatsapp_number VARCHAR(20) NOT NULL,
    share_amount BIGINT NOT NULL DEFAULT 0,
    payment_status VARCHAR(20) DEFAULT 'pending' CHECK (payment_status IN ('pending', 'submitted', 'paid', 'rejected')),
    rejection_count INTEGER DEFAULT 0,
    rejection_reason TEXT,
    last_notification_at TIMESTAMP,
    notification_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (session_id, whatsapp_number)
);

CREATE INDEX idx_participants_session ON session_participants(session_id);
CREATE INDEX idx_participants_contact ON session_participants(contact_id);
CREATE INDEX idx_participants_status ON session_participants(session_id, payment_status);
CREATE INDEX idx_participants_last_notif ON session_participants(last_notification_at);
