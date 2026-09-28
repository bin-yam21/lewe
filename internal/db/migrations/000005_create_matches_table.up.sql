-- A match is a two-way swap: user A gives item A (which satisfies want B)
-- and user B gives item B (which satisfies want A).
-- Rows are canonicalised with item_a_id < item_b_id so a pair is only matched once.
CREATE TABLE matches (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_a_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_a_id            UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    want_a_id            UUID REFERENCES wants(id) ON DELETE SET NULL,
    user_b_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_b_id            UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    want_b_id            UUID REFERENCES wants(id) ON DELETE SET NULL,
    score                DOUBLE PRECISION NOT NULL DEFAULT 0,
    status               TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'accepted', 'completed', 'declined', 'cancelled')),
    a_accepted_at        TIMESTAMPTZ,
    b_accepted_at        TIMESTAMPTZ,
    exchange_method      TEXT CHECK (exchange_method IN ('meetup', 'shipping', 'dropoff')),
    exchange_details     TEXT,
    exchange_proposed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    a_completed_at       TIMESTAMPTZ,
    b_completed_at       TIMESTAMPTZ,
    closed_by            UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at         TIMESTAMPTZ,
    CONSTRAINT matches_item_order CHECK (item_a_id < item_b_id),
    CONSTRAINT matches_item_pair_unique UNIQUE (item_a_id, item_b_id)
);

CREATE INDEX idx_matches_user_a ON matches (user_a_id, status);
CREATE INDEX idx_matches_user_b ON matches (user_b_id, status);
CREATE INDEX idx_matches_item_b ON matches (item_b_id);
