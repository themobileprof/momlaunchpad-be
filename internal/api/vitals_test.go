package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestHasAnyVitalValue(t *testing.T) {
	systolic := 120
	empty := CreateVitalReadingRequest{RecordedAt: time.Now()}
	withValue := CreateVitalReadingRequest{RecordedAt: time.Now(), BloodPressureSystolic: &systolic}

	if hasAnyVitalValue(empty) {
		t.Fatal("expected false for empty vitals")
	}
	if !hasAnyVitalValue(withValue) {
		t.Fatal("expected true when systolic is set")
	}
}

func TestCreateVitalReading_RequiresMeasurement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := ginWithUserID("user-1")
	r.POST("/vitals", NewVitalsHandler(database).CreateVitalReading)

	req, err := jsonRequest(http.MethodPost, "/vitals", map[string]any{
		"recorded_at": time.Now().Format(time.RFC3339),
	})
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

func TestCreateVitalReading_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	systolic := 118
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO vital_readings`).
		WithArgs(userID, sqlmock.AnyArg(), systolic, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "manual").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("vital-1", now, now))

	r := ginWithUserID(userID)
	r.POST("/vitals", NewVitalsHandler(database).CreateVitalReading)

	req, err := jsonRequest(http.MethodPost, "/vitals", map[string]any{
		"recorded_at":             now.Format(time.RFC3339),
		"blood_pressure_systolic": systolic,
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteVitalReading_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	ownerID := "owner-id"
	otherID := "other-id"
	now := time.Now()

	mock.ExpectQuery(`FROM vital_readings`).
		WithArgs("vital-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "recorded_at", "blood_pressure_systolic", "blood_pressure_diastolic",
			"weight_kg", "heart_rate_bpm", "temperature_celsius", "fundal_height_cm",
			"fetal_heart_rate_bpm", "gestational_age_weeks", "notes", "source", "created_at", "updated_at",
		}).AddRow("vital-1", ownerID, now, nil, nil, nil, nil, nil, nil, nil, nil, nil, "manual", now, now))

	r := ginWithUserID(otherID)
	r.DELETE("/vitals/:id", NewVitalsHandler(database).DeleteVitalReading)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/vitals/vital-1", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListVitalReadings_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"

	mock.ExpectQuery(`FROM vital_readings`).
		WithArgs(userID, 30).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "recorded_at", "blood_pressure_systolic", "blood_pressure_diastolic",
			"weight_kg", "heart_rate_bpm", "temperature_celsius", "fundal_height_cm",
			"fetal_heart_rate_bpm", "gestational_age_weeks", "notes", "source", "created_at", "updated_at",
		}))

	r := ginWithUserID(userID)
	r.GET("/vitals", NewVitalsHandler(database).ListVitalReadings)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/vitals", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
