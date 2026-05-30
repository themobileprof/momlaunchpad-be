package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// DoctorVisitHandler handles patient and provider visit record endpoints.
type DoctorVisitHandler struct {
	db *db.DB
}

// NewDoctorVisitHandler creates a new doctor visit handler.
func NewDoctorVisitHandler(database *db.DB) *DoctorVisitHandler {
	return &DoctorVisitHandler{db: database}
}

// VisitMedication represents a prescribed medication entry.
type VisitMedication struct {
	Name         string `json:"name"`
	Dosage       string `json:"dosage"`
	Frequency    string `json:"frequency"`
	Route        string `json:"route,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

// VisitLabResult represents a lab test result entry.
type VisitLabResult struct {
	TestName       string `json:"test_name"`
	Result         string `json:"result"`
	Unit           string `json:"unit,omitempty"`
	ReferenceRange string `json:"reference_range,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

// DoctorVisitPayload is the shared body for create/update requests.
type DoctorVisitPayload struct {
	VisitDate              time.Time         `json:"visit_date" binding:"required"`
	VisitType              string            `json:"visit_type" binding:"required"`
	ProviderName           *string           `json:"provider_name"`
	FacilityName           *string           `json:"facility_name"`
	ChiefComplaint         *string           `json:"chief_complaint"`
	ClinicalNotes          *string           `json:"clinical_notes"`
	Diagnosis              *string           `json:"diagnosis"`
	TreatmentPlan          *string           `json:"treatment_plan"`
	FollowUpInstructions   *string           `json:"follow_up_instructions"`
	BloodPressureSystolic  *int              `json:"blood_pressure_systolic"`
	BloodPressureDiastolic *int              `json:"blood_pressure_diastolic"`
	WeightKg               *float64          `json:"weight_kg"`
	HeartRateBpm           *int              `json:"heart_rate_bpm"`
	TemperatureCelsius     *float64          `json:"temperature_celsius"`
	FundalHeightCm         *float64          `json:"fundal_height_cm"`
	FetalHeartRateBpm      *int              `json:"fetal_heart_rate_bpm"`
	GestationalAgeWeeks    *int              `json:"gestational_age_weeks"`
	Medications            []VisitMedication `json:"medications"`
	LabResults             []VisitLabResult  `json:"lab_results"`
	NextAppointmentAt      *time.Time        `json:"next_appointment_at"`
	NextAppointmentNotes   *string           `json:"next_appointment_notes"`
}

// CreateDoctorVisitRequest is used by patients to create their own records.
type CreateDoctorVisitRequest struct {
	DoctorVisitPayload
}

// UpdateDoctorVisitRequest supports partial updates for patient-owned records.
type UpdateDoctorVisitRequest struct {
	VisitDate              *time.Time         `json:"visit_date"`
	VisitType              *string            `json:"visit_type"`
	ProviderName           *string            `json:"provider_name"`
	FacilityName           *string            `json:"facility_name"`
	ChiefComplaint         *string            `json:"chief_complaint"`
	ClinicalNotes          *string            `json:"clinical_notes"`
	Diagnosis              *string            `json:"diagnosis"`
	TreatmentPlan          *string            `json:"treatment_plan"`
	FollowUpInstructions   *string            `json:"follow_up_instructions"`
	BloodPressureSystolic  *int               `json:"blood_pressure_systolic"`
	BloodPressureDiastolic *int               `json:"blood_pressure_diastolic"`
	WeightKg               *float64           `json:"weight_kg"`
	HeartRateBpm           *int               `json:"heart_rate_bpm"`
	TemperatureCelsius     *float64           `json:"temperature_celsius"`
	FundalHeightCm         *float64           `json:"fundal_height_cm"`
	FetalHeartRateBpm      *int               `json:"fetal_heart_rate_bpm"`
	GestationalAgeWeeks    *int               `json:"gestational_age_weeks"`
	Medications            *[]VisitMedication `json:"medications"`
	LabResults             *[]VisitLabResult  `json:"lab_results"`
	NextAppointmentAt      *time.Time         `json:"next_appointment_at"`
	NextAppointmentNotes   *string            `json:"next_appointment_notes"`
}

// ProviderCreateDoctorVisitRequest is used when a clinician records a visit for a patient.
type ProviderCreateDoctorVisitRequest struct {
	PatientUserID string `json:"patient_user_id" binding:"required"`
	DoctorVisitPayload
}

// DoctorVisitResponse is the API representation of a visit record.
type DoctorVisitResponse struct {
	ID                     string            `json:"id"`
	UserID                 string            `json:"user_id"`
	VisitDate              time.Time         `json:"visit_date"`
	VisitType              string            `json:"visit_type"`
	ProviderName           string            `json:"provider_name,omitempty"`
	FacilityName           string            `json:"facility_name,omitempty"`
	ChiefComplaint         string            `json:"chief_complaint,omitempty"`
	ClinicalNotes          string            `json:"clinical_notes,omitempty"`
	Diagnosis              string            `json:"diagnosis,omitempty"`
	TreatmentPlan          string            `json:"treatment_plan,omitempty"`
	FollowUpInstructions   string            `json:"follow_up_instructions,omitempty"`
	BloodPressureSystolic  *int              `json:"blood_pressure_systolic,omitempty"`
	BloodPressureDiastolic *int              `json:"blood_pressure_diastolic,omitempty"`
	WeightKg               *float64          `json:"weight_kg,omitempty"`
	HeartRateBpm           *int              `json:"heart_rate_bpm,omitempty"`
	TemperatureCelsius     *float64          `json:"temperature_celsius,omitempty"`
	FundalHeightCm         *float64          `json:"fundal_height_cm,omitempty"`
	FetalHeartRateBpm      *int              `json:"fetal_heart_rate_bpm,omitempty"`
	GestationalAgeWeeks    *int              `json:"gestational_age_weeks,omitempty"`
	Medications            []VisitMedication `json:"medications"`
	LabResults             []VisitLabResult  `json:"lab_results"`
	NextAppointmentAt      *time.Time        `json:"next_appointment_at,omitempty"`
	NextAppointmentNotes   string            `json:"next_appointment_notes,omitempty"`
	RecordedBy             string            `json:"recorded_by"`
	ProviderUserID         string            `json:"provider_user_id,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

// ListVisits returns all visit records for the authenticated patient.
func (h *DoctorVisitHandler) ListVisits(c *gin.Context) {
	userID := middleware.GetUserID(c)

	visits, err := h.db.GetUserDoctorVisits(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch visit records"})
		return
	}

	response := make([]DoctorVisitResponse, 0, len(visits))
	for i := range visits {
		response = append(response, visitToResponse(&visits[i]))
	}

	c.JSON(http.StatusOK, response)
}

// GetVisit returns a single visit owned by the authenticated patient.
func (h *DoctorVisitHandler) GetVisit(c *gin.Context) {
	userID := middleware.GetUserID(c)
	visitID := c.Param("id")

	visit, err := h.db.GetDoctorVisitByID(c.Request.Context(), visitID)
	if err != nil || visit == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Visit record not found"})
		return
	}
	if visit.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, visitToResponse(visit))
}

// CreateVisit lets a patient create their own visit record.
func (h *DoctorVisitHandler) CreateVisit(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateDoctorVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	visit, err := payloadToVisit(userID, req.DoctorVisitPayload, "user", nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.CreateDoctorVisit(c.Request.Context(), visit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create visit record"})
		return
	}

	c.JSON(http.StatusCreated, visitToResponse(visit))
}

// UpdateVisit lets a patient update their own visit record.
func (h *DoctorVisitHandler) UpdateVisit(c *gin.Context) {
	userID := middleware.GetUserID(c)
	visitID := c.Param("id")

	var req UpdateDoctorVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	visit, err := h.db.GetDoctorVisitByID(c.Request.Context(), visitID)
	if err != nil || visit == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Visit record not found"})
		return
	}
	if visit.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	applyVisitUpdates(visit, req)

	if err := h.db.UpdateDoctorVisit(c.Request.Context(), visit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update visit record"})
		return
	}

	c.JSON(http.StatusOK, visitToResponse(visit))
}

// DeleteVisit removes a patient-owned visit record.
func (h *DoctorVisitHandler) DeleteVisit(c *gin.Context) {
	userID := middleware.GetUserID(c)
	visitID := c.Param("id")

	visit, err := h.db.GetDoctorVisitByID(c.Request.Context(), visitID)
	if err != nil || visit == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Visit record not found"})
		return
	}
	if visit.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.db.DeleteDoctorVisit(c.Request.Context(), visitID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete visit record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Visit record deleted successfully"})
}

// ProviderListPatientVisits lists visit records for a patient (clinician portal).
func (h *DoctorVisitHandler) ProviderListPatientVisits(c *gin.Context) {
	patientID := c.Param("patientId")

	visits, err := h.db.GetUserDoctorVisits(c.Request.Context(), patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch visit records"})
		return
	}

	response := make([]DoctorVisitResponse, 0, len(visits))
	for i := range visits {
		response = append(response, visitToResponse(&visits[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_user_id": patientID,
		"visits":          response,
		"count":           len(response),
	})
}

// ProviderCreateVisit lets a clinician create a visit record for a patient.
func (h *DoctorVisitHandler) ProviderCreateVisit(c *gin.Context) {
	providerUserID := middleware.GetUserID(c)

	var req ProviderCreateDoctorVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	visit, err := payloadToVisit(req.PatientUserID, req.DoctorVisitPayload, "provider", &providerUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.CreateDoctorVisit(c.Request.Context(), visit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create visit record"})
		return
	}

	c.JSON(http.StatusCreated, visitToResponse(visit))
}

// ProviderUpdateVisit lets a clinician update any visit record.
func (h *DoctorVisitHandler) ProviderUpdateVisit(c *gin.Context) {
	providerUserID := middleware.GetUserID(c)
	visitID := c.Param("id")

	var req UpdateDoctorVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	visit, err := h.db.GetDoctorVisitByID(c.Request.Context(), visitID)
	if err != nil || visit == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Visit record not found"})
		return
	}

	applyVisitUpdates(visit, req)
	visit.RecordedBy = "provider"
	visit.ProviderUserID = &providerUserID

	if err := h.db.UpdateDoctorVisit(c.Request.Context(), visit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update visit record"})
		return
	}

	c.JSON(http.StatusOK, visitToResponse(visit))
}

// ProviderGetVisit returns a visit record for clinician review.
func (h *DoctorVisitHandler) ProviderGetVisit(c *gin.Context) {
	visitID := c.Param("id")

	visit, err := h.db.GetDoctorVisitByID(c.Request.Context(), visitID)
	if err != nil || visit == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Visit record not found"})
		return
	}

	c.JSON(http.StatusOK, visitToResponse(visit))
}

func payloadToVisit(
	userID string,
	payload DoctorVisitPayload,
	recordedBy string,
	providerUserID *string,
) (*db.DoctorVisit, error) {
	medicationsJSON, err := json.Marshal(payload.Medications)
	if err != nil {
		return nil, err
	}
	if payload.Medications == nil {
		medicationsJSON = []byte("[]")
	}

	labResultsJSON, err := json.Marshal(payload.LabResults)
	if err != nil {
		return nil, err
	}
	if payload.LabResults == nil {
		labResultsJSON = []byte("[]")
	}

	return &db.DoctorVisit{
		UserID:                 userID,
		VisitDate:              payload.VisitDate,
		VisitType:              payload.VisitType,
		ProviderName:           payload.ProviderName,
		FacilityName:           payload.FacilityName,
		ChiefComplaint:         payload.ChiefComplaint,
		ClinicalNotes:          payload.ClinicalNotes,
		Diagnosis:              payload.Diagnosis,
		TreatmentPlan:          payload.TreatmentPlan,
		FollowUpInstructions:   payload.FollowUpInstructions,
		BloodPressureSystolic:  payload.BloodPressureSystolic,
		BloodPressureDiastolic: payload.BloodPressureDiastolic,
		WeightKg:               payload.WeightKg,
		HeartRateBpm:           payload.HeartRateBpm,
		TemperatureCelsius:     payload.TemperatureCelsius,
		FundalHeightCm:         payload.FundalHeightCm,
		FetalHeartRateBpm:      payload.FetalHeartRateBpm,
		GestationalAgeWeeks:    payload.GestationalAgeWeeks,
		Medications:            medicationsJSON,
		LabResults:             labResultsJSON,
		NextAppointmentAt:      payload.NextAppointmentAt,
		NextAppointmentNotes:   payload.NextAppointmentNotes,
		RecordedBy:             recordedBy,
		ProviderUserID:         providerUserID,
	}, nil
}

func applyVisitUpdates(visit *db.DoctorVisit, req UpdateDoctorVisitRequest) {
	if req.VisitDate != nil {
		visit.VisitDate = *req.VisitDate
	}
	if req.VisitType != nil {
		visit.VisitType = *req.VisitType
	}
	if req.ProviderName != nil {
		visit.ProviderName = req.ProviderName
	}
	if req.FacilityName != nil {
		visit.FacilityName = req.FacilityName
	}
	if req.ChiefComplaint != nil {
		visit.ChiefComplaint = req.ChiefComplaint
	}
	if req.ClinicalNotes != nil {
		visit.ClinicalNotes = req.ClinicalNotes
	}
	if req.Diagnosis != nil {
		visit.Diagnosis = req.Diagnosis
	}
	if req.TreatmentPlan != nil {
		visit.TreatmentPlan = req.TreatmentPlan
	}
	if req.FollowUpInstructions != nil {
		visit.FollowUpInstructions = req.FollowUpInstructions
	}
	if req.BloodPressureSystolic != nil {
		visit.BloodPressureSystolic = req.BloodPressureSystolic
	}
	if req.BloodPressureDiastolic != nil {
		visit.BloodPressureDiastolic = req.BloodPressureDiastolic
	}
	if req.WeightKg != nil {
		visit.WeightKg = req.WeightKg
	}
	if req.HeartRateBpm != nil {
		visit.HeartRateBpm = req.HeartRateBpm
	}
	if req.TemperatureCelsius != nil {
		visit.TemperatureCelsius = req.TemperatureCelsius
	}
	if req.FundalHeightCm != nil {
		visit.FundalHeightCm = req.FundalHeightCm
	}
	if req.FetalHeartRateBpm != nil {
		visit.FetalHeartRateBpm = req.FetalHeartRateBpm
	}
	if req.GestationalAgeWeeks != nil {
		visit.GestationalAgeWeeks = req.GestationalAgeWeeks
	}
	if req.Medications != nil {
		if data, err := json.Marshal(*req.Medications); err == nil {
			visit.Medications = data
		}
	}
	if req.LabResults != nil {
		if data, err := json.Marshal(*req.LabResults); err == nil {
			visit.LabResults = data
		}
	}
	if req.NextAppointmentAt != nil {
		visit.NextAppointmentAt = req.NextAppointmentAt
	}
	if req.NextAppointmentNotes != nil {
		visit.NextAppointmentNotes = req.NextAppointmentNotes
	}
}

func visitToResponse(visit *db.DoctorVisit) DoctorVisitResponse {
	medications := make([]VisitMedication, 0)
	if len(visit.Medications) > 0 {
		_ = json.Unmarshal(visit.Medications, &medications)
	}

	labResults := make([]VisitLabResult, 0)
	if len(visit.LabResults) > 0 {
		_ = json.Unmarshal(visit.LabResults, &labResults)
	}

	return DoctorVisitResponse{
		ID:                     visit.ID,
		UserID:                 visit.UserID,
		VisitDate:              visit.VisitDate,
		VisitType:              visit.VisitType,
		ProviderName:           derefString(visit.ProviderName),
		FacilityName:           derefString(visit.FacilityName),
		ChiefComplaint:         derefString(visit.ChiefComplaint),
		ClinicalNotes:          derefString(visit.ClinicalNotes),
		Diagnosis:              derefString(visit.Diagnosis),
		TreatmentPlan:          derefString(visit.TreatmentPlan),
		FollowUpInstructions:   derefString(visit.FollowUpInstructions),
		BloodPressureSystolic:  visit.BloodPressureSystolic,
		BloodPressureDiastolic: visit.BloodPressureDiastolic,
		WeightKg:               visit.WeightKg,
		HeartRateBpm:           visit.HeartRateBpm,
		TemperatureCelsius:     visit.TemperatureCelsius,
		FundalHeightCm:         visit.FundalHeightCm,
		FetalHeartRateBpm:      visit.FetalHeartRateBpm,
		GestationalAgeWeeks:    visit.GestationalAgeWeeks,
		Medications:            medications,
		LabResults:             labResults,
		NextAppointmentAt:      visit.NextAppointmentAt,
		NextAppointmentNotes:   derefString(visit.NextAppointmentNotes),
		RecordedBy:             visit.RecordedBy,
		ProviderUserID:         derefString(visit.ProviderUserID),
		CreatedAt:              visit.CreatedAt,
		UpdatedAt:              visit.UpdatedAt,
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
