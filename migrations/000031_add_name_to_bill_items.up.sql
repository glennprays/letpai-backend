-- Add name to bill_items (swagger BillItem requires name; description becomes nullable)
ALTER TABLE bill_items ADD COLUMN name VARCHAR(200);
UPDATE bill_items SET name = description WHERE name IS NULL OR name = '';
ALTER TABLE bill_items ALTER COLUMN name SET NOT NULL;
