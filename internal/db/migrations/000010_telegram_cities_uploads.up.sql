-- Telegram sign-in: accounts created from a Telegram Mini App have no email
-- or password, only a Telegram user id.
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ADD COLUMN telegram_id BIGINT UNIQUE;
ALTER TABLE users ADD COLUMN telegram_username TEXT;
ALTER TABLE users ADD CONSTRAINT users_has_login CHECK (email IS NOT NULL OR telegram_id IS NOT NULL);

-- Location-aware matching: users pick a city; wants match within it unless
-- the user is willing to trade with other cities (e.g. by shipping).
ALTER TABLE users ADD COLUMN city TEXT;
ALTER TABLE wants ADD COLUMN any_city BOOLEAN NOT NULL DEFAULT false;

-- Outbox for delivering notifications as Telegram messages. Existing rows
-- are treated as already delivered so nothing old is sent on deploy.
ALTER TABLE notifications ADD COLUMN delivered_at TIMESTAMPTZ;
UPDATE notifications SET delivered_at = created_at;
CREATE INDEX idx_notifications_undelivered ON notifications (created_at) WHERE delivered_at IS NULL;
