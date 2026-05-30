package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestResolveConversationTitle(t *testing.T) {
	custom := "My chat"
	got := resolveConversationTitle(custom)
	if got == nil || *got != custom {
		t.Fatalf("got %v", got)
	}

	got = resolveConversationTitle("")
	if got == nil || *got != "New Conversation" {
		t.Fatalf("got %v", got)
	}
}

func TestListConversations_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	now := time.Now()

	mock.ExpectQuery(`FROM conversations`).
		WithArgs(userID, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "is_starred", "created_at", "updated_at"}).
			AddRow("conv-1", userID, "Chat", false, now, now))

	r := ginWithUserID(userID)
	h := NewConversationHandler(database)
	r.GET("/conversations", h.ListConversations)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/conversations", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateConversation_DefaultTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO conversations`).
		WithArgs(userID, "New Conversation").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "is_starred", "created_at", "updated_at"}).
			AddRow("conv-1", userID, "New Conversation", false, now, now))

	r := ginWithUserID(userID)
	h := NewConversationHandler(database)
	r.POST("/conversations", h.CreateConversation)

	req, _ := jsonRequest(http.MethodPost, "/conversations", map[string]string{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetConversation_NotOwned(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	ownerID := "owner-id"
	otherID := "other-id"
	now := time.Now()

	mock.ExpectQuery(`FROM conversations`).
		WithArgs("conv-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "is_starred", "created_at", "updated_at"}).
			AddRow("conv-1", ownerID, "Chat", false, now, now))

	r := ginWithUserID(otherID)
	h := NewConversationHandler(database)
	r.GET("/conversations/:id", h.GetConversation)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/conversations/conv-1", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateConversation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	now := time.Now()
	newTitle := "Renamed"

	mock.ExpectQuery(`FROM conversations`).
		WithArgs("conv-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "is_starred", "created_at", "updated_at"}).
			AddRow("conv-1", userID, "Chat", false, now, now))
	mock.ExpectQuery(`UPDATE conversations`).
		WithArgs(newTitle, "conv-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "is_starred", "created_at", "updated_at"}).
			AddRow("conv-1", userID, newTitle, false, now, now))

	r := ginWithUserID(userID)
	h := NewConversationHandler(database)
	r.PATCH("/conversations/:id", h.UpdateConversation)

	req, _ := jsonRequest(http.MethodPatch, "/conversations/conv-1", map[string]string{"title": newTitle})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteConversation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	now := time.Now()

	mock.ExpectQuery(`FROM conversations`).
		WithArgs("conv-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "is_starred", "created_at", "updated_at"}).
			AddRow("conv-1", userID, "Chat", false, now, now))
	mock.ExpectExec(`DELETE FROM conversations`).
		WithArgs("conv-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := ginWithUserID(userID)
	h := NewConversationHandler(database)
	r.DELETE("/conversations/:id", h.DeleteConversation)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/conversations/conv-1", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
