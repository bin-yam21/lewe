CREATE TABLE ratings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    rater_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ratee_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score       INTEGER NOT NULL CHECK (score >= 1 AND score <= 5),
    comment     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (match_id, rater_id)
);

CREATE INDEX idx_ratings_ratee_id ON ratings (ratee_id);
