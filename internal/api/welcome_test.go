package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/welcome"
)

func TestGetWelcome_ReturnsCachedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	cacheDate := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now()

	mock.ExpectQuery(`FROM user_welcome_messages`).
		WithArgs(userID, cacheDate.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "cache_date", "message", "source", "created_at"}).
			AddRow("wm-1", userID, cacheDate, "Good morning!", "gemini", now))

	svc := welcome.NewService(database, nil, nil)
	handler := NewWelcomeHandler(svc)

	r := ginWithUserID(userID)
	r.GET("/welcome", handler.GetWelcome)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/welcome", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message   string `json:"message"`
		Source    string `json:"source"`
		CacheDate string `json:"cache_date"`
	}
	decodeJSONBody(t, w, &resp)
	if resp.Message != "Good morning!" || resp.Source != "cache" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
