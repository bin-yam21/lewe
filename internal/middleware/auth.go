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
			// Extract token from Authorization header
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			// Validate the JWT
			claims, err := auth.ValidateAccessToken(parts[1], jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Parse the user ID from claims and inject into context
			var userID pgtype.UUID
			if err := userID.Scan(claims.UserID); err != nil {
				response.Error(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user's UUID from the request context.
func UserIDFromContext(ctx context.Context) (pgtype.UUID, error) {
	id, ok := ctx.Value(userIDKey).(pgtype.UUID)
	if !ok {
		return pgtype.UUID{}, errors.New("user ID not found in context")
	}
	return id, nil
}
