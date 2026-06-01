package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
)

func ginAdmin() *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "admin-1")
		c.Set("is_admin", true)
		c.Next()
	})
	return r
}

func TestAdminCreatePlan_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO plans`).
		WithArgs("pro", "Pro Plan", "Best value").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "description", "active", "created_at"}).
			AddRow(1, "pro", "Pro Plan", "Best value", true, now))

	r := ginAdmin()
	h := NewAdminHandler(database, language.NewManager())
	r.POST("/plans", h.CreatePlan)

	req, _ := jsonRequest(http.MethodPost, "/plans", map[string]string{
		"code": "pro", "name": "Pro Plan", "description": "Best value",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCreatePlan_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, _ := newMockDB(t)

	r := ginAdmin()
	r.POST("/plans", NewAdminHandler(database, language.NewManager()).CreatePlan)

	req, _ := jsonRequest(http.MethodPost, "/plans", map[string]string{"code": "pro"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAdminCreatePlan_Duplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO plans`).
		WithArgs("free", "Free", "").
		WillReturnError(&duplicateKeyError{})

	r := ginAdmin()
	r.POST("/plans", NewAdminHandler(database, language.NewManager()).CreatePlan)

	req, _ := jsonRequest(http.MethodPost, "/plans", map[string]string{"code": "free", "name": "Free"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type duplicateKeyError struct{}

func (e *duplicateKeyError) Error() string {
	return `duplicate key value violates unique constraint "plans_code_key"`
}

func TestAdminListFeatures_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`FROM features`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "feature_key", "name", "description", "created_at"}).
			AddRow(1, "chat", "Chat", "AI chat", now))

	r := ginAdmin()
	r.GET("/features", NewAdminHandler(database, language.NewManager()).ListFeatures)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/features", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCreateFeature_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO features`).
		WithArgs("voice", "Voice", "Voice calls").
		WillReturnRows(sqlmock.NewRows([]string{"id", "feature_key", "name", "description", "created_at"}).
			AddRow(2, "voice", "Voice", "Voice calls", now))

	r := ginAdmin()
	r.POST("/features", NewAdminHandler(database, language.NewManager()).CreateFeature)

	req, _ := jsonRequest(http.MethodPost, "/features", map[string]string{
		"feature_key": "voice", "name": "Voice", "description": "Voice calls",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminAssignFeatureToPlan_InvalidPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, _ := newMockDB(t)

	r := ginAdmin()
	r.POST("/plans/:planId/features/:featureId", NewAdminHandler(database, language.NewManager()).AssignFeatureToPlan)

	req, _ := jsonRequest(http.MethodPost, "/plans/1/features/2", map[string]string{
		"quota_period": "yearly",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeleteLanguage_BlocksEnglish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, _ := newMockDB(t)

	r := ginAdmin()
	r.DELETE("/languages/:code", NewAdminHandler(database, language.NewManager()).DeleteLanguage)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/languages/en", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestAdminCreateLanguage_SyncsManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()
	langMgr := language.NewManager()

	mock.ExpectQuery(`INSERT INTO languages`).
		WithArgs("fr", "French", "Français", true, false).
		WillReturnRows(sqlmock.NewRows([]string{"code", "name", "native_name", "is_enabled", "is_experimental", "created_at"}).
			AddRow("fr", "French", "Français", true, false, now))

	r := ginAdmin()
	r.POST("/languages", NewAdminHandler(database, langMgr).CreateLanguage)

	req, _ := jsonRequest(http.MethodPost, "/languages", map[string]any{
		"code": "fr", "name": "French", "native_name": "Français", "is_enabled": true,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if !langMgr.IsSupported("fr") {
		t.Fatal("language manager was not updated after create")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminGetSystemSettings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()
	desc := "AI name"

	mock.ExpectQuery(`FROM system_settings`).
		WillReturnRows(sqlmock.NewRows([]string{"key", "value", "description", "updated_at"}).
			AddRow("ai_name", "Maya", desc, now))

	r := ginAdmin()
	r.GET("/settings", NewAdminHandler(database, language.NewManager()).GetSystemSettings)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/settings", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminUpdateSystemSetting_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE system_settings`).
		WithArgs("ai_name", "Nova").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := ginAdmin()
	r.PUT("/settings/:key", NewAdminHandler(database, language.NewManager()).UpdateSystemSetting)

	req, _ := jsonRequest(http.MethodPut, "/settings/ai_name", map[string]string{"value": "Nova"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminUpdatePlan_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE plans`).
		WithArgs(99, "Name", "", nil).
		WillReturnResult(sqlmock.NewResult(0, 0))

	r := ginAdmin()
	r.PUT("/plans/:planId", NewAdminHandler(database, language.NewManager()).UpdatePlan)

	req, _ := jsonRequest(http.MethodPut, "/plans/99", map[string]string{"name": "Name"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminGetUserStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery(`COUNT\(DISTINCT user_id\) FROM messages WHERE created_at >= NOW\(\) - INTERVAL '7 days'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`COUNT\(DISTINCT user_id\) FROM messages WHERE created_at >= NOW\(\) - INTERVAL '30 days'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	mock.ExpectQuery(`FROM subscriptions s`).
		WillReturnRows(sqlmock.NewRows([]string{"code", "count"}).AddRow("free", 8))
	mock.ExpectQuery(`FROM users GROUP BY preferred_language`).
		WillReturnRows(sqlmock.NewRows([]string{"language", "count"}).AddRow("en", 9))

	r := ginAdmin()
	r.GET("/analytics/users", NewAdminHandler(database, language.NewManager()).GetUserStats)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/analytics/users", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
