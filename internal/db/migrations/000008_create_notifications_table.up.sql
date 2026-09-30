CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    match_id   UUID REFERENCES matches(id) ON DELETE CASCADE,
    read_at    TIMESTAMPTZ,
    -- clock_timestamp() (not now()) so notifications created in one transaction
    -- keep their insertion order when sorted by time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id) WHERE read_at IS NULL;
