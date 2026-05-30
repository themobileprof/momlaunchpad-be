package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &JWTClaims{
		UserID:  "user-1",
		Email:   "user@example.com",
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(JWTAuth(secret))
	r.GET("/protected", func(c *gin.Context) {
		if GetUserID(c) != "user-1" {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(JWTAuth("secret"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAdminOnly_RequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("is_admin", false)
		c.Next()
	})
	r.Use(AdminOnly())
	r.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d", w.Code)
	}
}
