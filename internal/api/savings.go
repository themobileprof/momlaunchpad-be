package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// SavingsHandler handles savings endpoints
type SavingsHandler struct {
	db *db.DB
}

// NewSavingsHandler creates a new savings handler
func NewSavingsHandler(database *db.DB) *SavingsHandler {
	return &SavingsHandler{
		db: database,
	}
}

// SavingsSummaryResponse represents the savings summary
type SavingsSummaryResponse struct {
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date,omitempty"`
	SavingsGoal          float64    `json:"savings_goal"`
	TotalSaved           float64    `json:"total_saved"`
	ProgressPercentage   float64    `json:"progress_percentage"`
	DaysUntilDelivery    *int       `json:"days_until_delivery,omitempty"`
}

// CreateSavingsEntryRequest represents a savings entry creation request
type CreateSavingsEntryRequest struct {
	Amount      float64    `json:"amount" binding:"required"`
	Description string     `json:"description"`
	EntryDate   *time.Time `json:"entry_date"`
}

// SavingsEntryResponse represents a savings entry
type SavingsEntryResponse struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description,omitempty"`
	EntryDate   time.Time `json:"entry_date"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateEDDRequest represents an EDD update request
type UpdateEDDRequest struct {
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`
}

// UpdateSavingsGoalRequest represents a savings goal update request
type UpdateSavingsGoalRequest struct {
	SavingsGoal float64 `json:"savings_goal" binding:"required,min=0"`
}

// GetSavingsSummary retrieves the savings summary for the current user
func (h *SavingsHandler) GetSavingsSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)
	log.Printf("[DEBUG] GetSavingsSummary called for user: %s", userID)

	// Get user to fetch EDD and goal
	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] GetSavingsSummary failed to fetch user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Get total savings
	totalSaved, err := h.db.GetTotalSavings(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] GetSavingsSummary failed to get total savings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate savings"})
		return
	}

	savingsGoal := 0.0
	if user.SavingsGoal != nil {
		savingsGoal = *user.SavingsGoal
	}

	progressPercentage := 0.0
	if savingsGoal > 0 {
		progressPercentage = (totalSaved / savingsGoal) * 100
		if progressPercentage > 100 {
			progressPercentage = 100
		}
	}

	response := SavingsSummaryResponse{
		ExpectedDeliveryDate: user.ExpectedDeliveryDate,
		SavingsGoal:          savingsGoal,
		TotalSaved:           totalSaved,
		ProgressPercentage:   progressPercentage,
	}

	// Calculate days until delivery if EDD is set
	if user.ExpectedDeliveryDate != nil {
		days := int(time.Until(*user.ExpectedDeliveryDate).Hours() / 24)
		response.DaysUntilDelivery = &days
	}

	log.Printf("[DEBUG] GetSavingsSummary success. Response: %+v", response)
	c.JSON(http.StatusOK, response)
}

// CreateSavingsEntry creates a new savings entry
func (h *SavingsHandler) CreateSavingsEntry(c *gin.Context) {
	userID := middleware.GetUserID(c)
	log.Printf("[DEBUG] CreateSavingsEntry called for user: %s", userID)

	var req CreateSavingsEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entryDate := time.Now()
	if req.EntryDate != nil {
		entryDate = *req.EntryDate
	}

	entry := &db.SavingsEntry{
		UserID:      userID,
		Amount:      req.Amount,
		Description: &req.Description,
		EntryDate:   entryDate,
	}

	if err := h.db.CreateSavingsEntry(c.Request.Context(), entry); err != nil {
		log.Printf("[ERROR] CreateSavingsEntry failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create savings entry"})
		return
	}

	response := savingsEntryToResponse(entry)
	c.JSON(http.StatusCreated, response)
}

// GetSavingsEntries retrieves all savings entries for the current user
func (h *SavingsHandler) GetSavingsEntries(c *gin.Context) {
	userID := middleware.GetUserID(c)
	log.Printf("[DEBUG] GetSavingsEntries called for user: %s", userID)

	entries, err := h.db.GetUserSavingsEntries(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] GetSavingsEntries failed to fetch entries: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch savings entries"})
		return
	}

	response := make([]SavingsEntryResponse, 0, len(entries))
	for i := range entries {
		response = append(response, savingsEntryToResponse(&entries[i]))
	}

	log.Printf("[DEBUG] GetSavingsEntries success. Count: %d", len(response))
	c.JSON(http.StatusOK, response)
}

// UpdateEDD updates the user's expected delivery date
func (h *SavingsHandler) UpdateEDD(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req UpdateEDDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.UpdateUserEDD(c.Request.Context(), userID, req.ExpectedDeliveryDate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expected delivery date"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expected delivery date updated successfully"})
}

// UpdateSavingsGoal updates the user's savings goal
func (h *SavingsHandler) UpdateSavingsGoal(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req UpdateSavingsGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.UpdateUserSavingsGoal(c.Request.Context(), userID, req.SavingsGoal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update savings goal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Savings goal updated successfully"})
}

// savingsEntryToResponse converts a db.SavingsEntry to SavingsEntryResponse
func savingsEntryToResponse(entry *db.SavingsEntry) SavingsEntryResponse {
	response := SavingsEntryResponse{
		ID:        entry.ID,
		Amount:    entry.Amount,
		EntryDate: entry.EntryDate,
		CreatedAt: entry.CreatedAt,
	}
	if entry.Description != nil {
		response.Description = *entry.Description
	}
	return response
}
