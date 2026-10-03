package users

import (
	"errors"
	"net/http"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/request"
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
	if err := request.DecodeJSON(w, r, &req); err != nil {
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
	if err := request.DecodeJSON(w, r, &req); err != nil {
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

// TelegramLogin handles POST /api/v1/auth/telegram
func (h *Handler) TelegramLogin(w http.ResponseWriter, r *http.Request) {
	var req TelegramRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	authResp, err := h.svc.TelegramLogin(r.Context(), req.InitData)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, authResp)
}

// RefreshToken handles POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
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

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// VerifyEmail handles POST /api/v1/auth/verify-email
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), req.Token); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ResendVerification handles POST /api/v1/users/me/verify-email
func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	if err := h.svc.ResendVerification(r.Context(), userID); err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "Verification email sent"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req EmailRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]string{
		"message": "If an account uses that email, we've sent a link to reset the password",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword handles PUT /api/v1/users/me/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	var req ChangePasswordRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	authResp, err := h.svc.ChangePassword(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, authResp)
}

// GetPublicProfile handles GET /api/v1/users/{id}
func (h *Handler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	id, err := request.PathUUID(r, "id")
	if err != nil {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	profile, err := h.svc.GetPublicProfile(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, profile)
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
	if err := request.DecodeJSON(w, r, &req); err != nil {
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
		response.ValidationError(w, validationErrs)
		return
	}

	switch {
	case errors.Is(err, ErrEmailTaken):
		response.Error(w, http.StatusConflict, "Email already registered")
	case errors.Is(err, ErrTelegramDisabled):
		response.Error(w, http.StatusNotImplemented, err.Error())
	case errors.Is(err, ErrInvalidTelegramData):
		response.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "Invalid email or password")
	case errors.Is(err, ErrInvalidRefreshToken):
		response.Error(w, http.StatusUnauthorized, "Invalid or expired refresh token")
	case errors.Is(err, ErrInvalidAccountToken):
		response.Error(w, http.StatusBadRequest, "This link is invalid or has expired")
	case errors.Is(err, ErrAlreadyVerified):
		response.Error(w, http.StatusConflict, "Email is already verified")
	case errors.Is(err, ErrUserNotFound):
		response.Error(w, http.StatusNotFound, "User not found")
	default:
		response.InternalError(w, err)
	}
}
