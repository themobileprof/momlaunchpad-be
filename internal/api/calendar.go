package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// CalendarHandler handles calendar/reminder endpoints
type CalendarHandler struct {
	db *db.DB
}

// NewCalendarHandler creates a new calendar handler
func NewCalendarHandler(database *db.DB) *CalendarHandler {
	return &CalendarHandler{
		db: database,
	}
}

// CreateReminderRequest represents a reminder creation request
type CreateReminderRequest struct {
	Title        string    `json:"title" binding:"required"`
	Description  string    `json:"description"`
	ReminderTime time.Time `json:"reminder_time" binding:"required"`
}

// UpdateReminderRequest represents a reminder update request
type UpdateReminderRequest struct {
	Title        *string    `json:"title"`
	Description  *string    `json:"description"`
	ReminderTime *time.Time `json:"reminder_time"`
	IsCompleted  *bool      `json:"is_completed"`
}

// ReminderResponse represents a reminder response
type ReminderResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	ReminderTime time.Time `json:"reminder_time"`
	IsCompleted  bool      `json:"is_completed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateReminder creates a new reminder
func (h *CalendarHandler) CreateReminder(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reminder := &db.Reminder{
		UserID:       userID,
		Title:        req.Title,
		Description:  &req.Description,
		ReminderTime: req.ReminderTime,
		IsCompleted:  false,
	}

	if err := h.db.CreateReminder(c.Request.Context(), reminder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reminder"})
		return
	}

	c.JSON(http.StatusCreated, reminderToResponse(reminder))
}

// GetReminders retrieves all reminders for the current user
func (h *CalendarHandler) GetReminders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	reminders, err := h.db.GetUserReminders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reminders"})
		return
	}

	response := make([]ReminderResponse, 0, len(reminders))
	for i := range reminders {
		response = append(response, reminderToResponse(&reminders[i]))
	}

	c.JSON(http.StatusOK, response)
}

// UpdateReminder updates a reminder
func (h *CalendarHandler) UpdateReminder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	reminderID := c.Param("id")

	var req UpdateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing reminder to verify ownership
	reminder, err := h.db.GetReminderByID(c.Request.Context(), reminderID)
	if err != nil || reminder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reminder not found"})
		return
	}

	if reminder.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields if provided
	if req.Title != nil {
		reminder.Title = *req.Title
	}
	if req.Description != nil {
		reminder.Description = req.Description
	}
	if req.ReminderTime != nil {
		reminder.ReminderTime = *req.ReminderTime
	}
	if req.IsCompleted != nil {
		reminder.IsCompleted = *req.IsCompleted
	}

	if err := h.db.UpdateReminder(c.Request.Context(), reminder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reminder"})
		return
	}

	c.JSON(http.StatusOK, reminderToResponse(reminder))
}

// DeleteReminder deletes a reminder
func (h *CalendarHandler) DeleteReminder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	reminderID := c.Param("id")

	// Get existing reminder to verify ownership
	reminder, err := h.db.GetReminderByID(c.Request.Context(), reminderID)
	if err != nil || reminder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reminder not found"})
		return
	}

	if reminder.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.db.DeleteReminder(c.Request.Context(), reminderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reminder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder deleted successfully"})
}

// reminderToResponse converts a db.Reminder to ReminderResponse
func reminderToResponse(reminder *db.Reminder) ReminderResponse {
	description := ""
	if reminder.Description != nil {
		description = *reminder.Description
	}

	return ReminderResponse{
		ID:           reminder.ID,
		Title:        reminder.Title,
		Description:  description,
		ReminderTime: reminder.ReminderTime,
		IsCompleted:  reminder.IsCompleted,
		CreatedAt:    reminder.CreatedAt,
		UpdatedAt:    reminder.UpdatedAt,
	}
}
