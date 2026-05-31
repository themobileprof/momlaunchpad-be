package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

var userRowColumns = []string{
	"id", "email", "password_hash", "display_name", "preferred_language", "currency",
	"pregnancy_week", "pregnancy_start_date", "expected_delivery_date",
	"is_first_pregnancy", "primary_concern", "diet_preference",
	"journey_stage", "journey_stage_since", "baby_birth_date", "loss_date",
	"profile_photo_url", "country", "country_code", "state_province", "city",
	"community_onboarding_completed_at",
	"savings_goal", "is_admin", "onboarding_completed_at", "created_at", "updated_at",
}

func newMockDB(t *testing.T) (*db.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB}, mock
}

func mockUserRows(userID, email string) *sqlmock.Rows {
	now := time.Now()
	name := "Test User"
	return sqlmock.NewRows(userRowColumns).AddRow(
		userID, email, "", name, "en", "", nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil,
		nil, false, nil, now, now,
	)
}

func mockUserRowsWithPassword(userID, email, passwordHash string, isAdmin bool) *sqlmock.Rows {
	now := time.Now()
	name := "Test User"
	if isAdmin {
		name = "Admin"
	}
	return sqlmock.NewRows(userRowColumns).AddRow(
		userID, email, passwordHash, name, "en", "", nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil,
		nil, isAdmin, nil, now, now,
	)
}

func ginWithUserID(userID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	return r
}

func jsonRequest(method, url string, body any) (*http.Request, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func decodeJSONBody(t *testing.T, w *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(dest); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func expectUserByEmail(mock sqlmock.Sqlmock, email string, userID string) {
	if userID == "" {
		mock.ExpectQuery(`FROM users`).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)
		return
	}
	mock.ExpectQuery(`FROM users`).
		WithArgs(email).
		WillReturnRows(mockUserRows(userID, email))
}
