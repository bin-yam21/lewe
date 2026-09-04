-- Direct offers: a trade proposed straight to an owner, rather than discovered
-- by the mutual-wants query. Same lifecycle as any other match, so it reuses
-- the matches table instead of growing a parallel one.
ALTER TABLE matches
    ADD COLUMN origin  TEXT NOT NULL DEFAULT 'discovered'
        CHECK (origin IN ('discovered', 'direct')),
    ADD COLUMN message TEXT;

CREATE INDEX idx_matches_origin ON matches (origin);

-- 'private' lets someone offer an item without publishing it to the feed.
-- The item still exists as a normal row — it just never appears in browse.
ALTER TABLE items DROP CONSTRAINT IF EXISTS items_status_check;
ALTER TABLE items ADD CONSTRAINT items_status_check
    CHECK (status IN ('active', 'matched', 'exchanged', 'archived', 'private'));
