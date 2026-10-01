DROP TABLE IF EXISTS account_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
