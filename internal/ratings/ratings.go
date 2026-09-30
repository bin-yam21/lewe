// Package ratings lets the two users of a completed match rate each other.
package ratings

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

var (
	ErrMatchNotFound    = errors.New("match not found")
	ErrMatchNotComplete = errors.New("match is not completed")
	ErrAlreadyRated     = errors.New("already rated")
)

// RatingRequest is the payload for rating the other participant of a match.
type RatingRequest struct {
	Score   int     `json:"score"`
	Comment *string `json:"comment"`
}

// RatingResponse is a rating as shown on a user's profile.
type RatingResponse struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"match_id"`
	Rater     Rater     `json:"rater"`
	Score     int       `json:"score"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Rater identifies who left a rating.
type Rater struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
}

// --- repository ---

// Repository handles rating persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new rating repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// matchParticipants returns the match's two users and status.
func (r *Repository) matchParticipants(ctx context.Context, q db.DBTX, matchID pgtype.UUID) (a, b pgtype.UUID, status string, err error) {
	err = q.QueryRow(ctx, `SELECT user_a_id, user_b_id, status FROM matches WHERE id = $1`, matchID).
		Scan(&a, &b, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrMatchNotFound
	}
	return a, b, status, err
}

const ratingSelect = `SELECT r.id, r.match_id, r.rater_id, u.full_name, r.score, r.comment, r.created_at
	FROM ratings r JOIN users u ON u.id = r.rater_id`

func scanRating(row pgx.Row) (*RatingResponse, error) {
	var (
		id, matchID, raterID pgtype.UUID
		score                int16
		comment              pgtype.Text
		createdAt            pgtype.Timestamptz
		resp                 RatingResponse
	)
	if err := row.Scan(&id, &matchID, &raterID, &resp.Rater.FullName, &score, &comment, &createdAt); err != nil {
		return nil, err
	}
	resp.ID, resp.MatchID, resp.Rater.ID = id.String(), matchID.String(), raterID.String()
	resp.Score = int(score)
	resp.CreatedAt = createdAt.Time
	if comment.Valid {
		resp.Comment = &comment.String
	}
	return &resp, nil
}

// Create stores a rating.
func (r *Repository) Create(ctx context.Context, matchID, raterID, rateeID pgtype.UUID, req RatingRequest) (*RatingResponse, error) {
	var id pgtype.UUID
	err := db.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO ratings (match_id, rater_id, ratee_id, score, comment)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			matchID, raterID, rateeID, req.Score, req.Comment).Scan(&id)
		if err != nil {
			return err
		}
		return notifications.Create(ctx, tx, rateeID, notifications.TypeRatingReceived, matchID)
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrAlreadyRated
		}
		return nil, err
	}
	return scanRating(r.pool.QueryRow(ctx, ratingSelect+` WHERE r.id = $1`, id))
}

// ListForUser lists ratings received by a user, newest first.
func (r *Repository) ListForUser(ctx context.Context, userID pgtype.UUID, page request.Page) ([]RatingResponse, error) {
	rows, err := r.pool.Query(ctx,
		ratingSelect+` WHERE r.ratee_id = $1 ORDER BY r.created_at DESC, r.id LIMIT $2 OFFSET $3`,
		userID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RatingResponse{}
	for rows.Next() {
		rt, err := scanRating(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rt)
	}
	return out, rows.Err()
}

// --- service ---

// Service contains rating business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new rating service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Rate records userID's rating of the other participant in a completed match.
func (s *Service) Rate(ctx context.Context, matchID, userID pgtype.UUID, req RatingRequest) (*RatingResponse, error) {
	if req.Comment != nil {
		c := strings.TrimSpace(*req.Comment)
		req.Comment = &c
		if c == "" {
			req.Comment = nil
		}
	}
	errs := make(validator.Errors)
	validator.ValidateRange(errs, "score", req.Score, 1, 5)
	if req.Comment != nil {
		validator.ValidateMaxLength(errs, "comment", *req.Comment, 1000)
	}
	if errs.HasErrors() {
		return nil, errs
	}

	a, b, status, err := s.repo.matchParticipants(ctx, s.repo.pool, matchID)
	if err != nil {
		return nil, err
	}
	var ratee pgtype.UUID
	switch userID {
	case a:
		ratee = b
	case b:
		ratee = a
	default:
		return nil, ErrMatchNotFound
	}
	if status != "completed" {
		return nil, ErrMatchNotComplete
	}
	return s.repo.Create(ctx, matchID, userID, ratee, req)
}

// ListForUser lists ratings a user has received.
func (s *Service) ListForUser(ctx context.Context, userID pgtype.UUID, page request.Page) ([]RatingResponse, error) {
	return s.repo.ListForUser(ctx, userID, page)
}

// --- handler ---

// Handler handles HTTP requests for ratings.
type Handler struct {
	svc *Service
}

// NewHandler creates a new rating handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Rate handles POST /api/v1/matches/{id}/rating
func (h *Handler) Rate(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	matchID, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrMatchNotFound)
		return
	}

	var req RatingRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rating, err := h.svc.Rate(r.Context(), matchID, userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, rating)
}

// ListForUser handles GET /api/v1/users/{id}/ratings
func (h *Handler) ListForUser(w http.ResponseWriter, r *http.Request) {
	userID, err := request.PathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	page := request.Pagination(r)
	ratings, err := h.svc.ListForUser(r.Context(), userID, page)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[RatingResponse]{Data: ratings, Limit: page.Limit, Offset: page.Offset})
}

// handleError maps domain errors to HTTP status codes.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var validationErrs validator.Errors
	if errors.As(err, &validationErrs) {
		response.ValidationError(w, validationErrs)
		return
	}

	switch {
	case errors.Is(err, ErrMatchNotFound):
		response.Error(w, http.StatusNotFound, "Match not found")
	case errors.Is(err, ErrMatchNotComplete):
		response.Error(w, http.StatusConflict, "Only completed matches can be rated")
	case errors.Is(err, ErrAlreadyRated):
		response.Error(w, http.StatusConflict, "You have already rated this match")
	default:
		response.InternalError(w, err)
	}
}
