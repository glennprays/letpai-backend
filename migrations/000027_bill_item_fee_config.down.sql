ALTER TABLE bill_items
    DROP COLUMN IF EXISTS service_charge_percentage,
    DROP COLUMN IF EXISTS tax_percentage;
