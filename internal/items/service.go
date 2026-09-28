package items

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/catalog"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/validator"
)

var (
	ErrForbidden   = errors.New("not the owner of this item")
	ErrNotEditable = errors.New("item can only be changed while available")
)

const maxImages = 10

// Service contains item business logic.
type Service struct {
	repo     *Repository
	onChange func()
}

// NewService creates a new item service.
// onChange (may be nil) is called after a item is created or updated, so the
// matching worker can look for new matches right away.
func NewService(repo *Repository, onChange func()) *Service {
	if onChange == nil {
		onChange = func() {}
	}
	return &Service{repo: repo, onChange: onChange}
}

// Create validates and stores a new item listing.
func (s *Service) Create(ctx context.Context, ownerID pgtype.UUID, req ItemRequest) (*ItemResponse, error) {
	req = normalize(req)
	if errs := validate(req); errs.HasErrors() {
		return nil, errs
	}
	it, err := s.repo.Create(ctx, ownerID, req)
	if err != nil {
		return nil, err
	}
	s.onChange()
	return toResponse(it), nil
}

// Get returns an item by ID. Withdrawn items are only visible to their owner.
func (s *Service) Get(ctx context.Context, id pgtype.UUID, viewer pgtype.UUID) (*ItemResponse, error) {
	it, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if it.Status == StatusWithdrawn && it.OwnerID != viewer {
		return nil, ErrItemNotFound
	}
	return toResponse(it), nil
}

// Browse lists available items for the public marketplace.
func (s *Service) Browse(ctx context.Context, f Filter, page request.Page) ([]ItemResponse, error) {
	f.Status = StatusAvailable
	f.Category = strings.ToLower(strings.TrimSpace(f.Category))
	f.Query = strings.TrimSpace(f.Query)
	return s.list(ctx, f, page)
}

// ListOwned lists all of a user's items, optionally filtered by status.
func (s *Service) ListOwned(ctx context.Context, ownerID pgtype.UUID, status string, page request.Page) ([]ItemResponse, error) {
	if status != "" && !isStatus(status) {
		return nil, validator.Errors{"status": "must be one of: available, reserved, exchanged, withdrawn"}
	}
	return s.list(ctx, Filter{OwnerID: ownerID, Status: status}, page)
}

func (s *Service) list(ctx context.Context, f Filter, page request.Page) ([]ItemResponse, error) {
	rows, err := s.repo.List(ctx, f, page)
	if err != nil {
		return nil, err
	}
	out := make([]ItemResponse, 0, len(rows))
	for _, it := range rows {
		out = append(out, *toResponse(it))
	}
	return out, nil
}

// Update replaces an item's editable fields. Only the owner may edit, and only while available.
func (s *Service) Update(ctx context.Context, id, userID pgtype.UUID, req ItemRequest) (*ItemResponse, error) {
	if err := s.checkEditable(ctx, id, userID); err != nil {
		return nil, err
	}
	req = normalize(req)
	if errs := validate(req); errs.HasErrors() {
		return nil, errs
	}
	it, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	s.onChange()
	return toResponse(it), nil
}

// Withdraw removes an item from the marketplace.
func (s *Service) Withdraw(ctx context.Context, id, userID pgtype.UUID) error {
	if err := s.checkEditable(ctx, id, userID); err != nil {
		return err
	}
	if err := s.repo.Withdraw(ctx, id); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			return ErrNotEditable // status changed concurrently
		}
		return err
	}
	return nil
}

func (s *Service) checkEditable(ctx context.Context, id, userID pgtype.UUID) error {
	it, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if it.OwnerID != userID {
		return ErrForbidden
	}
	if it.Status != StatusAvailable {
		return ErrNotEditable
	}
	return nil
}

// --- helpers ---

func normalize(req ItemRequest) ItemRequest {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.ToLower(strings.TrimSpace(req.Category))
	req.Condition = strings.ToLower(strings.TrimSpace(req.Condition))
	if req.Location != nil {
		loc := strings.TrimSpace(*req.Location)
		req.Location = &loc
		if loc == "" {
			req.Location = nil
		}
	}
	if req.ImageURLs == nil {
		req.ImageURLs = []string{}
	}
	return req
}

func validate(req ItemRequest) validator.Errors {
	errs := make(validator.Errors)
	validator.ValidateRequired(errs, "title", req.Title)
	validator.ValidateMaxLength(errs, "title", req.Title, 120)
	validator.ValidateMaxLength(errs, "description", req.Description, 5000)
	validator.ValidateOneOf(errs, "category", req.Category, catalog.Categories)
	validator.ValidateOneOf(errs, "condition", req.Condition, catalog.Conditions)
	if req.EstimatedValue != nil && *req.EstimatedValue < 0 {
		errs["estimated_value"] = "must not be negative"
	}
	if len(req.ImageURLs) > maxImages {
		errs["image_urls"] = "must contain at most 10 images"
	}
	for _, u := range req.ImageURLs {
		parsed, err := url.ParseRequestURI(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			errs["image_urls"] = "must contain valid http(s) URLs"
			break
		}
	}
	return errs
}

func isStatus(s string) bool {
	switch s {
	case StatusAvailable, StatusReserved, StatusExchanged, StatusWithdrawn:
		return true
	}
	return false
}

func toResponse(it *Row) *ItemResponse {
	resp := &ItemResponse{
		ID:          it.ID.String(),
		OwnerID:     it.OwnerID.String(),
		Title:       it.Title,
		Description: it.Description,
		Category:    it.Category,
		Condition:   it.Condition,
		ImageURLs:   it.ImageURLs,
		Status:      it.Status,
		CreatedAt:   it.CreatedAt.Time,
		UpdatedAt:   it.UpdatedAt.Time,
	}
	if it.EstimatedValue.Valid {
		v := int(it.EstimatedValue.Int32)
		resp.EstimatedValue = &v
	}
	if it.Location.Valid {
		resp.Location = &it.Location.String
	}
	if resp.ImageURLs == nil {
		resp.ImageURLs = []string{}
	}
	return resp
}
