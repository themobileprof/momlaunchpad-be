package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/symptoms"
)

const maxSummariesPerRequest = 2

// SymptomHandler handles symptom history API endpoints
type SymptomHandler struct {
	db         *db.DB
	summarizer *symptoms.Summarizer
}

// NewSymptomHandler creates a new symptom handler
func NewSymptomHandler(database *db.DB, summarizer *symptoms.Summarizer) *SymptomHandler {
	return &SymptomHandler{
		db:         database,
		summarizer: summarizer,
	}
}

// GetSymptomHistory returns symptom history for the authenticated user
// GET /api/symptoms/history?limit=20&type=headache
func (h *SymptomHandler) GetSymptomHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}

	symptomType := c.Query("type") // Optional filter by symptom type

	symptoms, err := h.db.GetSymptomHistory(c.Request.Context(), userID.(string), symptomType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symptom history"})
		return
	}

	h.ensureSummaries(c.Request.Context(), symptoms)

	c.JSON(http.StatusOK, gin.H{
		"symptoms": symptoms,
		"count":    len(symptoms),
	})
}

// GetRecentSymptoms returns most recent symptoms for dashboard/overview
// GET /api/symptoms/recent?limit=5
func (h *SymptomHandler) GetRecentSymptoms(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}

	records, err := h.db.GetRecentSymptoms(c.Request.Context(), userID.(string), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symptoms"})
		return
	}

	h.ensureSummaries(c.Request.Context(), records)

	c.JSON(http.StatusOK, gin.H{
		"symptoms": records,
	})
}

// MarkSymptomResolved marks a symptom as resolved
// PUT /api/symptoms/:id/resolve
func (h *SymptomHandler) MarkSymptomResolved(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	symptomID := c.Param("id")
	if symptomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symptom ID required"})
		return
	}

	err := h.db.MarkSymptomResolved(c.Request.Context(), symptomID, userID.(string))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Symptom not found or already resolved"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark symptom as resolved"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Symptom marked as resolved",
	})
}

// GetSymptomStats provides summary statistics about symptoms
// GET /api/symptoms/stats
func (h *SymptomHandler) GetSymptomStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get all symptoms for stats calculation
	symptoms, err := h.db.GetSymptomHistory(c.Request.Context(), userID.(string), "", 1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symptoms"})
		return
	}

	// Calculate stats
	stats := map[string]interface{}{
		"total_symptoms": len(symptoms),
		"ongoing":        0,
		"resolved":       0,
		"by_type":        make(map[string]int),
		"by_severity":    make(map[string]int),
	}

	for _, symptom := range symptoms {
		// Count by resolution status
		if isResolved, ok := symptom["is_resolved"].(bool); ok && isResolved {
			stats["resolved"] = stats["resolved"].(int) + 1
		} else {
			stats["ongoing"] = stats["ongoing"].(int) + 1
		}

		// Count by type
		if symptomType, ok := symptom["symptom_type"].(string); ok {
			byType := stats["by_type"].(map[string]int)
			byType[symptomType]++
		}

		// Count by severity
		if severity, ok := symptom["severity"].(string); ok {
			bySeverity := stats["by_severity"].(map[string]int)
			bySeverity[severity]++
		}
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SymptomHandler) ensureSummaries(ctx context.Context, records []map[string]interface{}) {
	llmAttempts := 0
	for _, record := range records {
		if summary, ok := record["summary"].(string); ok && summary != "" {
			continue
		}

		id, _ := record["id"].(string)
		symptomType, _ := record["symptom_type"].(string)
		description, _ := record["description"].(string)
		severity := stringField(record["severity"])
		frequency := stringField(record["frequency"])
		if id == "" || description == "" {
			continue
		}

		var summary string
		if h.summarizer != nil && llmAttempts < maxSummariesPerRequest {
			var usedLLM bool
			summary, usedLLM = h.summarizer.TrySummarize(
				ctx, symptomType, description, severity, frequency,
			)
			if usedLLM {
				llmAttempts++
			}
		} else {
			summary = symptoms.FallbackSummary(symptomType, description, severity)
		}

		record["summary"] = summary
		_ = h.db.UpdateSymptomSummary(ctx, id, summary)
	}
}

func stringField(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
