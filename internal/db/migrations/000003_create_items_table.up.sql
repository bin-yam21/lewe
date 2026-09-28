-- Orders item conditions from worst to best so wants can express a minimum.
CREATE OR REPLACE FUNCTION item_condition_rank(c TEXT) RETURNS INT
LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE c
        WHEN 'poor'     THEN 1
        WHEN 'fair'     THEN 2
        WHEN 'good'     THEN 3
        WHEN 'like_new' THEN 4
        WHEN 'new'      THEN 5
        ELSE 0
    END
$$;

CREATE TABLE items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL,
    condition       TEXT NOT NULL CHECK (condition IN ('new', 'like_new', 'good', 'fair', 'poor')),
    estimated_value INTEGER CHECK (estimated_value >= 0),
    location        TEXT,
    image_urls      TEXT[] NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'available'
                    CHECK (status IN ('available', 'reserved', 'exchanged', 'withdrawn')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_items_owner_id ON items (owner_id);
CREATE INDEX idx_items_category_status ON items (category, status);
