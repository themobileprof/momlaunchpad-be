package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
)

// AdminHandler handles admin management endpoints
type AdminHandler struct {
	db      *db.DB
	langMgr *language.Manager
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(database *db.DB, langMgr *language.Manager) *AdminHandler {
	return &AdminHandler{
		db:      database,
		langMgr: langMgr,
	}
}

// ============================================================================
// PLAN MANAGEMENT
// ============================================================================

// CreatePlanRequest represents a request to create a plan
type CreatePlanRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdatePlanRequest represents a request to update a plan
type UpdatePlanRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      *bool  `json:"active"`
}

// CreatePlan creates a new subscription plan
// POST /api/admin/plans
func (h *AdminHandler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	plan, err := h.db.CreatePlan(c.Request.Context(), req.Code, req.Name, req.Description)
	if err != nil {
		if err == db.ErrAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "plan code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create plan"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"plan": plan})
}

// UpdatePlan updates an existing plan
// PUT /api/admin/plans/:planId
func (h *AdminHandler) UpdatePlan(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("planId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan ID"})
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.db.UpdatePlan(c.Request.Context(), planID, req.Name, req.Description, req.Active); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "plan updated successfully"})
}

// DeletePlan deactivates a plan (soft delete)
// DELETE /api/admin/plans/:planId
func (h *AdminHandler) DeletePlan(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("planId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan ID"})
		return
	}

	if err := h.db.DeactivatePlan(c.Request.Context(), planID); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "plan deactivated successfully"})
}

// ============================================================================
// FEATURE MANAGEMENT
// ============================================================================

// CreateFeatureRequest represents a request to create a feature
type CreateFeatureRequest struct {
	FeatureKey  string `json:"feature_key" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateFeatureRequest represents a request to update a feature
type UpdateFeatureRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AssignFeatureToPlanRequest represents a request to assign a feature to a plan
type AssignFeatureToPlanRequest struct {
	QuotaLimit  *int   `json:"quota_limit"`  // null = unlimited
	QuotaPeriod string `json:"quota_period"` // daily, weekly, monthly, unlimited
}

// ListFeatures returns all features
// GET /api/admin/features
func (h *AdminHandler) ListFeatures(c *gin.Context) {
	features, err := h.db.GetAllFeatures(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list features"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"features": features})
}

// CreateFeature creates a new feature
// POST /api/admin/features
func (h *AdminHandler) CreateFeature(c *gin.Context) {
	var req CreateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	feature, err := h.db.CreateFeature(c.Request.Context(), req.FeatureKey, req.Name, req.Description)
	if err != nil {
		if err == db.ErrAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "feature key already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create feature"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"feature": feature})
}

// UpdateFeature updates an existing feature
// PUT /api/admin/features/:featureId
func (h *AdminHandler) UpdateFeature(c *gin.Context) {
	featureID, err := strconv.Atoi(c.Param("featureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var req UpdateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.db.UpdateFeature(c.Request.Context(), featureID, req.Name, req.Description); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update feature"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature updated successfully"})
}

// DeleteFeature removes a feature
// DELETE /api/admin/features/:featureId
func (h *AdminHandler) DeleteFeature(c *gin.Context) {
	featureID, err := strconv.Atoi(c.Param("featureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	if err := h.db.DeleteFeature(c.Request.Context(), featureID); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete feature"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature deleted successfully"})
}

// AssignFeatureToPlan assigns a feature to a plan with quota settings
// POST /api/admin/plans/:planId/features/:featureId
func (h *AdminHandler) AssignFeatureToPlan(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("planId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan ID"})
		return
	}

	featureID, err := strconv.Atoi(c.Param("featureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var req AssignFeatureToPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate quota period
	validPeriods := map[string]bool{"daily": true, "weekly": true, "monthly": true, "unlimited": true}
	if req.QuotaPeriod == "" {
		req.QuotaPeriod = "unlimited"
	}
	if !validPeriods[req.QuotaPeriod] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota period"})
		return
	}

	if err := h.db.AssignFeatureToPlan(c.Request.Context(), planID, featureID, req.QuotaLimit, req.QuotaPeriod); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign feature to plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature assigned to plan successfully"})
}

// RemoveFeatureFromPlan removes a feature from a plan
// DELETE /api/admin/plans/:planId/features/:featureId
func (h *AdminHandler) RemoveFeatureFromPlan(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("planId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan ID"})
		return
	}

	featureID, err := strconv.Atoi(c.Param("featureId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	if err := h.db.RemoveFeatureFromPlan(c.Request.Context(), planID, featureID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove feature from plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature removed from plan successfully"})
}

// GetPlanFeatures returns all features for a plan with quotas
// GET /api/admin/plans/:planId/features
func (h *AdminHandler) GetPlanFeatures(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("planId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan ID"})
		return
	}

	features, err := h.db.GetPlanFeatures(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get plan features"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"features": features})
}

// ============================================================================
// LANGUAGE MANAGEMENT
// ============================================================================

// CreateLanguageRequest represents a request to create a language
type CreateLanguageRequest struct {
	Code           string `json:"code" binding:"required"`
	Name           string `json:"name" binding:"required"`
	NativeName     string `json:"native_name" binding:"required"`
	IsEnabled      bool   `json:"is_enabled"`
	IsExperimental bool   `json:"is_experimental"`
}

// UpdateLanguageRequest represents a request to update a language
type UpdateLanguageRequest struct {
	Name           string `json:"name"`
	NativeName     string `json:"native_name"`
	IsEnabled      *bool  `json:"is_enabled"`
	IsExperimental *bool  `json:"is_experimental"`
}

// ListLanguages returns all languages
// GET /api/admin/languages
func (h *AdminHandler) ListLanguages(c *gin.Context) {
	languages, err := h.db.GetAllLanguages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list languages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"languages": languages})
}

// CreateLanguage creates a new language
// POST /api/admin/languages
func (h *AdminHandler) CreateLanguage(c *gin.Context) {
	var req CreateLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	lang, err := h.db.CreateLanguage(c.Request.Context(), req.Code, req.Name, req.NativeName, req.IsEnabled, req.IsExperimental)
	if err != nil {
		if err == db.ErrAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "language code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create language"})
		return
	}

	// Sync with in-memory language manager
	h.langMgr.AddLanguage(language.LanguageInfo{
		Code:           lang.Code,
		Name:           lang.Name,
		NativeName:     lang.NativeName,
		IsEnabled:      lang.IsEnabled,
		IsExperimental: lang.IsExperimental,
	})

	c.JSON(http.StatusCreated, gin.H{"language": lang})
}

// UpdateLanguage updates an existing language
// PUT /api/admin/languages/:code
func (h *AdminHandler) UpdateLanguage(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language code required"})
		return
	}

	var req UpdateLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	lang, err := h.db.UpdateLanguage(c.Request.Context(), code, req.Name, req.NativeName, req.IsEnabled, req.IsExperimental)
	if err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "language not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update language"})
		return
	}

	// Sync with in-memory language manager
	h.langMgr.AddLanguage(language.LanguageInfo{
		Code:           lang.Code,
		Name:           lang.Name,
		NativeName:     lang.NativeName,
		IsEnabled:      lang.IsEnabled,
		IsExperimental: lang.IsExperimental,
	})

	c.JSON(http.StatusOK, gin.H{"message": "language updated successfully", "language": lang})
}

// DeleteLanguage removes a language (cannot delete English)
// DELETE /api/admin/languages/:code
func (h *AdminHandler) DeleteLanguage(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language code required"})
		return
	}

	if code == "en" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete default language (English)"})
		return
	}

	if err := h.db.DeleteLanguage(c.Request.Context(), code); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "language not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete language"})
		return
	}

	// Disable in memory manager (don't remove to allow fallback)
	h.langMgr.DisableLanguage(code)

	c.JSON(http.StatusOK, gin.H{"message": "language deleted successfully"})
}

// ============================================================================
// ANALYTICS
// ============================================================================

// TopicAnalytics represents aggregated topic data
type TopicAnalytics struct {
	Topic       string  `json:"topic"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	SampleQuery string  `json:"sample_query"`
}

// GetChatAnalytics returns analytics on what users are asking about
// GET /api/admin/analytics/topics
func (h *AdminHandler) GetChatAnalytics(c *gin.Context) {
	// Parse query parameters
	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get message intent analytics
	since := time.Now().AddDate(0, 0, -days)
	analytics, err := h.db.GetMessageAnalytics(c.Request.Context(), since, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period_days": days,
		"analytics":   analytics,
	})
}

// GetUserStats returns user statistics
// GET /api/admin/analytics/users
func (h *AdminHandler) GetUserStats(c *gin.Context) {
	stats, err := h.db.GetUserStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// GetCallHistory returns voice call history
// GET /api/admin/analytics/calls
func (h *AdminHandler) GetCallHistory(c *gin.Context) {
	// Parse query parameters
	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	since := time.Now().AddDate(0, 0, -days)
	calls, err := h.db.GetVoiceCallHistory(c.Request.Context(), since, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get call history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period_days": days,
		"calls":       calls,
	})
}

// ============================================================================
// SYSTEM SETTINGS
// ============================================================================

// UpdateSystemSettingRequest represents a request to update a setting
type UpdateSystemSettingRequest struct {
	Value string `json:"value" binding:"required"`
}

// GetSystemSettings returns all system settings
// GET /api/admin/settings
func (h *AdminHandler) GetSystemSettings(c *gin.Context) {
	settings, err := h.db.GetAllSystemSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// GetSystemSetting returns a single system setting by key
// GET /api/admin/settings/:key
func (h *AdminHandler) GetSystemSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "setting key required"})
		return
	}

	setting, err := h.db.GetSystemSetting(c.Request.Context(), key)
	if err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"setting": setting})
}

// UpdateSystemSetting updates a system setting value
// PUT /api/admin/settings/:key
func (h *AdminHandler) UpdateSystemSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "setting key required"})
		return
	}

	var req UpdateSystemSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.db.UpdateSystemSetting(c.Request.Context(), key, req.Value); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "setting updated successfully"})
}
