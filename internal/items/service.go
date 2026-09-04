package items

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/validator"
)

var (
	allowedConditions      = []string{"new", "like_new", "good", "fair", "poor"}
	allowedExchangeMethods = []string{"in_person", "shipping", "either"}

	// publicStatuses are the item statuses any caller may filter by.
	// "archived" is deliberately excluded — archived items are soft-deleted
	// and only their owner may list them.
	publicStatuses = []string{"active", "matched", "exchanged"}
	ownerStatuses  = []string{"active", "matched", "exchanged", "archived"}
)

// Service contains item business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new item service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateItem validates input and creates a new item listing.
func (s *Service) CreateItem(ctx context.Context, userID pgtype.UUID, req CreateItemRequest) (*ItemResponse, error) {
	errs := make(validator.Errors)
	validator.ValidateRequired(errs, "title", req.Title)
	validator.ValidateMaxLength(errs, "title", req.Title, 200)
	validator.ValidateRequired(errs, "description", req.Description)
	validator.ValidateRequired(errs, "category", req.Category)
	validator.ValidateRequired(errs, "condition", req.Condition)
	validator.ValidateOneOf(errs, "condition", req.Condition, allowedConditions)
	if req.ExchangeMethod == "" {
		req.ExchangeMethod = "either"
	}
	validator.ValidateOneOf(errs, "exchange_method", req.ExchangeMethod, allowedExchangeMethods)

	if len(req.Wants) == 0 {
		errs["wants"] = "at least one want is required"
	}
	for i, w := range req.Wants {
		if w.Category == "" {
			errs[wantFieldKey(i, "category")] = "is required"
		}
	}

	if errs.HasErrors() {
		return nil, errs
	}

	if req.Images == nil {
		req.Images = []string{}
	}

	item, wants, err := s.repo.Create(ctx, userID, req.Title, req.Description, req.Category, req.Condition, req.ExchangeMethod, req.Images, req.Location, req.Wants)
	if err != nil {
		return nil, err
	}

	resp := toItemResponse(item, wants)
	return &resp, nil
}

// GetItem retrieves an item by ID.
func (s *Service) GetItem(ctx context.Context, itemID pgtype.UUID) (*ItemResponse, error) {
	item, wants, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	resp := toItemResponse(item, wants)
	return &resp, nil
}

// UpdateItem validates and updates an item. Only the owner can update.
func (s *Service) UpdateItem(ctx context.Context, userID, itemID pgtype.UUID, req UpdateItemRequest) (*ItemResponse, error) {
	errs := make(validator.Errors)

	if req.Title != nil {
		validator.ValidateRequired(errs, "title", *req.Title)
		validator.ValidateMaxLength(errs, "title", *req.Title, 200)
	}
	if req.Description != nil {
		validator.ValidateRequired(errs, "description", *req.Description)
	}
	if req.Category != nil {
		validator.ValidateRequired(errs, "category", *req.Category)
	}
	if req.Condition != nil {
		validator.ValidateOneOf(errs, "condition", *req.Condition, allowedConditions)
	}
	if req.ExchangeMethod != nil {
		validator.ValidateOneOf(errs, "exchange_method", *req.ExchangeMethod, allowedExchangeMethods)
	}
	if req.Wants != nil {
		if len(*req.Wants) == 0 {
			errs["wants"] = "at least one want is required"
		}
		for i, w := range *req.Wants {
			if w.Category == "" {
				errs[wantFieldKey(i, "category")] = "is required"
			}
		}
	}

	if errs.HasErrors() {
		return nil, errs
	}

	item, wants, err := s.repo.Update(ctx, itemID, userID, req.Title, req.Description, req.Category, req.Condition, req.ExchangeMethod, req.Images, req.Location, req.Wants)
	if err != nil {
		return nil, err
	}

	resp := toItemResponse(item, wants)
	return &resp, nil
}

// DeleteItem archives an item. Only the owner can delete.
func (s *Service) DeleteItem(ctx context.Context, userID, itemID pgtype.UUID) error {
	return s.repo.Delete(ctx, itemID, userID)
}

// ListItems retrieves a paginated, filtered list of items.
func (s *Service) ListItems(ctx context.Context, f ListItemsFilter) (*ItemListResponse, error) {
	if f.Status != "" {
		allowed := publicStatuses
		if f.AllowArchived {
			allowed = ownerStatuses
		}
		errs := make(validator.Errors)
		validator.ValidateOneOf(errs, "status", f.Status, allowed)
		if errs.HasErrors() {
			return nil, errs
		}
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 50 {
		f.PerPage = 20
	}

	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]ItemResponse, 0, len(rows))
	for _, row := range rows {
		// Fetch wants for each item (N+1 — acceptable at this scale)
		wants, err := s.repo.getWantsByItemID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, toItemResponse(&row, wants))
	}

	return &ItemListResponse{
		Items:   items,
		Total:   total,
		Page:    f.Page,
		PerPage: f.PerPage,
	}, nil
}

// --- helpers ---

// toItemResponse maps DB rows to the public-facing DTO.
func toItemResponse(item *ItemRow, wants []WantRow) ItemResponse {
	resp := ItemResponse{
		ID:             uuidToString(item.ID),
		UserID:         uuidToString(item.UserID),
		Title:          item.Title,
		Description:    item.Description,
		Category:       item.Category,
		Condition:      item.Condition,
		ExchangeMethod: item.ExchangeMethod,
		Status:         item.Status,
		Images:         []string{},
		Wants:          make([]WantResponse, 0, len(wants)),
	}

	// Parse images JSON
	if item.Images != nil {
		_ = json.Unmarshal(item.Images, &resp.Images)
	}
	if resp.Images == nil {
		resp.Images = []string{}
	}

	if item.Location.Valid {
		resp.Location = &item.Location.String
	}
	if item.CreatedAt.Valid {
		resp.CreatedAt = item.CreatedAt.Time
	}
	if item.UpdatedAt.Valid {
		resp.UpdatedAt = item.UpdatedAt.Time
	}

	for _, w := range wants {
		wr := WantResponse{
			ID:       uuidToString(w.ID),
			Category: w.Category,
		}
		if w.Description.Valid {
			wr.Description = &w.Description.String
		}
		resp.Wants = append(resp.Wants, wr)
	}

	return resp
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

// wantFieldKey returns a validation error key like "wants[0].category".
func wantFieldKey(index int, field string) string {
	return "wants[" + strconv.Itoa(index) + "]." + field
}
