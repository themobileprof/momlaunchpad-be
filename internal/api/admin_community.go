package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/community"
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

// RegisterRoutes mounts admin community moderation routes.
func (h *AdminCommunityHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/community")
	{
		g.GET("/reports", h.ListReports)
		g.PUT("/reports/:id", h.UpdateReport)
		g.PUT("/posts/:id/status", h.UpdatePostStatus)
		g.POST("/users/:userId/badges", h.GrantBadge)
		g.DELETE("/users/:userId/badges/:badgeType", h.RevokeBadge)
	}
}

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

func (h *AdminCommunityHandler) GrantBadge(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	var req grantBadgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !community.ValidBadgeTypes()[req.BadgeType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid badge type"})
		return
	}
	if err := h.db.GrantUserBadge(c.Request.Context(), c.Param("userId"), req.BadgeType, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant badge"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge_type": req.BadgeType})
}

func (h *AdminCommunityHandler) RevokeBadge(c *gin.Context) {
	if err := h.db.RevokeUserBadge(c.Request.Context(), c.Param("userId"), c.Param("badgeType")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke badge"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
