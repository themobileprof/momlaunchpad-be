package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/welcome"
)

// WelcomeHandler serves personalized weekly welcome messages.
type WelcomeHandler struct {
	service *welcome.Service
}

// NewWelcomeHandler creates a welcome handler.
func NewWelcomeHandler(service *welcome.Service) *WelcomeHandler {
	return &WelcomeHandler{service: service}
}

// GetWelcome returns this week's welcome message (cached per user per week).
// GET /api/users/me/welcome
func (h *WelcomeHandler) GetWelcome(c *gin.Context) {
	userID := middleware.GetUserID(c)

	result, err := h.service.GetWeeklyWelcome(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load welcome message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      result.Message,
		"cache_date":   result.CacheDate.Format("2006-01-02"),
		"source":       result.Source,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
