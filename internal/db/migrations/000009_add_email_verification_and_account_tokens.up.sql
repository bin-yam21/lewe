ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

-- One-time tokens emailed to users: email verification and password reset.
-- Only the SHA-256 hash is stored; used_at makes each token single-use.
CREATE TABLE account_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_account_tokens_user_purpose ON account_tokens (user_id, purpose);
