CREATE TABLE bill_items (
    bill_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES sessions(session_id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'IDR',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_bill_items_session ON bill_items(session_id);
CREATE INDEX idx_bill_items_created ON bill_items(created_at DESC);
