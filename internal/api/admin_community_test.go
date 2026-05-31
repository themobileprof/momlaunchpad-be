package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
)

func ginAdminWithUser(userID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("is_admin", true)
		c.Next()
	})
	return r
}

func TestAdminCommunity_ListReports_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectQuery(`FROM community_reports`).
		WithArgs("open", 100).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "target_type", "target_id", "reason", "details",
			"status", "reviewed_by", "reviewed_at", "created_at",
		}).AddRow("report-1", "user-1", "post", "post-1", "spam", nil, "open", nil, nil, now))

	r := ginAdminWithUser("admin-1")
	r.GET("/reports", NewAdminCommunityHandler(database).ListReports)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/reports?status=open", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCommunity_UpdateReport_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, _ := newMockDB(t)

	r := ginAdminWithUser("admin-1")
	r.PUT("/reports/:id", NewAdminCommunityHandler(database).UpdateReport)

	req, _ := jsonRequest(http.MethodPut, "/reports/r1", map[string]string{"status": "invalid"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestAdminCommunity_UpdateReport_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE community_reports`).
		WithArgs("reviewed", "admin-1", "report-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := ginAdminWithUser("admin-1")
	r.PUT("/reports/:id", NewAdminCommunityHandler(database).UpdateReport)

	req, _ := jsonRequest(http.MethodPut, "/reports/report-1", map[string]string{"status": "reviewed"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCommunity_UpdatePostStatus_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, _ := newMockDB(t)

	r := ginAdminWithUser("admin-1")
	r.PUT("/posts/:id/status", NewAdminCommunityHandler(database).UpdatePostStatus)

	req, _ := jsonRequest(http.MethodPut, "/posts/p1/status", map[string]string{"status": "bad"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAdminCommunity_GrantBadge_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectQuery(`FROM community_badge_types`).
		WithArgs("unknown").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r := ginAdminWithUser("admin-1")
	r.POST("/users/:userId/badges", NewAdminCommunityHandler(database).GrantBadge)

	req, _ := jsonRequest(http.MethodPost, "/users/user-1/badges", map[string]string{"badge_type": "unknown"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestAdminCommunity_RevokeBadge_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM community_user_badges`).
		WithArgs("user-1", "verified").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := ginAdminWithUser("admin-1")
	r.DELETE("/users/:userId/badges/:badgeType", NewAdminCommunityHandler(database).RevokeBadge)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/users/user-1/badges/verified", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Ensure middleware import stays linked for reviewer ID extraction.
var _ = middleware.GetUserID
