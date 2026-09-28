package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const secret = "test-secret"

func TestAccessTokenRoundTrip(t *testing.T) {
	tok, err := GenerateAccessToken("user-123", secret)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateAccessToken(tok, secret)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-123" || claims.Issuer != "lewe" {
		t.Errorf("claims = %+v", claims)
	}
}

func TestAccessTokenRejected(t *testing.T) {
	good, _ := GenerateAccessToken("u", secret)

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: "u",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	expiredStr, _ := expired.SignedString([]byte(secret))

	none := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u"})
	noneStr, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)

	for name, tok := range map[string]string{
		"wrong secret": good,
		"expired":      expiredStr,
		"alg none":     noneStr,
		"garbage":      "not.a.jwt",
	} {
		s := secret
		if name == "wrong secret" {
			s = "other-secret"
		}
		if _, err := ValidateAccessToken(tok, s); err != ErrInvalidToken {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestRefreshTokenString(t *testing.T) {
	a, _ := GenerateRefreshTokenString()
	b, _ := GenerateRefreshTokenString()
	if len(a) != 64 || a == b {
		t.Errorf("refresh tokens should be 64 hex chars and unique: %q %q", a, b)
	}
}
