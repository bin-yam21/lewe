ALTER TABLE items DROP CONSTRAINT IF EXISTS items_status_check;
UPDATE items SET status = 'archived' WHERE status = 'private';
ALTER TABLE items ADD CONSTRAINT items_status_check
    CHECK (status IN ('active', 'matched', 'exchanged', 'archived'));

DROP INDEX IF EXISTS idx_matches_origin;
ALTER TABLE matches DROP COLUMN IF EXISTS origin;
ALTER TABLE matches DROP COLUMN IF EXISTS message;
