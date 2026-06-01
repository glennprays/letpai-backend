-- Seed initial super admin
-- Note: This creates a default super admin account.
-- Update the whatsapp_number and password_hash via API or direct SQL after migration.
INSERT INTO admins (admin_id, whatsapp_number, password_hash, full_name, role, is_active, created_at, updated_at)
SELECT
    gen_random_uuid() as admin_id,
    '6280000000000' as whatsapp_number,  -- Default placeholder (13 chars to pass len=13 validation). Rotate before production use.
    '$2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' as password_hash,  -- Placeholder bcrypt hash
    'Super Admin' as full_name,
    'super_admin' as role,
    TRUE as is_active,
    CURRENT_TIMESTAMP as created_at,
    CURRENT_TIMESTAMP as updated_at
WHERE NOT EXISTS (SELECT 1 FROM admins WHERE role = 'super_admin');
