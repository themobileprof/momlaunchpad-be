package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

func TestPayloadToVisit(t *testing.T) {
	visitDate := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	provider := "Dr. Smith"

	visit, err := payloadToVisit("patient-1", DoctorVisitPayload{
		VisitDate:    visitDate,
		VisitType:    "prenatal",
		ProviderName: &provider,
		Medications:  []VisitMedication{{Name: "Iron", Dosage: "65mg", Frequency: "daily"}},
	}, "user", nil)
	if err != nil {
		t.Fatal(err)
	}
	if visit.UserID != "patient-1" || visit.VisitType != "prenatal" || visit.RecordedBy != "user" {
		t.Fatalf("unexpected visit: %+v", visit)
	}

	var meds []VisitMedication
	if err := json.Unmarshal(visit.Medications, &meds); err != nil || len(meds) != 1 {
		t.Fatalf("medications not serialized: %v", err)
	}
}

func TestVisitToResponse(t *testing.T) {
	visitDate := time.Now()
	visit := &db.DoctorVisit{
		ID:        "visit-1",
		UserID:    "patient-1",
		VisitDate: visitDate,
		VisitType: "prenatal",
		Medications: mustJSON(t, []VisitMedication{
			{Name: "Iron", Dosage: "65mg", Frequency: "daily"},
		}),
		LabResults: []byte("[]"),
		RecordedBy: "user",
	}

	resp := visitToResponse(visit)
	if resp.ID != "visit-1" || len(resp.Medications) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCreateVisit_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := ginWithUserID("user-1")
	r.POST("/visits", NewDoctorVisitHandler(database).CreateVisit)

	req, err := jsonRequest(http.MethodPost, "/visits", map[string]string{"visit_type": "prenatal"})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetVisit_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	ownerID := "owner-id"
	otherID := "other-id"
	now := time.Now()

	mock.ExpectQuery(`FROM doctor_visits`).
		WithArgs("visit-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "visit_date", "visit_type", "provider_name", "facility_name",
			"chief_complaint", "clinical_notes", "diagnosis", "treatment_plan", "follow_up_instructions",
			"blood_pressure_systolic", "blood_pressure_diastolic", "weight_kg", "heart_rate_bpm",
			"temperature_celsius", "fundal_height_cm", "fetal_heart_rate_bpm", "gestational_age_weeks",
			"medications", "lab_results", "next_appointment_at", "next_appointment_notes",
			"recorded_by", "provider_user_id", "created_at", "updated_at",
		}).AddRow(
			"visit-1", ownerID, now, "prenatal", nil, nil,
			nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil, nil,
			[]byte("[]"), []byte("[]"), nil, nil,
			"user", nil, now, now,
		))

	r := ginWithUserID(otherID)
	r.GET("/visits/:id", NewDoctorVisitHandler(database).GetVisit)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/visits/visit-1", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyVisitUpdates(t *testing.T) {
	visit := &db.DoctorVisit{VisitType: "prenatal"}
	newType := "follow_up"
	applyVisitUpdates(visit, UpdateDoctorVisitRequest{VisitType: &newType})
	if visit.VisitType != "follow_up" {
		t.Fatalf("visit type = %q", visit.VisitType)
	}
}
