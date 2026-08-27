package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshTokenRow represents a refresh_tokens row from the database.
type RefreshTokenRow struct {
	ID        pgtype.UUID
	UserID    pgtype.UUID
	TokenHash string
	ExpiresAt pgtype.Timestamptz
	CreatedAt pgtype.Timestamptz
	RevokedAt pgtype.Timestamptz
}

// RefreshTokenRepository handles refresh token persistence.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository creates a new refresh token repository.
func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

// Create stores a new hashed refresh token.
func (r *RefreshTokenRepository) Create(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*RefreshTokenRow, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at`,
		userID, tokenHash, expiresAt,
	)

	rt := &RefreshTokenRow{}
	err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// GetByHash retrieves a valid (non-revoked, non-expired) refresh token by its hash.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshTokenRow, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		 FROM refresh_tokens
		 WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`,
		tokenHash,
	)

	rt := &RefreshTokenRow{}
	err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound // reuse — token not found
		}
		return nil, err
	}
	return rt, nil
}

// Revoke marks a refresh token as revoked.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}
