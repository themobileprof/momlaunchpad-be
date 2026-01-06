package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/subscription"
)

// SubscriptionHandler handles subscription and quota management endpoints
type SubscriptionHandler struct {
	subManager *subscription.Manager
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(subMgr *subscription.Manager) *SubscriptionHandler {
	return &SubscriptionHandler{
		subManager: subMgr,
	}
}

// GetMyQuota returns the current user's quota status for a feature
// GET /api/subscription/quota/:feature
func (h *SubscriptionHandler) GetMyQuota(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	featureCode := c.Param("feature")
	if featureCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feature code required"})
		return
	}

	// Check if user has the feature
	hasFeature, err := h.subManager.HasFeature(c.Request.Context(), userID, featureCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check feature access"})
		return
	}

	if !hasFeature {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "feature not available",
			"feature": featureCode,
		})
		return
	}

	// Check quota status
	withinQuota, err := h.subManager.CheckQuota(c.Request.Context(), userID, featureCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check quota"})
		return
	}

	// Get detailed quota info (requires new method)
	quotaInfo, err := h.subManager.GetQuotaInfo(c.Request.Context(), userID, featureCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quota info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"feature":      featureCode,
		"has_access":   true,
		"within_quota": withinQuota,
		"quota_limit":  quotaInfo.QuotaLimit,
		"quota_used":   quotaInfo.UsageCount,
		"quota_period": quotaInfo.QuotaPeriod,
		"period_end":   quotaInfo.PeriodEnd,
	})
}

// GetMyFeatures returns all features available to the current user
// GET /api/subscription/features
func (h *SubscriptionHandler) GetMyFeatures(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	features, err := h.subManager.GetUserFeatures(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get features"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"features": features,
	})
}

// GetMySubscription returns the current user's active subscription
// GET /api/subscription/me
func (h *SubscriptionHandler) GetMySubscription(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := h.subManager.GetActiveSubscription(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}

	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub,
	})
}

// Admin endpoints

// ListAllPlans returns all available subscription plans (admin only)
// GET /api/admin/plans
func (h *SubscriptionHandler) ListAllPlans(c *gin.Context) {
	plans, err := h.subManager.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list plans"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plans": plans,
	})
}

// GetUserSubscription returns a user's subscription details (admin only)
// GET /api/admin/users/:userId/subscription
func (h *SubscriptionHandler) GetUserSubscription(c *gin.Context) {
	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID required"})
		return
	}

	sub, err := h.subManager.GetActiveSubscription(c.Request.Context(), targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}

	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":      targetUserID,
		"subscription": sub,
	})
}

// UpdateUserPlan changes a user's subscription plan (admin only)
// PUT /api/admin/users/:userId/plan
func (h *SubscriptionHandler) UpdateUserPlan(c *gin.Context) {
	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID required"})
		return
	}

	var req struct {
		PlanCode string `json:"plan_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.subManager.UpdateUserPlan(c.Request.Context(), targetUserID, req.PlanCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "plan updated successfully",
		"user_id": targetUserID,
		"plan":    req.PlanCode,
	})
}

// GetUserQuotaUsage returns quota usage stats for a user (admin only)
// GET /api/admin/users/:userId/quota/:feature
func (h *SubscriptionHandler) GetUserQuotaUsage(c *gin.Context) {
	targetUserID := c.Param("userId")
	featureCode := c.Param("feature")

	if targetUserID == "" || featureCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID and feature code required"})
		return
	}

	quotaInfo, err := h.subManager.GetQuotaInfo(c.Request.Context(), targetUserID, featureCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quota info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": targetUserID,
		"feature": featureCode,
		"quota":   quotaInfo,
	})
}

// ResetUserQuota resets quota usage for a user/feature (admin only)
// POST /api/admin/users/:userId/quota/:feature/reset
func (h *SubscriptionHandler) ResetUserQuota(c *gin.Context) {
	targetUserID := c.Param("userId")
	featureCode := c.Param("feature")

	if targetUserID == "" || featureCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID and feature code required"})
		return
	}

	if err := h.subManager.ResetQuota(c.Request.Context(), targetUserID, featureCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset quota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "quota reset successfully",
		"user_id": targetUserID,
		"feature": featureCode,
	})
}

// GetQuotaStats returns system-wide quota statistics (admin only)
// GET /api/admin/quota/stats
func (h *SubscriptionHandler) GetQuotaStats(c *gin.Context) {
	// Optional query params for filtering
	featureCode := c.Query("feature")
	planCode := c.Query("plan")

	// Parse period (e.g., "daily", "weekly", "monthly", default "today")
	period := c.DefaultQuery("period", "today")

	stats, err := h.subManager.GetQuotaStats(c.Request.Context(), featureCode, planCode, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quota stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period": period,
		"stats":  stats,
	})
}

// GrantFeature grants a specific feature to a user temporarily (admin only)
// POST /api/admin/users/:userId/features
func (h *SubscriptionHandler) GrantFeature(c *gin.Context) {
	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID required"})
		return
	}

	var req struct {
		FeatureKey string `json:"feature_key" binding:"required"`
		ExpiresAt  *int64 `json:"expires_at"` // Unix timestamp, optional
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.subManager.GrantFeature(c.Request.Context(), targetUserID, req.FeatureKey, req.ExpiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to grant feature"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "feature granted successfully",
		"user_id": targetUserID,
		"feature": req.FeatureKey,
	})
}
