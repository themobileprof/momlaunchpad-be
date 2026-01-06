package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubFeatureChecker struct {
	allow bool
	err   error
}

func (s stubFeatureChecker) HasFeature(_ context.Context, _ string, _ string) (bool, error) {
	return s.allow, s.err
}

// TestRequireFeature tests the RequireFeature middleware
func TestRequireFeature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		checker    FeatureChecker
		userID     interface{}
		wantStatus int
	}{
		{
			name:       "allowed",
			checker:    stubFeatureChecker{allow: true},
			userID:     "u1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "forbidden",
			checker:    stubFeatureChecker{allow: false},
			userID:     "u1",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no user id",
			checker:    stubFeatureChecker{allow: true},
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "checker error",
			checker:    stubFeatureChecker{err: errors.New("db")},
			userID:     "u1",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tt.userID != nil {
					c.Set("userID", tt.userID)
				}
				c.Next()
			})
			r.Use(RequireFeature(tt.checker, "chat"))
			r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestQuotaCheckMiddleware tests the quota checking middleware
func TestQuotaCheckMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		featureCode    string
		userID         string
		withinQuota    bool
		wantStatusCode int
	}{
		{
			name:           "within quota - allowed",
			featureCode:    "chat_quota",
			userID:         "user1",
			withinQuota:    true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "quota exceeded - too many requests",
			featureCode:    "chat_quota",
			userID:         "user1",
			withinQuota:    false,
			wantStatusCode: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			r := gin.New()

			if tt.userID != "" {
				r.Use(func(c *gin.Context) {
					c.Set("userID", tt.userID)
					c.Next()
				})
			}

			mockManager := &mockSubManager{
				withinQuota: tt.withinQuota,
			}

			r.Use(CheckQuota(mockManager, tt.featureCode))
			r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// Mock subscription manager

type mockSubManager struct {
	hasFeature  bool
	withinQuota bool
}

func (m *mockSubManager) HasFeature(ctx context.Context, userID, featureCode string) (bool, error) {
	return m.hasFeature, nil
}

func (m *mockSubManager) CheckQuota(ctx context.Context, userID, featureCode string) (bool, error) {
	return m.withinQuota, nil
}

func (m *mockSubManager) IncrementUsage(ctx context.Context, userID, featureCode string) error {
	return nil
}
