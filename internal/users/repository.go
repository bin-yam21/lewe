package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already registered")
)

// Repository handles user persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new user repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// UserRow represents a user row from the database.
type UserRow struct {
	ID           pgtype.UUID
	Email        string
	PasswordHash string
	FullName     string
	Phone        pgtype.Text
	Location     pgtype.Text
	Bio          pgtype.Text
	AvatarURL    pgtype.Text
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
}

// Create inserts a new user into the database.
func (r *Repository) Create(ctx context.Context, email, passwordHash, fullName string, phone, location, bio *string) (*UserRow, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, full_name, phone, location, bio)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, email, password_hash, full_name, phone, location, bio, avatar_url, created_at, updated_at`,
		email, passwordHash, fullName, phone, location, bio,
	)

	user := &UserRow{}
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Phone, &user.Location, &user.Bio, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation on email
		if isDuplicateKeyError(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a user by their UUID.
func (r *Repository) GetByID(ctx context.Context, id pgtype.UUID) (*UserRow, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, full_name, phone, location, bio, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	)

	user := &UserRow{}
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Phone, &user.Location, &user.Bio, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*UserRow, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, full_name, phone, location, bio, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	)

	user := &UserRow{}
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Phone, &user.Location, &user.Bio, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// Update modifies a user's profile fields.
func (r *Repository) Update(ctx context.Context, id pgtype.UUID, fullName string, phone, location, bio, avatarURL *string) (*UserRow, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE users
		 SET full_name = $2, phone = $3, location = $4, bio = $5, avatar_url = $6, updated_at = now()
		 WHERE id = $1
		 RETURNING id, email, password_hash, full_name, phone, location, bio, avatar_url, created_at, updated_at`,
		id, fullName, phone, location, bio, avatarURL,
	)

	user := &UserRow{}
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Phone, &user.Location, &user.Bio, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// isDuplicateKeyError checks if a pgx error is a unique constraint violation (code 23505).
func isDuplicateKeyError(err error) bool {
	// pgx wraps errors; check the error message for the PG error code
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
