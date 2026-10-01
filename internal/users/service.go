package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/yeabt/lewe/internal/auth"
	"github.com/yeabt/lewe/internal/catalog"
	"github.com/yeabt/lewe/internal/mail"
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
	mailer    mail.Mailer
	appURL    string
	botToken  string
}

// Options configures the user service.
type Options struct {
	JWTSecret string
	// Mailer sends verification and password-reset emails (defaults to logging them).
	Mailer mail.Mailer
	// AppURL is the base URL of the client app; emailed links point at
	// {AppURL}/verify-email?token=… and {AppURL}/reset-password?token=….
	AppURL string
	// TelegramBotToken enables sign-in from the Telegram Mini App.
	TelegramBotToken string
}

// NewService creates a new user service.
func NewService(repo *Repository, tokenRepo *RefreshTokenRepository, opts Options) *Service {
	if opts.Mailer == nil {
		opts.Mailer = mail.LogMailer{}
	}
	return &Service{
		repo:      repo,
		tokenRepo: tokenRepo,
		jwtSecret: opts.JWTSecret,
		mailer:    opts.Mailer,
		appURL:    strings.TrimRight(opts.AppURL, "/"),
		botToken:  opts.TelegramBotToken,
	}
}

// Register creates a new user account and returns auth tokens.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	req.Email = normalizeEmail(req.Email)
	req.FullName = strings.TrimSpace(req.FullName)

	// Validate input
	errs := make(validator.Errors)
	validator.ValidateEmail(errs, "email", req.Email)
	validator.ValidateRequired(errs, "full_name", req.FullName)
	validator.ValidateMaxLength(errs, "full_name", req.FullName, 100)
	validatePassword(errs, "password", req.Password)
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

	// Ask the user to confirm their address. A mail failure shouldn't undo the
	// sign-up; they can request another link.
	if err := s.sendVerification(ctx, user); err != nil {
		log.Printf("send verification email to user %s: %v", user.ID.String(), err)
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

// Login authenticates a user and returns auth tokens.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	req.Email = normalizeEmail(req.Email)

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

	// Verify password (Telegram-only accounts have none)
	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
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

	// Revoke the presented token and learn its owner in one step (rotation)
	userID, err := s.tokenRepo.Consume(ctx, hashToken(refreshTokenStr))
	if err != nil {
		return nil, err
	}

	// Get the user
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Issue fresh token pair
	return s.generateAuthResponse(ctx, user)
}

// Logout revokes the given refresh token. Unknown or already-revoked tokens
// are ignored so the call is idempotent.
func (s *Service) Logout(ctx context.Context, refreshTokenStr string) error {
	if refreshTokenStr == "" {
		return nil
	}
	_, err := s.tokenRepo.Consume(ctx, hashToken(refreshTokenStr))
	if err != nil && !errors.Is(err, ErrInvalidRefreshToken) {
		return err
	}
	return nil
}

// GetPublicProfile returns the publicly visible profile of any user.
func (s *Service) GetPublicProfile(ctx context.Context, userID pgtype.UUID) (*PublicProfileResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	avg, count, err := s.repo.RatingSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	full := toUserResponse(user)
	return &PublicProfileResponse{
		ID:        full.ID,
		FullName:  full.FullName,
		Location:  full.Location,
		Bio:       full.Bio,
		AvatarURL: full.AvatarURL,
		City:      full.City,
		Rating:    RatingSummary{Average: math.Round(avg*100) / 100, Count: count},
		CreatedAt: full.CreatedAt,
	}, nil
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
	req.FullName = strings.TrimSpace(req.FullName)

	errs := make(validator.Errors)
	validator.ValidateRequired(errs, "full_name", req.FullName)
	validator.ValidateMaxLength(errs, "full_name", req.FullName, 100)
	if req.Bio != nil {
		validator.ValidateMaxLength(errs, "bio", *req.Bio, 2000)
	}
	req.City = trimmedOrNil(req.City)
	if req.City != nil {
		validator.ValidateOneOf(errs, "city", *req.City, catalog.Cities)
	}
	req.AvatarURL = trimmedOrNil(req.AvatarURL)
	if req.AvatarURL != nil && !validator.IsImageURL(*req.AvatarURL) {
		errs["avatar_url"] = "must be an http(s) URL or an uploaded image"
	}
	if errs.HasErrors() {
		return nil, errs
	}

	user, err := s.repo.Update(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	resp := toUserResponse(user)
	return &resp, nil
}

// --- helpers ---

// generateAuthResponse creates access + refresh tokens and builds the response.
func (s *Service) generateAuthResponse(ctx context.Context, user *UserRow) (*AuthResponse, error) {
	userIDStr := user.ID.String()

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
		ID:       u.ID.String(),
		Email:    u.Email,
		FullName: u.FullName,

		EmailVerified: u.EmailVerifiedAt.Valid,
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
	if u.City.Valid {
		resp.City = &u.City.String
	}
	if u.TelegramUsername.Valid {
		resp.TelegramUsername = &u.TelegramUsername.String
	}
	return resp
}

// trimmedOrNil trims s and turns an empty result into nil.
func trimmedOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// normalizeEmail lower-cases and trims an email so lookups are case-insensitive.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// hashToken returns the SHA-256 hex digest of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
