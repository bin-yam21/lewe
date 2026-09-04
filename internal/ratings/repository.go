package ratings

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRatingNotFound  = errors.New("rating not found")
	ErrAlreadyRated    = errors.New("already rated this exchange")
	ErrDuplicateRating = errors.New("duplicate rating")
)

// RatingRow represents a rating row from the database.
type RatingRow struct {
	ID        pgtype.UUID
	MatchID   pgtype.UUID
	RaterID   pgtype.UUID
	RateeID   pgtype.UUID
	Score     int
	Comment   pgtype.Text
	CreatedAt pgtype.Timestamptz
}

// Repository handles rating persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new ratings repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new rating.
func (r *Repository) Create(ctx context.Context, matchID, raterID, rateeID pgtype.UUID, score int, comment *string) (*RatingRow, error) {
	row := &RatingRow{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO ratings (match_id, rater_id, ratee_id, score, comment)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, match_id, rater_id, ratee_id, score, comment, created_at`,
		matchID, raterID, rateeID, score, comment,
	).Scan(
		&row.ID, &row.MatchID, &row.RaterID, &row.RateeID,
		&row.Score, &row.Comment, &row.CreatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrAlreadyRated
		}
		return nil, err
	}
	return row, nil
}

// GetByMatch retrieves all ratings for a given match.
func (r *Repository) GetByMatch(ctx context.Context, matchID pgtype.UUID) ([]RatingRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, match_id, rater_id, ratee_id, score, comment, created_at
		 FROM ratings WHERE match_id = $1 ORDER BY created_at`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []RatingRow
	for rows.Next() {
		var rr RatingRow
		if err := rows.Scan(&rr.ID, &rr.MatchID, &rr.RaterID, &rr.RateeID, &rr.Score, &rr.Comment, &rr.CreatedAt); err != nil {
			return nil, err
		}
		ratings = append(ratings, rr)
	}
	return ratings, nil
}

// GetUserAverage returns the average score and total rating count for a user.
func (r *Repository) GetUserAverage(ctx context.Context, userID pgtype.UUID) (float64, int, error) {
	var avg float64
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(score)::float, 0), COUNT(*) FROM ratings WHERE ratee_id = $1`,
		userID,
	).Scan(&avg, &count)
	if err != nil {
		return 0, 0, err
	}
	return avg, count, nil
}

// ExistsForMatchAndRater checks if a rating already exists for a given match and rater.
func (r *Repository) ExistsForMatchAndRater(ctx context.Context, matchID, raterID pgtype.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ratings WHERE match_id = $1 AND rater_id = $2)`,
		matchID, raterID,
	).Scan(&exists)
	return exists, err
}

// --- helpers ---

// isDuplicateKeyError checks if a pgx error is a unique constraint violation (code 23505).
func isDuplicateKeyError(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

// uuidToString converts a pgtype.UUID to its string representation.
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	b := id.Bytes
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}

// parseUUID parses a string into a pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(s)
	return id, err
}

// Suppress unused import warning
var _ = pgx.ErrNoRows
