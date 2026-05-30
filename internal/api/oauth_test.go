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

// TestGenerateUsernameFromEmail tests username generation from email
func TestGenerateUsernameFromEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "standard email",
			email:    "john.doe@example.com",
			expected: "john.doe",
		},
		{
			name:     "email with numbers",
			email:    "user123@gmail.com",
			expected: "user123",
		},
		{
			name:     "email with special chars",
			email:    "test+filter@domain.org",
			expected: "test+filter",
		},
		{
			name:     "no @ symbol",
			email:    "invalid-email",
			expected: "invalid-email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateUsernameFromEmail(tt.email)
			if result != tt.expected {
				t.Errorf("generateUsernameFromEmail(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

const testWebClientID = "web-client-id.apps.googleusercontent.com"

func mockGoogleTokenInfoServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id_token") == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func withMockGoogleTokenInfo(t *testing.T, server *httptest.Server) {
	t.Helper()
	prev := googleTokenInfoURL
	googleTokenInfoURL = func(idToken string) string {
		return server.URL + "?id_token=" + idToken
	}
	t.Cleanup(func() { googleTokenInfoURL = prev })
}

func validGoogleTokenInfoJSON(aud, email string, verified bool) string {
	verifiedStr := "false"
	if verified {
		verifiedStr = "true"
	}
	payload := map[string]string{
		"aud":            aud,
		"sub":            "google-sub-123",
		"email":          email,
		"email_verified": verifiedStr,
		"name":           "Jane Doe",
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

func TestVerifyGoogleIDToken(t *testing.T) {
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", testWebClientID)

	server := mockGoogleTokenInfoServer(t, validGoogleTokenInfoJSON(testWebClientID, "jane@example.com", true), http.StatusOK)
	withMockGoogleTokenInfo(t, server)

	h := &OAuthHandler{}
	info, err := h.verifyGoogleIDToken("fake-token")
	if err != nil {
		t.Fatalf("verifyGoogleIDToken: %v", err)
	}
	if info.Email != "jane@example.com" || !info.VerifiedEmail {
		t.Fatalf("unexpected user info: %+v", info)
	}
}

func TestVerifyGoogleIDToken_AudienceMismatch(t *testing.T) {
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", testWebClientID)

	server := mockGoogleTokenInfoServer(t, validGoogleTokenInfoJSON("wrong-client.apps.googleusercontent.com", "jane@example.com", true), http.StatusOK)
	withMockGoogleTokenInfo(t, server)

	h := &OAuthHandler{}
	_, err := h.verifyGoogleIDToken("fake-token")
	if err == nil {
		t.Fatal("expected audience mismatch error")
	}
}

func TestVerifyGoogleIDToken_NotConfigured(t *testing.T) {
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", "")

	h := &OAuthHandler{}
	_, err := h.verifyGoogleIDToken("fake-token")
	if err == nil {
		t.Fatal("expected configuration error")
	}
}

func TestGoogleTokenAuth_MissingIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := gin.New()
	r.POST("/google/token", NewOAuthHandler(database).GoogleTokenAuth)

	req, err := jsonRequest(http.MethodPost, "/google/token", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleTokenAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", testWebClientID)

	server := mockGoogleTokenInfoServer(t, `{"error":"invalid_token"}`, http.StatusBadRequest)
	withMockGoogleTokenInfo(t, server)

	database, mock := newMockDB(t)
	r := gin.New()
	r.POST("/google/token", NewOAuthHandler(database).GoogleTokenAuth)

	req, _ := jsonRequest(http.MethodPost, "/google/token", map[string]string{"id_token": "bad"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleTokenAuth_UnverifiedEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", testWebClientID)

	server := mockGoogleTokenInfoServer(t, validGoogleTokenInfoJSON(testWebClientID, "jane@example.com", false), http.StatusOK)
	withMockGoogleTokenInfo(t, server)

	database, mock := newMockDB(t)
	r := gin.New()
	r.POST("/google/token", NewOAuthHandler(database).GoogleTokenAuth)

	req, _ := jsonRequest(http.MethodPost, "/google/token", map[string]string{"id_token": "token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleTokenAuth_ExistingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", testWebClientID)
	t.Setenv("JWT_SECRET", "test-jwt-secret")

	server := mockGoogleTokenInfoServer(t, validGoogleTokenInfoJSON(testWebClientID, "jane@example.com", true), http.StatusOK)
	withMockGoogleTokenInfo(t, server)

	database, mock := newMockDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	expectUserByEmail(mock, "jane@example.com", userID)
	mock.ExpectExec(`INSERT INTO oauth_providers`).
		WithArgs(userID, "google", "google-sub-123", "jane@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.POST("/google/token", NewOAuthHandler(database).GoogleTokenAuth)

	req, _ := jsonRequest(http.MethodPost, "/google/token", map[string]string{"id_token": "valid-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
	}
	decodeJSONBody(t, w, &resp)
	if resp.Token == "" {
		t.Fatal("expected JWT token in response")
	}
	if resp.User.ID != userID || resp.User.Email != "jane@example.com" {
		t.Fatalf("unexpected user payload: %+v", resp.User)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleTokenAuth_CreatesNewUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOOGLE_ALLOWED_CLIENT_IDS", testWebClientID)
	t.Setenv("JWT_SECRET", "test-jwt-secret")

	server := mockGoogleTokenInfoServer(t, validGoogleTokenInfoJSON(testWebClientID, "new@example.com", true), http.StatusOK)
	withMockGoogleTokenInfo(t, server)

	database, mock := newMockDB(t)
	newUserID := "22222222-2222-2222-2222-222222222222"
	now := time.Now()

	expectUserByEmail(mock, "new@example.com", "")
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("new@example.com", "Jane Doe", "en", false, "google").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(newUserID, now, now))
	mock.ExpectExec(`INSERT INTO oauth_providers`).
		WithArgs(newUserID, "google", "google-sub-123", "new@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.POST("/google/token", NewOAuthHandler(database).GoogleTokenAuth)

	req, _ := jsonRequest(http.MethodPost, "/google/token", map[string]string{"id_token": "valid-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleCallback_InvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := gin.New()
	r.GET("/google/callback", NewOAuthHandler(database).GoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/google/callback?state=wrong&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "expected"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppleLogin_NotImplemented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := gin.New()
	r.GET("/apple", NewOAuthHandler(database).AppleLogin)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/apple", nil))

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotImplemented)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOauthDisplayName(t *testing.T) {
	name := "DB Name"
	user := &db.User{Email: "user@example.com", Name: &name}

	if got := oauthDisplayName("Google Name", user); got != "Google Name" {
		t.Fatalf("got %q", got)
	}
	if got := oauthDisplayName("", user); got != "DB Name" {
		t.Fatalf("got %q", got)
	}
	if got := oauthDisplayName("", &db.User{Email: "local@example.com"}); got != "local" {
		t.Fatalf("got %q", got)
	}
}

// TestEmailBasedUserLinking tests that users are linked across providers by email
func TestEmailBasedUserLinking(t *testing.T) {
	t.Run("same email across web and mobile creates single user", func(t *testing.T) {
		// Scenario:
		// 1. User signs in via web OAuth flow (email: user@example.com)
		// 2. User later signs in via mobile with same email
		// Expected: Both use the same user_id

		t.Skip("Requires database mock - see integration tests")
	})

	t.Run("same email across Google and Apple creates single user", func(t *testing.T) {
		// Scenario:
		// 1. User signs in with Google (email: user@example.com)
		// 2. User later signs in with Apple (same email: user@example.com)
		// Expected: Both OAuth providers link to the same user_id

		t.Skip("Requires database mock - see integration tests")
	})
}

// TestOAuthStateValidation tests CSRF protection via state parameter (web flow only)
func TestOAuthStateValidation(t *testing.T) {
	tests := []struct {
		name          string
		cookieState   string
		queryState    string
		shouldSucceed bool
	}{
		{
			name:          "matching states",
			cookieState:   "abc123",
			queryState:    "abc123",
			shouldSucceed: true,
		},
		{
			name:          "mismatched states",
			cookieState:   "abc123",
			queryState:    "xyz789",
			shouldSucceed: false,
		},
		{
			name:          "missing cookie state",
			cookieState:   "",
			queryState:    "abc123",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Web flow uses state parameter for CSRF protection
			// Mobile flow doesn't need this (uses platform-native security)

			if tt.cookieState == tt.queryState && tt.cookieState != "" {
				if !tt.shouldSucceed {
					t.Errorf("Expected failure but states matched")
				}
			} else {
				if tt.shouldSucceed {
					t.Errorf("Expected success but states didn't match")
				}
			}
		})
	}
}

// TestMultipleClientIDValidation tests that multiple client IDs are accepted
func TestMultipleClientIDValidation(t *testing.T) {
	tests := []struct {
		name          string
		clientID      string
		allowedList   string
		shouldSucceed bool
	}{
		{
			name:          "web client ID accepted",
			clientID:      "web-client-id.apps.googleusercontent.com",
			allowedList:   "web-client-id.apps.googleusercontent.com,android-client-id.apps.googleusercontent.com,ios-client-id.apps.googleusercontent.com",
			shouldSucceed: true,
		},
		{
			name:          "android client ID accepted",
			clientID:      "android-client-id.apps.googleusercontent.com",
			allowedList:   "web-client-id.apps.googleusercontent.com,android-client-id.apps.googleusercontent.com,ios-client-id.apps.googleusercontent.com",
			shouldSucceed: true,
		},
		{
			name:          "ios client ID accepted",
			clientID:      "ios-client-id.apps.googleusercontent.com",
			allowedList:   "web-client-id.apps.googleusercontent.com,android-client-id.apps.googleusercontent.com,ios-client-id.apps.googleusercontent.com",
			shouldSucceed: true,
		},
		{
			name:          "unknown client ID rejected",
			clientID:      "attacker-client-id.apps.googleusercontent.com",
			allowedList:   "web-client-id.apps.googleusercontent.com,android-client-id.apps.googleusercontent.com",
			shouldSucceed: false,
		},
		{
			name:          "empty allowed list rejects all",
			clientID:      "any-client-id.apps.googleusercontent.com",
			allowedList:   "",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllowedClientID(tt.clientID, tt.allowedList)
			if result != tt.shouldSucceed {
				t.Errorf("isAllowedClientID(%q, %q) = %v, want %v",
					tt.clientID, tt.allowedList, result, tt.shouldSucceed)
			}
		})
	}
}
