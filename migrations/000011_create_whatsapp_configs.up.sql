CREATE TABLE whatsapp_configs (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    gateway_token TEXT,
    is_connected BOOLEAN DEFAULT FALSE,
    qr_code_base64 TEXT,
    qr_code_expires_at TIMESTAMP,
    connection_status VARCHAR(20) DEFAULT 'disconnected' CHECK (connection_status IN ('connected', 'disconnected', 'qr_pending', 'failed')),
    last_connected_at TIMESTAMP,
    last_disconnected_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_whatsapp_configs_phone ON whatsapp_configs(phone_number);
CREATE INDEX idx_whatsapp_configs_status ON whatsapp_configs(connection_status);
