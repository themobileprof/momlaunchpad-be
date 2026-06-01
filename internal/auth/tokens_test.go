package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

func TestGenerateAndRefreshToken(t *testing.T) {
	t.Setenv("JWT_EXPIRY", "720h")
	t.Setenv("JWT_REFRESH_GRACE", "168h")

	user := &db.User{ID: "user-1", Email: "a@example.com", IsAdmin: false}
	secret := "test-secret"

	token, err := GenerateUserToken(user, secret)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ParseTokenForRefresh(token, secret)
	if err != nil {
		t.Fatalf("parse fresh token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("user id = %q", claims.UserID)
	}
}

func TestParseTokenForRefresh_AllowsRecentlyExpired(t *testing.T) {
	t.Setenv("JWT_REFRESH_GRACE", "168h")

	secret := "test-secret"
	claims := &middleware.JWTClaims{
		UserID:  "user-1",
		Email:   "a@example.com",
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ParseTokenForRefresh(tokenString, secret); err != nil {
		t.Fatalf("expected grace refresh, got %v", err)
	}
}

func TestParseTokenForRefresh_RejectsBeyondGrace(t *testing.T) {
	t.Setenv("JWT_REFRESH_GRACE", "24h")

	secret := "test-secret"
	claims := &middleware.JWTClaims{
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-10 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ParseTokenForRefresh(tokenString, secret); err == nil {
		t.Fatal("expected session expired error")
	}
}
