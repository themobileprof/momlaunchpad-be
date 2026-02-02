package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FeatureChecker interface {
	HasFeature(ctx context.Context, userID string, featureKey string) (bool, error)
}

type QuotaChecker interface {
	CheckQuota(ctx context.Context, userID string, featureCode string) (bool, error)
}

func RequireFeature(checker FeatureChecker, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, ok := c.Get("user_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}

		userID, ok := userIDVal.(string)
		if !ok || userID == "" {
			userID = c.GetString("userID")
		}
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		allowed, err := checker.HasFeature(c.Request.Context(), userID, featureKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "feature check failed"})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "feature not available"})
			return
		}
		c.Next()
	}
}

// CheckQuota validates user quota for a feature (pure function, deterministic)
func CheckQuota(checker QuotaChecker, featureCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract userID from context (set by JWT middleware upstream)
		userIDVal, ok := c.Get("user_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}

		userID, ok := userIDVal.(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		// Deterministic quota check (backend logic, not AI)
		withinQuota, err := checker.CheckQuota(c.Request.Context(), userID, featureCode)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "quota check failed"})
			return
		}

		if !withinQuota {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "quota exceeded"})
			return
		}

		c.Next()
	}
}
