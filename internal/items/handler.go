package items

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

// Handler handles HTTP requests for the items domain.
type Handler struct {
	svc *Service
}

// NewHandler creates a new item handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /api/v1/items
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	item, err := h.svc.CreateItem(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, item)
}

// Get handles GET /api/v1/items/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	itemID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	item, err := h.svc.GetItem(r.Context(), itemID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, item)
}

// Update handles PUT /api/v1/items/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	item, err := h.svc.UpdateItem(r.Context(), userID, itemID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, item)
}

// Delete handles DELETE /api/v1/items/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	if err := h.svc.DeleteItem(r.Context(), userID, itemID); err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Item archived"})
}

// List handles GET /api/v1/items
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	filter := ListItemsFilter{
		Category: q.Get("category"),
		Status:   q.Get("status"),
		Query:    q.Get("q"),
		Page:     page,
		PerPage:  perPage,
	}

	result, err := h.svc.ListItems(r.Context(), filter)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ListMine handles GET /api/v1/users/me/items
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	filter := ListItemsFilter{
		UserID:   uuidToString(userID),
		Category: q.Get("category"),
		Status:   q.Get("status"),
		Query:    q.Get("q"),
		Page:     page,
		PerPage:  perPage,
		// Owners may see and filter their own archived (soft-deleted) items.
		AllowArchived: true,
	}

	result, err := h.svc.ListItems(r.Context(), filter)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// handleError maps domain errors to HTTP status codes.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var validationErrs validator.Errors
	if errors.As(err, &validationErrs) {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "Validation failed",
			"fields": validationErrs,
		})
		return
	}

	switch {
	case errors.Is(err, ErrItemNotFound):
		response.Error(w, http.StatusNotFound, "Item not found")
	case errors.Is(err, ErrNotItemOwner):
		response.Error(w, http.StatusForbidden, "You are not the owner of this item")
	default:
		response.Error(w, http.StatusInternalServerError, "Internal server error")
	}
}

// parsePathUUID extracts a UUID path parameter from the request.
// Go 1.22+ stdlib routing captures path params via r.PathValue.
func parsePathUUID(r *http.Request, name string) (pgtype.UUID, error) {
	raw := r.PathValue(name)
	var id pgtype.UUID
	err := id.Scan(raw)
	return id, err
}
