-- Convert per-bill-item fee percentages to boolean flags.
-- The session-level fee_config holds the percentages; each item just
-- declares whether those fees apply (includes_service_charge / includes_tax).
ALTER TABLE bill_items
    DROP COLUMN IF EXISTS service_charge_percentage,
    DROP COLUMN IF EXISTS tax_percentage,
    ADD COLUMN includes_service_charge BOOLEAN DEFAULT true,
    ADD COLUMN includes_tax BOOLEAN DEFAULT true;
