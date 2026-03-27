CREATE TABLE admin_otp_verifications (
    verification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL,
    otp_code VARCHAR(6) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admin_otp_admin_id ON admin_otp_verifications(admin_id);
CREATE INDEX idx_admin_otp_code ON admin_otp_verifications(otp_code);
CREATE INDEX idx_admin_otp_expires ON admin_otp_verifications(expires_at);

ALTER TABLE admin_otp_verifications
ADD CONSTRAINT fk_admin_otp_admin
FOREIGN KEY (admin_id) REFERENCES admins(admin_id) ON DELETE CASCADE;
