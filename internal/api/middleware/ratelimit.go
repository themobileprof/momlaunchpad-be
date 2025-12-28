package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Limiter tracks rate limits for a single identifier
type Limiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter manages rate limiting for multiple identifiers
type RateLimiter struct {
	limiters map[string]*Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second
// burst: maximum burst size
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*Limiter),
		rate:     r,
		burst:    b,
		cleanup:  5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanupStale()

	return rl
}

// GetLimiter returns the rate limiter for an identifier
func (rl *RateLimiter) GetLimiter(identifier string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[identifier]
	if !exists {
		limiter = &Limiter{
			limiter:  rate.NewLimiter(rl.rate, rl.burst),
			lastSeen: time.Now(),
		}
		rl.limiters[identifier] = limiter
	} else {
		limiter.lastSeen = time.Now()
	}

	return limiter.limiter
}

// cleanupStale removes stale limiters
func (rl *RateLimiter) cleanupStale() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for id, limiter := range rl.limiters {
			if time.Since(limiter.lastSeen) > rl.cleanup {
				delete(rl.limiters, id)
			}
		}
		rl.mu.Unlock()
	}
}

// PerIP creates middleware that rate limits by IP address
func PerIP(requestsPerSecond float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate.Limit(requestsPerSecond), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// PerUser creates middleware that rate limits by user ID
func PerUser(requestsPerSecond float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate.Limit(requestsPerSecond), burst)

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			// Not authenticated, skip user-based limiting
			c.Next()
			return
		}

		if !limiter.GetLimiter(userID.(string)).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please slow down.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// WebSocketLimiter tracks message rate for WebSocket connections
type WebSocketLimiter struct {
	limiter        *rate.Limiter
	messagesPerMin int
	messageCount   int
	windowStart    time.Time
	mu             sync.Mutex
}

// NewWebSocketLimiter creates a limiter for WebSocket messages
func NewWebSocketLimiter(messagesPerMinute int) *WebSocketLimiter {
	return &WebSocketLimiter{
		limiter:        rate.NewLimiter(rate.Limit(messagesPerMinute)/60.0, messagesPerMinute),
		messagesPerMin: messagesPerMinute,
		windowStart:    time.Now(),
	}
}

// Allow checks if a message is allowed
func (wsl *WebSocketLimiter) Allow() bool {
	return wsl.limiter.Allow()
}

// AllowN checks if N messages are allowed
func (wsl *WebSocketLimiter) AllowN(n int) bool {
	return wsl.limiter.AllowN(time.Now(), n)
}
