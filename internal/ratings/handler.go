package ratings

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/matching"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

// Handler handles HTTP requests for the ratings domain.
type Handler struct {
	svc *Service
}

// NewHandler creates a new ratings handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Rate handles POST /api/v1/matches/{id}/rate
func (h *Handler) Rate(w http.ResponseWriter, r *http.Request) {
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

	var req RateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.svc.RateExchange(r.Context(), userID, matchID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

// GetMatchRatings handles GET /api/v1/matches/{id}/ratings
func (h *Handler) GetMatchRatings(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.GetMatchRatings(r.Context(), userID, matchID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetUserRating handles GET /api/v1/users/{id}/rating
func (h *Handler) GetUserRating(w http.ResponseWriter, r *http.Request) {
	userID, err := parsePathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	result, err := h.svc.GetUserRating(r.Context(), userID)
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
	case errors.Is(err, ErrAlreadyRated):
		response.Error(w, http.StatusConflict, "You have already rated this exchange")
	case errors.Is(err, ErrNotMatchParticipant):
		response.Error(w, http.StatusForbidden, "You are not a participant in this match")
	case errors.Is(err, ErrMatchNotCompleted):
		response.Error(w, http.StatusConflict, "Match is not completed yet")
	case errors.Is(err, matching.ErrMatchNotFound):
		response.Error(w, http.StatusNotFound, "Match not found")
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
