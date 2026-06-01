ALTER TABLE bill_items
    DROP COLUMN IF EXISTS includes_service_charge,
    DROP COLUMN IF EXISTS includes_tax,
    ADD COLUMN service_charge_percentage NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN tax_percentage NUMERIC(5,2) DEFAULT 0;
