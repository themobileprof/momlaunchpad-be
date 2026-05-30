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
	"golang.org/x/crypto/bcrypt"
)

func TestAuthRegister_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	now := time.Now()
	userID := "11111111-1111-1111-1111-111111111111"

	mock.ExpectQuery(`FROM users`).
		WithArgs("new@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("new@example.com", sqlmock.AnyArg(), "Jane", "en", false).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(userID, now, now))

	r := gin.New()
	h := NewAuthHandler(database, "test-jwt-secret")
	r.POST("/register", h.Register)

	req, _ := jsonRequest(http.MethodPost, "/register", map[string]string{
		"email":    "new@example.com",
		"password": "password123",
		"name":     "Jane",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	var resp AuthResponse
	decodeJSONBody(t, w, &resp)
	if resp.Token == "" || resp.User.Email != "new@example.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthRegister_DuplicateEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectQuery(`FROM users`).
		WithArgs("exists@example.com").
		WillReturnRows(mockUserRows("user-1", "exists@example.com"))

	r := gin.New()
	h := NewAuthHandler(database, "test-jwt-secret")
	r.POST("/register", h.Register)

	req, _ := jsonRequest(http.MethodPost, "/register", map[string]string{
		"email":    "exists@example.com",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	rows := sqlmock.NewRows(userRowColumns).
		AddRow(userID, "user@example.com", string(hash), "Test User", "en", "", nil, nil, nil, nil, nil, nil, nil, false, nil, time.Now(), time.Now())

	mock.ExpectQuery(`FROM users`).
		WithArgs("user@example.com").
		WillReturnRows(rows)

	r := gin.New()
	h := NewAuthHandler(database, "test-jwt-secret")
	r.POST("/login", h.Login)

	req, _ := jsonRequest(http.MethodPost, "/login", map[string]string{
		"email":    "user@example.com",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthLogin_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	mock.ExpectQuery(`FROM users`).
		WithArgs("missing@example.com").
		WillReturnError(sql.ErrNoRows)

	r := gin.New()
	h := NewAuthHandler(database, "test-jwt-secret")
	r.POST("/login", h.Login)

	req, _ := jsonRequest(http.MethodPost, "/login", map[string]string{
		"email":    "missing@example.com",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthMe_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"

	mock.ExpectQuery(`FROM users`).
		WithArgs(userID).
		WillReturnRows(mockUserRows(userID, "user@example.com"))

	r := ginWithUserID(userID)
	r.GET("/me", NewAuthHandler(database, "test-jwt-secret").Me)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserToUserInfo(t *testing.T) {
	name := "Jane"
	user := mockUserFromRow("id-1", "jane@example.com", &name)
	info := userToUserInfo(user)
	if info.Name != "Jane" || info.Email != "jane@example.com" {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func mockUserFromRow(id, email string, name *string) *db.User {
	return &db.User{ID: id, Email: email, Name: name, Language: "en"}
}
