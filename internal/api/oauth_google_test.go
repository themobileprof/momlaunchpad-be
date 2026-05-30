package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetGoogleUserInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "test-token" {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(GoogleUserInfo{
			ID:            "g-1",
			Email:         "user@example.com",
			VerifiedEmail: true,
			Name:          "User",
		})
	}))
	t.Cleanup(server.Close)

	prev := googleUserInfoURL
	googleUserInfoURL = func(accessToken string) string {
		return server.URL + "?access_token=" + accessToken
	}
	t.Cleanup(func() { googleUserInfoURL = prev })

	h := &OAuthHandler{}
	info, err := h.getGoogleUserInfo("test-token")
	if err != nil {
		t.Fatal(err)
	}
	if info.Email != "user@example.com" || !info.VerifiedEmail {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestGoogleCallback_InvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock := newMockDB(t)

	r := gin.New()
	r.GET("/callback", NewOAuthHandler(database).GoogleCallback)

	req := httptest.NewRequest(http.MethodGet, "/callback?state=bad&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "good"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
