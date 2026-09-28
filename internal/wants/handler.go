package wants

import (
	"errors"
	"net/http"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

// Handler handles HTTP requests for the wants domain. All routes require auth.
type Handler struct {
	svc *Service
}

// NewHandler creates a new want handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /api/v1/wants
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req WantRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	want, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, want)
}

// List handles GET /api/v1/wants?status=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	page := request.Pagination(r)

	wants, err := h.svc.List(r.Context(), userID, r.URL.Query().Get("status"), page)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[WantResponse]{Data: wants, Limit: page.Limit, Offset: page.Offset})
}

// Get handles GET /api/v1/wants/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrWantNotFound)
		return
	}

	want, err := h.svc.Get(r.Context(), id, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, want)
}

// Update handles PUT /api/v1/wants/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrWantNotFound)
		return
	}

	var req WantRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	want, err := h.svc.Update(r.Context(), id, userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, want)
}

// Delete handles DELETE /api/v1/wants/{id} (marks the want cancelled)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrWantNotFound)
		return
	}

	if err := h.svc.Cancel(r.Context(), id, userID); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleError maps domain errors to HTTP status codes.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var validationErrs validator.Errors
	if errors.As(err, &validationErrs) {
		response.ValidationError(w, validationErrs)
		return
	}

	switch {
	case errors.Is(err, ErrWantNotFound):
		response.Error(w, http.StatusNotFound, "Want not found")
	case errors.Is(err, ErrNotActive):
		response.Error(w, http.StatusConflict, "Want is no longer active")
	default:
		response.InternalError(w, err)
	}
}
