CREATE TABLE contact_groups (
    group_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    group_name VARCHAR(100) NOT NULL,
    color VARCHAR(7) DEFAULT '#10B981',
    description TEXT,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    UNIQUE (user_id, group_name)
);

CREATE INDEX idx_groups_user ON contact_groups(user_id);
CREATE INDEX idx_groups_sort ON contact_groups(user_id, sort_order);
