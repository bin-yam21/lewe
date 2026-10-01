package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
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
	ID               pgtype.UUID
	Email            string
	PasswordHash     string
	FullName         string
	Phone            pgtype.Text
	Location         pgtype.Text
	Bio              pgtype.Text
	AvatarURL        pgtype.Text
	EmailVerifiedAt  pgtype.Timestamptz
	TelegramID       pgtype.Int8
	TelegramUsername pgtype.Text
	City             pgtype.Text
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
}

// Email and PasswordHash are empty for accounts created through Telegram.
const userColumns = `id, COALESCE(email, ''), COALESCE(password_hash, ''), full_name, phone, location, bio,
	avatar_url, email_verified_at, telegram_id, telegram_username, city, created_at, updated_at`

func scanUser(row pgx.Row) (*UserRow, error) {
	u := &UserRow{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Location, &u.Bio,
		&u.AvatarURL, &u.EmailVerifiedAt, &u.TelegramID, &u.TelegramUsername, &u.City, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// Create inserts a new user into the database.
func (r *Repository) Create(ctx context.Context, email, passwordHash, fullName string, phone, location, bio *string) (*UserRow, error) {
	user, err := scanUser(r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, full_name, phone, location, bio)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+userColumns,
		email, passwordHash, fullName, phone, location, bio,
	))
	if err != nil && db.IsUniqueViolation(err) {
		return nil, ErrEmailTaken
	}
	return user, err
}

// GetByID retrieves a user by their UUID.
func (r *Repository) GetByID(ctx context.Context, id pgtype.UUID) (*UserRow, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

// GetByEmail retrieves a user by their email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*UserRow, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

// Update modifies a user's profile fields.
func (r *Repository) Update(ctx context.Context, id pgtype.UUID, req UpdateProfileRequest) (*UserRow, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`UPDATE users
		 SET full_name = $2, phone = $3, location = $4, bio = $5, avatar_url = $6, city = $7, updated_at = now()
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, req.FullName, req.Phone, req.Location, req.Bio, req.AvatarURL, req.City,
	))
}

// UpsertTelegram finds the account linked to a Telegram user, creating it on
// first sign-in. The Telegram username is refreshed on every sign-in; the
// name and photo are only used to fill in a new account.
func (r *Repository) UpsertTelegram(ctx context.Context, telegramID int64, username, fullName, photoURL *string) (*UserRow, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`INSERT INTO users (telegram_id, telegram_username, full_name, avatar_url)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (telegram_id) DO UPDATE
		   SET telegram_username = EXCLUDED.telegram_username, updated_at = now()
		 RETURNING `+userColumns,
		telegramID, username, fullName, photoURL,
	))
}

// setPassword replaces a user's password hash.
func setPassword(ctx context.Context, q db.DBTX, id pgtype.UUID, passwordHash string) error {
	_, err := q.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	return err
}

// markEmailVerified records that the user proved they own their email address.
func markEmailVerified(ctx context.Context, q db.DBTX, id pgtype.UUID) error {
	_, err := q.Exec(ctx,
		`UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()), updated_at = now()
		 WHERE id = $1`, id)
	return err
}

// RatingSummary returns the average score and number of ratings a user has received.
func (r *Repository) RatingSummary(ctx context.Context, id pgtype.UUID) (average float64, count int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(score), 0)::float8, COUNT(*) FROM ratings WHERE ratee_id = $1`, id,
	).Scan(&average, &count)
	return average, count, err
}
