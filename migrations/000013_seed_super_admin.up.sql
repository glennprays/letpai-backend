-- Seed initial super admin from environment variables
INSERT INTO admins (admin_id, whatsapp_number, password_hash, full_name, role, is_active, created_at, updated_at)
SELECT
    gen_random_uuid() as admin_id,
    COALESCE(NULLIF(current_setting('admin_phone'), ''), '') as whatsapp_number,
    gen_random_uuid()::text as password_hash,  -- Temporary, will be set by admin via setup-password
    'Super Admin' as full_name,
    'super_admin' as role,
    TRUE as is_active,
    CURRENT_TIMESTAMP as created_at,
    CURRENT_TIMESTAMP as updated_at
WHERE NOT EXISTS (SELECT 1 FROM admins);

-- Note: Password is initially random UUID.
-- First admin should call POST /admin/auth/setup-password after login to set real password
