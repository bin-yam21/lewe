CREATE TABLE wants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      TEXT NOT NULL,
    keywords      TEXT[] NOT NULL DEFAULT '{}',
    min_condition TEXT CHECK (min_condition IN ('new', 'like_new', 'good', 'fair', 'poor')),
    status        TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'fulfilled', 'cancelled')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wants_user_id ON wants (user_id);
CREATE INDEX idx_wants_category_status ON wants (category, status);
