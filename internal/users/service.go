package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/yeabt/lewe/internal/auth"
	"github.com/yeabt/lewe/internal/validator"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// Service contains user business logic.
type Service struct {
	repo      *Repository
	tokenRepo *RefreshTokenRepository
	jwtSecret string
}

// NewService creates a new user service.
func NewService(repo *Repository, tokenRepo *RefreshTokenRepository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		tokenRepo: tokenRepo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user account and returns auth tokens.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Validate input
	errs := make(validator.Errors)
	validator.ValidateEmail(errs, "email", req.Email)
	validator.ValidateRequired(errs, "full_name", req.FullName)
	validator.ValidateRequired(errs, "password", req.Password)
	validator.ValidateMinLength(errs, "password", req.Password, 8)
	if errs.HasErrors() {
		return nil, errs
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user in DB
	user, err := s.repo.Create(ctx, req.Email, string(hash), req.FullName, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

// Login authenticates a user and returns auth tokens.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	// Validate input
	errs := make(validator.Errors)
	validator.ValidateEmail(errs, "email", req.Email)
	validator.ValidateRequired(errs, "password", req.Password)
	if errs.HasErrors() {
		return nil, errs
	}

	// Find user
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

// RefreshToken validates a refresh token and issues a new access token.
func (s *Service) RefreshToken(ctx context.Context, refreshTokenStr string) (*AuthResponse, error) {
	if refreshTokenStr == "" {
		return nil, ErrInvalidRefreshToken
	}

	// Hash the incoming token to look it up
	tokenHash := hashToken(refreshTokenStr)

	// Find valid token in DB
	rt, err := s.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Revoke the old refresh token (rotation)
	if err := s.tokenRepo.Revoke(ctx, tokenHash); err != nil {
		return nil, err
	}

	// Get the user
	user, err := s.repo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}

	// Issue fresh token pair
	return s.generateAuthResponse(ctx, user)
}

// GetProfile retrieves a user's profile by ID.
func (s *Service) GetProfile(ctx context.Context, userID pgtype.UUID) (*UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := toUserResponse(user)
	return &resp, nil
}

// UpdateProfile updates the authenticated user's profile.
func (s *Service) UpdateProfile(ctx context.Context, userID pgtype.UUID, req UpdateProfileRequest) (*UserResponse, error) {
	errs := make(validator.Errors)
	validator.ValidateRequired(errs, "full_name", req.FullName)
	if errs.HasErrors() {
		return nil, errs
	}

	user, err := s.repo.Update(ctx, userID, req.FullName, req.Phone, req.Location, req.Bio, req.AvatarURL)
	if err != nil {
		return nil, err
	}

	resp := toUserResponse(user)
	return &resp, nil
}

// --- helpers ---

// generateAuthResponse creates access + refresh tokens and builds the response.
func (s *Service) generateAuthResponse(ctx context.Context, user *UserRow) (*AuthResponse, error) {
	userIDStr := uuidToString(user.ID)

	accessToken, err := auth.GenerateAccessToken(userIDStr, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshTokenStr, err := auth.GenerateRefreshTokenString()
	if err != nil {
		return nil, err
	}

	// Store hashed refresh token in DB
	tokenHash := hashToken(refreshTokenStr)
	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(auth.RefreshTokenDuration), Valid: true}
	if _, err := s.tokenRepo.Create(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		User:         toUserResponse(user),
	}, nil
}

// toUserResponse maps a DB row to the public-facing DTO.
func toUserResponse(u *UserRow) UserResponse {
	resp := UserResponse{
		ID:       uuidToString(u.ID),
		Email:    u.Email,
		FullName: u.FullName,
	}
	if u.CreatedAt.Valid {
		resp.CreatedAt = u.CreatedAt.Time
	}
	if u.Phone.Valid {
		resp.Phone = &u.Phone.String
	}
	if u.Location.Valid {
		resp.Location = &u.Location.String
	}
	if u.Bio.Valid {
		resp.Bio = &u.Bio.String
	}
	if u.AvatarURL.Valid {
		resp.AvatarURL = &u.AvatarURL.String
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

// hashToken returns the SHA-256 hex digest of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
