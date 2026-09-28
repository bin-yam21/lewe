package matches

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

// Handler handles HTTP requests for the matches domain. All routes require auth.
type Handler struct {
	svc *Service
}

// NewHandler creates a new match handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/matches?status=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	page := request.Pagination(r)

	matches, err := h.svc.List(r.Context(), userID, r.URL.Query().Get("status"), page)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[MatchResponse]{Data: matches, Limit: page.Limit, Offset: page.Offset})
}

// Get handles GET /api/v1/matches/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Get)
}

// Accept handles POST /api/v1/matches/{id}/accept
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Accept)
}

// Decline handles POST /api/v1/matches/{id}/decline
func (h *Handler) Decline(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Decline)
}

// Cancel handles POST /api/v1/matches/{id}/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Cancel)
}

// Complete handles POST /api/v1/matches/{id}/complete
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Complete)
}

// SetExchange handles PUT /api/v1/matches/{id}/exchange
func (h *Handler) SetExchange(w http.ResponseWriter, r *http.Request) {
	var req ExchangeRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	h.action(w, r, func(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error) {
		return h.svc.SetExchange(ctx, id, userID, req)
	})
}

// action runs a service call for the match in the {id} path segment.
func (h *Handler) action(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error)) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrMatchNotFound)
		return
	}

	match, err := fn(r.Context(), id, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, match)
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
	case errors.Is(err, ErrInvalidState):
		response.Error(w, http.StatusConflict, "This action is not allowed in the match's current status")
	case errors.Is(err, ErrItemUnavailable):
		response.Error(w, http.StatusConflict, "One of the items is no longer available; the match was cancelled")
	case errors.Is(err, ErrExchangeNotSet):
		response.Error(w, http.StatusConflict, "Choose an exchange method before completing the match")
	default:
		response.InternalError(w, err)
	}
}
