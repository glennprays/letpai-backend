DROP TABLE IF EXISTS message_templates;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS bank_account_holder,
    DROP COLUMN IF EXISTS bank_account_number,
    DROP COLUMN IF EXISTS bank_name;
