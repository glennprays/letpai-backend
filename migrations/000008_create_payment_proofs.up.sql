CREATE TABLE payment_proofs (
    proof_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID REFERENCES session_participants(participant_id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    image_public_id VARCHAR(255),
    file_size INTEGER,
    file_format VARCHAR(10),
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_proofs_participant ON payment_proofs(participant_id);
CREATE INDEX idx_proofs_deleted ON payment_proofs(deleted_at) WHERE is_deleted = FALSE;
