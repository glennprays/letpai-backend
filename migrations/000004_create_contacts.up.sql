CREATE TABLE contacts (
    contact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    whatsapp_number VARCHAR(20) NOT NULL,
    group_id UUID REFERENCES contact_groups(group_id) ON DELETE SET NULL,
    avatar_url TEXT,
    notes TEXT,
    is_favorite BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    UNIQUE (user_id, whatsapp_number)
);

CREATE INDEX idx_contacts_user ON contacts(user_id);
CREATE INDEX idx_contacts_group ON contacts(group_id);
CREATE INDEX idx_contacts_fav ON contacts(user_id, is_favorite) WHERE is_favorite = TRUE;
