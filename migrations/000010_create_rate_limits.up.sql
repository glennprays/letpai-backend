CREATE TABLE rate_limits (
    rate_limit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    identifier VARCHAR(255) NOT NULL,
    request_count INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    window_end TIMESTAMP NOT NULL,

    UNIQUE (identifier, window_start)
);

CREATE INDEX idx_rate_user ON rate_limits(user_id);
CREATE INDEX idx_rate_identifier ON rate_limits(identifier);
CREATE INDEX idx_rate_window ON rate_limits(window_end);
