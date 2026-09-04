package matching

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

// Handler handles HTTP requests for the matching domain.
type Handler struct {
	svc *Service
}

// NewHandler creates a new matching handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// FindMatches handles POST /api/v1/items/{id}/matches
func (h *Handler) FindMatches(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.FindMatchesForItem(r.Context(), userID, itemID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ListMine handles GET /api/v1/matches
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	result, err := h.svc.ListMyMatches(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Get handles GET /api/v1/matches/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	matchID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid match ID")
		return
	}

	result, err := h.svc.GetMatch(r.Context(), userID, matchID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Respond handles POST /api/v1/matches/{id}/respond
func (h *Handler) Respond(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	matchID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid match ID")
		return
	}

	var req RespondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.svc.RespondToMatch(r.Context(), userID, matchID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Complete handles POST /api/v1/matches/{id}/complete
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	matchID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid match ID")
		return
	}

	result, err := h.svc.CompleteMatch(r.Context(), userID, matchID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Cancel handles POST /api/v1/matches/{id}/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	matchID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid match ID")
		return
	}

	result, err := h.svc.CancelMatch(r.Context(), userID, matchID)
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
	case errors.Is(err, ErrMatchNotFound):
		response.Error(w, http.StatusNotFound, "Match not found")
	case errors.Is(err, ErrNotMatchParty):
		response.Error(w, http.StatusForbidden, "You are not a participant in this match")
	case errors.Is(err, ErrInvalidTransition):
		response.Error(w, http.StatusConflict, "Invalid status transition")
	case errors.Is(err, items.ErrItemNotFound):
		response.Error(w, http.StatusNotFound, "Item not found")
	case errors.Is(err, items.ErrNotItemOwner):
		response.Error(w, http.StatusForbidden, "You are not the owner of this item")
	default:
		response.Error(w, http.StatusInternalServerError, "Internal server error")
	}
}

// parsePathUUID extracts a UUID path parameter from the request.
func parsePathUUID(r *http.Request, name string) (pgtype.UUID, error) {
	raw := r.PathValue(name)
	var id pgtype.UUID
	err := id.Scan(raw)
	return id, err
}
