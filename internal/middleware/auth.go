package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/auth"
	"github.com/yeabt/lewe/internal/response"
)

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey string

const userIDKey contextKey = "userID"

// Auth returns middleware that validates JWT access tokens.
// On success, the authenticated user's ID is injected into the request context.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, msg := authenticate(r, jwtSecret)
			if msg != "" {
				response.Error(w, http.StatusUnauthorized, msg)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth injects the user ID when a valid access token is supplied, but
// lets anonymous requests (or ones with a bad token) through unauthenticated.
func OptionalAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, msg := authenticate(r, jwtSecret); msg == "" {
				r = r.WithContext(context.WithValue(r.Context(), userIDKey, userID))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authenticate extracts and validates the bearer token. It returns a non-empty
// message describing the failure when the request is not authenticated.
func authenticate(r *http.Request, jwtSecret string) (pgtype.UUID, string) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return pgtype.UUID{}, "Missing authorization header"
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return pgtype.UUID{}, "Invalid authorization header format"
	}

	claims, err := auth.ValidateAccessToken(parts[1], jwtSecret)
	if err != nil {
		return pgtype.UUID{}, "Invalid or expired token"
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return pgtype.UUID{}, "Invalid token claims"
	}
	return userID, ""
}

// UserIDFromContext extracts the authenticated user's UUID from the request context.
func UserIDFromContext(ctx context.Context) (pgtype.UUID, error) {
	id, ok := ctx.Value(userIDKey).(pgtype.UUID)
	if !ok {
		return pgtype.UUID{}, errors.New("user ID not found in context")
	}
	return id, nil
}
