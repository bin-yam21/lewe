package items

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/catalog"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/request"
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

// Categories handles GET /api/v1/categories
func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string][]string{
		"categories":       catalog.Categories,
		"conditions":       catalog.Conditions,
		"exchange_methods": catalog.ExchangeMethods,
	})
}

// Create handles POST /api/v1/items
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req ItemRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	item, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

// List handles GET /api/v1/items?category=&q=&owner_id=&limit=&offset=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := Filter{Category: q.Get("category"), Query: q.Get("q")}
	if owner := q.Get("owner_id"); owner != "" {
		id, err := request.ParseUUID(owner)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid owner_id")
			return
		}
		f.OwnerID = id
	}

	page := request.Pagination(r)
	items, err := h.svc.Browse(r.Context(), f, page)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[ItemResponse]{Data: items, Limit: page.Limit, Offset: page.Offset})
}

// ListMine handles GET /api/v1/users/me/items?status=
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	page := request.Pagination(r)

	items, err := h.svc.ListOwned(r.Context(), userID, r.URL.Query().Get("status"), page)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[ItemResponse]{Data: items, Limit: page.Limit, Offset: page.Offset})
}

// Get handles GET /api/v1/items/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := request.PathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusNotFound, "Item not found")
		return
	}

	// Authentication is optional here; a signed-in owner can see their withdrawn items.
	viewer, _ := middleware.UserIDFromContext(r.Context())

	item, err := h.svc.Get(r.Context(), id, viewer)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

// Update handles PUT /api/v1/items/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := h.ids(w, r)
	if !ok {
		return
	}

	var req ItemRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	item, err := h.svc.Update(r.Context(), id, userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

// Delete handles DELETE /api/v1/items/{id} (soft delete: marks the item withdrawn)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := h.ids(w, r)
	if !ok {
		return
	}

	if err := h.svc.Withdraw(r.Context(), id, userID); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ids(w http.ResponseWriter, r *http.Request) (id, userID pgtype.UUID, ok bool) {
	id, err := request.PathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusNotFound, "Item not found")
		return id, userID, false
	}
	userID, _ = middleware.UserIDFromContext(r.Context())
	return id, userID, true
}

// handleError maps domain errors to HTTP status codes.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var validationErrs validator.Errors
	if errors.As(err, &validationErrs) {
		response.ValidationError(w, validationErrs)
		return
	}

	switch {
	case errors.Is(err, ErrItemNotFound):
		response.Error(w, http.StatusNotFound, "Item not found")
	case errors.Is(err, ErrForbidden):
		response.Error(w, http.StatusForbidden, "You do not own this item")
	case errors.Is(err, ErrNotEditable):
		response.Error(w, http.StatusConflict, "Item can only be changed while it is available")
	default:
		response.InternalError(w, err)
	}
}
