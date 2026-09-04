CREATE TABLE items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    category        TEXT NOT NULL,
    condition       TEXT NOT NULL CHECK (condition IN ('new','like_new','good','fair','poor')),
    exchange_method TEXT NOT NULL DEFAULT 'either' CHECK (exchange_method IN ('in_person','shipping','either')),
    images          JSONB NOT NULL DEFAULT '[]',
    location        TEXT,
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','matched','exchanged','archived')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_items_user_id ON items (user_id);
CREATE INDEX idx_items_status ON items (status);
CREATE INDEX idx_items_category ON items (category);
