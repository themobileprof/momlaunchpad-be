package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

func TestClampWeek(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 1},
		{1, 1},
		{20, 20},
		{42, 42},
		{50, 42},
	}
	for _, tt := range tests {
		if got := clampWeek(tt.in); got != tt.want {
			t.Errorf("clampWeek(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestResolvePregnancyTiming_FromWeek(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	week := 20

	gotWeek, edd, start, err := resolvePregnancyTiming(&week, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if gotWeek != 20 {
		t.Fatalf("week = %d", gotWeek)
	}
	if edd.Before(now) {
		t.Fatalf("EDD should be in the future, got %v", edd)
	}
	if start.After(now) {
		t.Fatalf("start date should be in the past, got %v", start)
	}
}

func TestResolvePregnancyTiming_FromEDD(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	edd := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	gotWeek, gotEDD, _, err := resolvePregnancyTiming(nil, &edd, now)
	if err != nil {
		t.Fatal(err)
	}
	if !gotEDD.Equal(edd) {
		t.Fatalf("EDD mismatch: %v", gotEDD)
	}
	if gotWeek < 1 || gotWeek > 42 {
		t.Fatalf("unexpected week: %d", gotWeek)
	}
}

func TestBuildProfileResponse(t *testing.T) {
	week := 24
	concern := "sleep"
	name := "Sarah"
	completed := time.Now()
	user := &db.User{
		ID:                    "user-1",
		Email:                 "sarah@example.com",
		Name:                  &name,
		Language:              "en",
		PregnancyWeek:         &week,
		PrimaryConcern:        &concern,
		OnboardingCompletedAt: &completed,
	}

	facts := []db.UserFact{
		{Key: "pregnancy_week", Value: "24", Confidence: profileFactConfidence},
		{Key: "favorite_food", Value: "mango", Confidence: 0.5},
	}

	resp := buildProfileResponse(user, facts)
	if resp.Name != "Sarah" || !resp.OnboardingCompleted {
		t.Fatalf("unexpected profile: %+v", resp)
	}
	if resp.PregnancyWeek == nil || *resp.PregnancyWeek != 24 {
		t.Fatalf("expected pregnancy week 24")
	}
	if _, ok := resp.LearnedFacts["favorite_food"]; !ok {
		t.Fatalf("expected learned fact, got %+v", resp.LearnedFacts)
	}
	if _, ok := resp.Facts["pregnancy_week"]; !ok {
		t.Fatalf("expected structured fact in Facts map")
	}
}

func TestProfileCompleteOnboarding_RequiresPregnancyTiming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := ginWithUserID("user-1")
	r.PUT("/onboarding", NewProfileHandler(database).CompleteOnboarding)

	req, err := jsonRequest(http.MethodPut, "/onboarding", map[string]string{"name": "Sarah"})
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

func TestProfileGetProfile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "missing-user"

	mock.ExpectQuery(`FROM users`).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	r := ginWithUserID(userID)
	r.GET("/profile", NewProfileHandler(database).GetProfile)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProfileGetProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"

	mock.ExpectQuery(`FROM users`).
		WithArgs(userID).
		WillReturnRows(mockUserRows(userID, "user@example.com"))
	mock.ExpectQuery(`FROM user_facts`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "key", "value", "confidence", "created_at", "updated_at"}))

	r := ginWithUserID(userID)
	r.GET("/profile", NewProfileHandler(database).GetProfile)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
