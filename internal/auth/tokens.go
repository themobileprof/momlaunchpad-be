package auth

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// TokenExpiryDuration returns access-token lifetime from JWT_EXPIRY (e.g. 24h, 2160h).
func TokenExpiryDuration() time.Duration {
	const defaultExpiry = 24 * time.Hour
	if v := os.Getenv("JWT_EXPIRY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	// Legacy fallback if JWT_EXPIRY_DAYS was set during rollout.
	if v := os.Getenv("JWT_EXPIRY_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			return time.Duration(parsed) * 24 * time.Hour
		}
	}
	return defaultExpiry
}

// RefreshGraceDuration is how long after expiry /auth/refresh still accepts a token.
func RefreshGraceDuration() time.Duration {
	const defaultGrace = 365 * 24 * time.Hour
	if v := os.Getenv("JWT_REFRESH_GRACE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 0 {
			return d
		}
	}
	if v := os.Getenv("JWT_REFRESH_GRACE_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			return time.Duration(parsed) * 24 * time.Hour
		}
	}
	return defaultGrace
}

// GenerateUserToken issues a signed JWT for the user.
func GenerateUserToken(user *db.User, secret string) (string, error) {
	now := time.Now()
	claims := &middleware.JWTClaims{
		UserID:  user.ID,
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenExpiryDuration())),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseTokenForRefresh validates signature and expiry/grace for session refresh.
func ParseTokenForRefresh(tokenString, secret string) (*middleware.JWTClaims, error) {
	claims := &middleware.JWTClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithoutClaimsValidation(),
	)
	token, err := parser.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token signature")
	}

	if claims.ExpiresAt != nil {
		graceDeadline := claims.ExpiresAt.Time.Add(RefreshGraceDuration())
		if time.Now().After(graceDeadline) {
			return nil, fmt.Errorf("session expired")
		}
	}

	if claims.UserID == "" {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
