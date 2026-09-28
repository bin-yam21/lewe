// Package wants lets users describe what they are looking for in exchange.
// The matching worker pairs active wants with other users' available items.
package wants

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/catalog"
	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/validator"
)

// Want statuses.
const (
	StatusActive    = "active"
	StatusFulfilled = "fulfilled"
	StatusCancelled = "cancelled"
)

const maxKeywords = 10

var (
	ErrWantNotFound = errors.New("want not found")
	ErrNotActive    = errors.New("want is no longer active")
)

// WantRequest is the payload for creating or replacing a want.
type WantRequest struct {
	Category     string   `json:"category"`
	Keywords     []string `json:"keywords"`
	MinCondition *string  `json:"min_condition"`
}

// WantResponse is the representation of a want returned to its owner.
type WantResponse struct {
	ID           string    `json:"id"`
	Category     string    `json:"category"`
	Keywords     []string  `json:"keywords"`
	MinCondition *string   `json:"min_condition,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// --- repository ---

// Row represents a wants row from the database.
type Row struct {
	ID           pgtype.UUID
	UserID       pgtype.UUID
	Category     string
	Keywords     []string
	MinCondition pgtype.Text
	Status       string
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
}

const wantColumns = `id, user_id, category, keywords, min_condition, status, created_at, updated_at`

func scanWant(row pgx.Row) (*Row, error) {
	w := &Row{}
	err := row.Scan(&w.ID, &w.UserID, &w.Category, &w.Keywords, &w.MinCondition, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWantNotFound
		}
		return nil, err
	}
	return w, nil
}

// Repository handles want persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new want repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new active want.
func (r *Repository) Create(ctx context.Context, userID pgtype.UUID, req WantRequest) (*Row, error) {
	return scanWant(r.pool.QueryRow(ctx,
		`INSERT INTO wants (user_id, category, keywords, min_condition)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+wantColumns,
		userID, req.Category, req.Keywords, req.MinCondition,
	))
}

// GetForUser retrieves a want owned by userID.
func (r *Repository) GetForUser(ctx context.Context, id, userID pgtype.UUID) (*Row, error) {
	return scanWant(r.pool.QueryRow(ctx,
		`SELECT `+wantColumns+` FROM wants WHERE id = $1 AND user_id = $2`, id, userID))
}

// ListForUser lists a user's wants, optionally filtered by status.
func (r *Repository) ListForUser(ctx context.Context, userID pgtype.UUID, status string, page request.Page) ([]*Row, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+wantColumns+` FROM wants
		 WHERE user_id = $1 AND ($2 = '' OR status = $2)
		 ORDER BY created_at DESC, id
		 LIMIT $3 OFFSET $4`,
		userID, status, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*Row{}
	for rows.Next() {
		w, err := scanWant(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, rows.Err()
}

// Update replaces an active want's criteria.
func (r *Repository) Update(ctx context.Context, id, userID pgtype.UUID, req WantRequest) (*Row, error) {
	return scanWant(r.pool.QueryRow(ctx,
		`UPDATE wants SET category = $3, keywords = $4, min_condition = $5, updated_at = now()
		 WHERE id = $1 AND user_id = $2 AND status = 'active'
		 RETURNING `+wantColumns,
		id, userID, req.Category, req.Keywords, req.MinCondition,
	))
}

// Cancel marks an active want as cancelled and cancels pending matches created from it.
func (r *Repository) Cancel(ctx context.Context, id, userID pgtype.UUID) error {
	return db.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE wants SET status = 'cancelled', updated_at = now()
			 WHERE id = $1 AND user_id = $2 AND status = 'active'`, id, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrWantNotFound
		}
		_, err = tx.Exec(ctx,
			`UPDATE matches SET status = 'cancelled', updated_at = now()
			 WHERE status = 'pending' AND (want_a_id = $1 OR want_b_id = $1)`, id)
		return err
	})
}

// --- service ---

// Service contains want business logic.
type Service struct {
	repo     *Repository
	onChange func()
}

// NewService creates a new want service.
// onChange (may be nil) is called after a want is created or updated, so the
// matching worker can look for new matches right away.
func NewService(repo *Repository, onChange func()) *Service {
	if onChange == nil {
		onChange = func() {}
	}
	return &Service{repo: repo, onChange: onChange}
}

// Create validates and stores a new want.
func (s *Service) Create(ctx context.Context, userID pgtype.UUID, req WantRequest) (*WantResponse, error) {
	req = normalize(req)
	if errs := validate(req); errs.HasErrors() {
		return nil, errs
	}
	w, err := s.repo.Create(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	s.onChange()
	return toResponse(w), nil
}

// Get returns one of the user's wants.
func (s *Service) Get(ctx context.Context, id, userID pgtype.UUID) (*WantResponse, error) {
	w, err := s.repo.GetForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return toResponse(w), nil
}

// List returns the user's wants.
func (s *Service) List(ctx context.Context, userID pgtype.UUID, status string, page request.Page) ([]WantResponse, error) {
	if status != "" && status != StatusActive && status != StatusFulfilled && status != StatusCancelled {
		return nil, validator.Errors{"status": "must be one of: active, fulfilled, cancelled"}
	}
	rows, err := s.repo.ListForUser(ctx, userID, status, page)
	if err != nil {
		return nil, err
	}
	out := make([]WantResponse, 0, len(rows))
	for _, w := range rows {
		out = append(out, *toResponse(w))
	}
	return out, nil
}

// Update replaces an active want's criteria.
func (s *Service) Update(ctx context.Context, id, userID pgtype.UUID, req WantRequest) (*WantResponse, error) {
	existing, err := s.repo.GetForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusActive {
		return nil, ErrNotActive
	}
	req = normalize(req)
	if errs := validate(req); errs.HasErrors() {
		return nil, errs
	}
	w, err := s.repo.Update(ctx, id, userID, req)
	if err != nil {
		if errors.Is(err, ErrWantNotFound) {
			return nil, ErrNotActive
		}
		return nil, err
	}
	s.onChange()
	return toResponse(w), nil
}

// Cancel deactivates a want.
func (s *Service) Cancel(ctx context.Context, id, userID pgtype.UUID) error {
	existing, err := s.repo.GetForUser(ctx, id, userID)
	if err != nil {
		return err
	}
	if existing.Status != StatusActive {
		return ErrNotActive
	}
	if err := s.repo.Cancel(ctx, id, userID); err != nil {
		if errors.Is(err, ErrWantNotFound) {
			return ErrNotActive
		}
		return err
	}
	return nil
}

func normalize(req WantRequest) WantRequest {
	req.Category = strings.ToLower(strings.TrimSpace(req.Category))
	if req.MinCondition != nil {
		c := strings.ToLower(strings.TrimSpace(*req.MinCondition))
		req.MinCondition = &c
		if c == "" {
			req.MinCondition = nil
		}
	}

	// Keywords are matched as case-insensitive substrings; strip LIKE wildcards
	// and duplicates so they behave as plain words.
	seen := map[string]bool{}
	keywords := []string{}
	for _, k := range req.Keywords {
		k = strings.ToLower(strings.TrimSpace(strings.NewReplacer("%", "", "_", "", `\`, "").Replace(k)))
		if k != "" && !seen[k] {
			seen[k] = true
			keywords = append(keywords, k)
		}
	}
	req.Keywords = keywords
	return req
}

func validate(req WantRequest) validator.Errors {
	errs := make(validator.Errors)
	validator.ValidateOneOf(errs, "category", req.Category, catalog.Categories)
	if req.MinCondition != nil {
		validator.ValidateOneOf(errs, "min_condition", *req.MinCondition, catalog.Conditions)
	}
	if len(req.Keywords) > maxKeywords {
		errs["keywords"] = "must contain at most 10 keywords"
	}
	for _, k := range req.Keywords {
		if len(k) > 50 {
			errs["keywords"] = "each keyword must be at most 50 characters"
			break
		}
	}
	return errs
}

func toResponse(w *Row) *WantResponse {
	resp := &WantResponse{
		ID:        w.ID.String(),
		Category:  w.Category,
		Keywords:  w.Keywords,
		Status:    w.Status,
		CreatedAt: w.CreatedAt.Time,
		UpdatedAt: w.UpdatedAt.Time,
	}
	if w.MinCondition.Valid {
		resp.MinCondition = &w.MinCondition.String
	}
	if resp.Keywords == nil {
		resp.Keywords = []string{}
	}
	return resp
}
