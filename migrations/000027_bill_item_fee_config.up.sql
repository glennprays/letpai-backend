-- Per-bill-item fee configuration: each item can have its own service charge and tax rates.
ALTER TABLE bill_items
    ADD COLUMN service_charge_percentage NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN tax_percentage NUMERIC(5,2) DEFAULT 0;
