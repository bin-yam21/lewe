CREATE TABLE matches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_a_id       UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    item_b_id       UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted_a','accepted_b','confirmed','completed','cancelled')),
    exchange_method TEXT CHECK (exchange_method IN ('in_person','shipping')),
    confirmed_a     BOOLEAN NOT NULL DEFAULT false,
    confirmed_b     BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_a_id, item_b_id)
);

CREATE INDEX idx_matches_item_a ON matches (item_a_id);
CREATE INDEX idx_matches_item_b ON matches (item_b_id);
CREATE INDEX idx_matches_status ON matches (status);
