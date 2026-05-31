package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// AdminCommunityHandler serves moderation endpoints for a separate admin web frontend.
type AdminCommunityHandler struct {
	db *db.DB
}

// NewAdminCommunityHandler creates an admin community handler.
func NewAdminCommunityHandler(database *db.DB) *AdminCommunityHandler {
	return &AdminCommunityHandler{db: database}
}

// RegisterRoutes mounts admin community moderation and catalog routes under /community.
func (h *AdminCommunityHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/community")
	{
		g.GET("/reports", h.ListReports)
		g.PUT("/reports/:id", h.UpdateReport)
		g.PUT("/posts/:id/status", h.UpdatePostStatus)
		g.POST("/users/:userId/badges", h.GrantBadge)
		g.DELETE("/users/:userId/badges/:badgeType", h.RevokeBadge)

		h.registerCatalogRoutes(g)
	}
}

func (h *AdminCommunityHandler) registerCatalogRoutes(g *gin.RouterGroup) {
	catalog := g.Group("/catalog")
	{
		catalog.GET("/interest-groups", h.ListInterestGroups)
		catalog.PUT("/interest-groups/:key", h.UpsertInterestGroup)
		catalog.GET("/interests", h.ListInterests)
		catalog.PUT("/interests/:key", h.UpsertInterest)
		catalog.GET("/badge-types", h.ListBadgeTypes)
		catalog.PUT("/badge-types/:key", h.UpsertBadgeType)
		catalog.GET("/event-types", h.ListEventTypes)
		catalog.PUT("/event-types/:key", h.UpsertEventType)
		catalog.GET("/countries", h.ListCountries)
		catalog.PUT("/countries/:code", h.UpsertCountry)
		catalog.GET("/regions", h.ListRegions)
		catalog.POST("/regions", h.CreateRegion)
		catalog.PUT("/regions/:id", h.UpdateRegion)
	}
}

// ListReports godoc
// @Summary      List moderation reports
// @Description  Returns community reports filtered by status for admin review.
// @Tags         admin-community
// @Produce      json
// @Param        status query string false "Report status filter" default(open)
// @Success      200 {object} ListCommunityReportsResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/reports [get]
func (h *AdminCommunityHandler) ListReports(c *gin.Context) {
	status := c.DefaultQuery("status", "open")
	reports, err := h.db.ListCommunityReports(c.Request.Context(), status, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load reports"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

type updateReportRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateReport godoc
// @Summary      Update report status
// @Description  Moves a report through the moderation workflow and records the reviewing admin.
// @Tags         admin-community
// @Accept       json
// @Produce      json
// @Param        id   path string true "Report ID"
// @Param        body body UpdateCommunityReportRequest true "New status"
// @Success      200 {object} UpdateCommunityReportResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/reports/{id} [put]
func (h *AdminCommunityHandler) UpdateReport(c *gin.Context) {
	reviewerID := middleware.GetUserID(c)
	var req updateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch req.Status {
	case "reviewed", "dismissed", "actioned":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}
	if err := h.db.UpdateCommunityReportStatus(c.Request.Context(), c.Param("id"), req.Status, reviewerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

type updatePostStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdatePostStatus godoc
// @Summary      Update post status
// @Description  Changes post visibility or moderation state (active, hidden, removed, pending_review).
// @Tags         admin-community
// @Accept       json
// @Produce      json
// @Param        id   path string true "Post ID"
// @Param        body body UpdateCommunityPostStatusRequest true "New status"
// @Success      200 {object} UpdateCommunityPostStatusResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/posts/{id}/status [put]
func (h *AdminCommunityHandler) UpdatePostStatus(c *gin.Context) {
	var req updatePostStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch req.Status {
	case "active", "hidden", "removed", "pending_review":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}
	if err := h.db.UpdateCommunityPostStatus(c.Request.Context(), c.Param("id"), req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

type grantBadgeRequest struct {
	BadgeType string `json:"badge_type" binding:"required"`
}

// GrantBadge godoc
// @Summary      Grant user badge
// @Description  Assigns a catalog badge type to a user.
// @Tags         admin-community
// @Accept       json
// @Produce      json
// @Param        userId path string true "User ID"
// @Param        body   body GrantCommunityBadgeRequest true "Badge type key"
// @Success      200 {object} GrantCommunityBadgeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/users/{userId}/badges [post]
func (h *AdminCommunityHandler) GrantBadge(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	var req grantBadgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ok, err := h.db.IsEnabledBadgeType(c.Request.Context(), req.BadgeType)
	if err != nil || !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid badge type"})
		return
	}
	if err := h.db.GrantUserBadge(c.Request.Context(), c.Param("userId"), req.BadgeType, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant badge"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge_type": req.BadgeType})
}

// RevokeBadge godoc
// @Summary      Revoke user badge
// @Description  Removes a badge type from a user.
// @Tags         admin-community
// @Produce      json
// @Param        userId    path string true "User ID"
// @Param        badgeType path string true "Badge type key"
// @Success      200 {object} RevokeCommunityBadgeResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/users/{userId}/badges/{badgeType} [delete]
func (h *AdminCommunityHandler) RevokeBadge(c *gin.Context) {
	if err := h.db.RevokeUserBadge(c.Request.Context(), c.Param("userId"), c.Param("badgeType")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke badge"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
