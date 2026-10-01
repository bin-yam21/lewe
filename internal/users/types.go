package users

import "time"

// --- Request DTOs ---

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the payload for refreshing an access token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// UpdateProfileRequest is the payload for updating a user's profile.
type UpdateProfileRequest struct {
	FullName  string  `json:"full_name"`
	Phone     *string `json:"phone"`
	Location  *string `json:"location"`
	Bio       *string `json:"bio"`
	AvatarURL *string `json:"avatar_url"`
	City      *string `json:"city"`
}

// TelegramAuthRequest carries Telegram.WebApp.initData from the Mini App.
type TelegramAuthRequest struct {
	InitData string `json:"init_data"`
}

// ChangePasswordRequest is the payload for changing a known password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// EmailRequest carries just an email address (forgot password).
type EmailRequest struct {
	Email string `json:"email"`
}

// TokenRequest carries an emailed one-time token (verify email).
type TokenRequest struct {
	Token string `json:"token"`
}

// ResetPasswordRequest sets a new password using an emailed reset token.
type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// --- Response DTOs ---

// AuthResponse is returned after successful registration or login.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// UserResponse is the public-facing representation of a user.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email,omitempty"`
	FullName  string    `json:"full_name"`
	Phone     *string   `json:"phone,omitempty"`
	Location  *string   `json:"location,omitempty"`
	Bio       *string   `json:"bio,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	EmailVerified    bool    `json:"email_verified"`
	City             *string `json:"city,omitempty"`
	TelegramUsername *string `json:"telegram_username,omitempty"`
}

// PublicProfileResponse is the representation of a user visible to anyone.
type PublicProfileResponse struct {
	ID        string        `json:"id"`
	FullName  string        `json:"full_name"`
	Location  *string       `json:"location,omitempty"`
	Bio       *string       `json:"bio,omitempty"`
	AvatarURL *string       `json:"avatar_url,omitempty"`
	City      *string       `json:"city,omitempty"`
	Rating    RatingSummary `json:"rating"`
	CreatedAt time.Time     `json:"created_at"`
}

// RatingSummary aggregates the ratings a user has received.
type RatingSummary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}
