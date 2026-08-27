package users

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

// Handler handles HTTP requests for the users domain.
type Handler struct {
	svc *Service
}

// NewHandler creates a new user handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register handles POST /api/v1/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	authResp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, authResp)
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	authResp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, authResp)
}

// RefreshToken handles POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	authResp, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, authResp)
}

// GetProfile handles GET /api/v1/users/me
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/users/me
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	profile, err := h.svc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, profile)
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
	case errors.Is(err, ErrEmailTaken):
		response.Error(w, http.StatusConflict, "Email already registered")
	case errors.Is(err, ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "Invalid email or password")
	case errors.Is(err, ErrInvalidRefreshToken):
		response.Error(w, http.StatusUnauthorized, "Invalid or expired refresh token")
	case errors.Is(err, ErrUserNotFound):
		response.Error(w, http.StatusNotFound, "User not found")
	default:
		response.Error(w, http.StatusInternalServerError, "Internal server error")
	}
}

// parseUUID parses a string into a pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(s)
	return id, err
}
