-- Down migration: delete the seeded super admin only if it still has the placeholder
-- whatsapp_number. Operators who rotated the number/credentials keep their account.
DELETE FROM admins
WHERE role = 'super_admin'
  AND whatsapp_number = '6280000000000';
