-- Fee configuration for sessions: service charge and tax percentages.
-- Stored directly on sessions (1:1 relationship).
ALTER TABLE sessions
    ADD COLUMN service_charge_percentage NUMERIC(5,2) DEFAULT 0,
    ADD COLUMN tax_percentage NUMERIC(5,2) DEFAULT 0;
