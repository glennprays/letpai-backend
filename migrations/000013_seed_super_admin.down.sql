-- Down migration: delete seeded super admin
-- Only deletes if whatsapp_number is empty (indicating seeded admin)
DELETE FROM admins
WHERE role = 'super_admin'
  AND whatsapp_number = ''
  AND created_at = (SELECT MIN(created_at) FROM admins WHERE role = 'super_admin');
