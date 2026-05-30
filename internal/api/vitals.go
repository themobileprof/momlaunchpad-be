package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// VitalsHandler handles standalone vital sign readings.
type VitalsHandler struct {
	db *db.DB
}

// NewVitalsHandler creates a new vitals handler.
func NewVitalsHandler(database *db.DB) *VitalsHandler {
	return &VitalsHandler{db: database}
}

// CreateVitalReadingRequest is the body for logging vitals manually.
type CreateVitalReadingRequest struct {
	RecordedAt             time.Time `json:"recorded_at" binding:"required"`
	BloodPressureSystolic  *int      `json:"blood_pressure_systolic"`
	BloodPressureDiastolic *int      `json:"blood_pressure_diastolic"`
	WeightKg               *float64  `json:"weight_kg"`
	HeartRateBpm           *int      `json:"heart_rate_bpm"`
	TemperatureCelsius     *float64  `json:"temperature_celsius"`
	FundalHeightCm         *float64  `json:"fundal_height_cm"`
	FetalHeartRateBpm      *int      `json:"fetal_heart_rate_bpm"`
	GestationalAgeWeeks    *int      `json:"gestational_age_weeks"`
	Notes                  *string   `json:"notes"`
}

// VitalReadingResponse is the API representation of a vital reading.
type VitalReadingResponse struct {
	ID                     string    `json:"id"`
	UserID                 string    `json:"user_id"`
	RecordedAt             time.Time `json:"recorded_at"`
	BloodPressureSystolic  *int      `json:"blood_pressure_systolic,omitempty"`
	BloodPressureDiastolic *int      `json:"blood_pressure_diastolic,omitempty"`
	WeightKg               *float64  `json:"weight_kg,omitempty"`
	HeartRateBpm           *int      `json:"heart_rate_bpm,omitempty"`
	TemperatureCelsius     *float64  `json:"temperature_celsius,omitempty"`
	FundalHeightCm         *float64  `json:"fundal_height_cm,omitempty"`
	FetalHeartRateBpm      *int      `json:"fetal_heart_rate_bpm,omitempty"`
	GestationalAgeWeeks    *int      `json:"gestational_age_weeks,omitempty"`
	Notes                  string    `json:"notes,omitempty"`
	Source                 string    `json:"source"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// ListVitalReadings returns recent vital readings for the authenticated user.
func (h *VitalsHandler) ListVitalReadings(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limitStr := c.DefaultQuery("limit", "30")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 30
	}

	readings, err := h.db.GetUserVitalReadings(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vital readings"})
		return
	}

	response := make([]VitalReadingResponse, 0, len(readings))
	for i := range readings {
		response = append(response, vitalReadingToResponse(&readings[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"readings": response,
		"count":    len(response),
	})
}

// CreateVitalReading logs a new vital reading for the authenticated user.
func (h *VitalsHandler) CreateVitalReading(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateVitalReadingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !hasAnyVitalValue(req) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one vital measurement is required"})
		return
	}

	reading := &db.VitalReading{
		UserID:                 userID,
		RecordedAt:             req.RecordedAt,
		BloodPressureSystolic:  req.BloodPressureSystolic,
		BloodPressureDiastolic: req.BloodPressureDiastolic,
		WeightKg:               req.WeightKg,
		HeartRateBpm:           req.HeartRateBpm,
		TemperatureCelsius:     req.TemperatureCelsius,
		FundalHeightCm:         req.FundalHeightCm,
		FetalHeartRateBpm:      req.FetalHeartRateBpm,
		GestationalAgeWeeks:    req.GestationalAgeWeeks,
		Notes:                  req.Notes,
		Source:                 "manual",
	}

	if err := h.db.CreateVitalReading(c.Request.Context(), reading); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save vital reading"})
		return
	}

	c.JSON(http.StatusCreated, vitalReadingToResponse(reading))
}

// DeleteVitalReading removes a vital reading owned by the user.
func (h *VitalsHandler) DeleteVitalReading(c *gin.Context) {
	userID := middleware.GetUserID(c)
	readingID := c.Param("id")

	reading, err := h.db.GetVitalReadingByID(c.Request.Context(), readingID)
	if err != nil || reading == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vital reading not found"})
		return
	}
	if reading.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.db.DeleteVitalReading(c.Request.Context(), readingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vital reading"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vital reading deleted successfully"})
}

func hasAnyVitalValue(req CreateVitalReadingRequest) bool {
	return req.BloodPressureSystolic != nil ||
		req.BloodPressureDiastolic != nil ||
		req.WeightKg != nil ||
		req.HeartRateBpm != nil ||
		req.TemperatureCelsius != nil ||
		req.FundalHeightCm != nil ||
		req.FetalHeartRateBpm != nil ||
		req.GestationalAgeWeeks != nil
}

func vitalReadingToResponse(reading *db.VitalReading) VitalReadingResponse {
	return VitalReadingResponse{
		ID:                     reading.ID,
		UserID:                 reading.UserID,
		RecordedAt:             reading.RecordedAt,
		BloodPressureSystolic:  reading.BloodPressureSystolic,
		BloodPressureDiastolic: reading.BloodPressureDiastolic,
		WeightKg:               reading.WeightKg,
		HeartRateBpm:           reading.HeartRateBpm,
		TemperatureCelsius:     reading.TemperatureCelsius,
		FundalHeightCm:         reading.FundalHeightCm,
		FetalHeartRateBpm:      reading.FetalHeartRateBpm,
		GestationalAgeWeeks:    reading.GestationalAgeWeeks,
		Notes:                  derefString(reading.Notes),
		Source:                 reading.Source,
		CreatedAt:              reading.CreatedAt,
		UpdatedAt:              reading.UpdatedAt,
	}
}
