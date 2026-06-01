package auth

import (
	"testing"
	"time"
)

func TestTokenExpiryDuration_FromJWTExpiry(t *testing.T) {
	t.Setenv("JWT_EXPIRY", "24h")
	t.Setenv("JWT_EXPIRY_DAYS", "90")

	if got := TokenExpiryDuration(); got != 24*time.Hour {
		t.Fatalf("TokenExpiryDuration() = %v, want 24h", got)
	}
}

func TestRefreshGraceDuration_FromJWTRefreshGrace(t *testing.T) {
	t.Setenv("JWT_REFRESH_GRACE", "720h")

	if got := RefreshGraceDuration(); got != 720*time.Hour {
		t.Fatalf("RefreshGraceDuration() = %v, want 720h", got)
	}
}
