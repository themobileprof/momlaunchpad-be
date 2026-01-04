package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuthConfig holds OAuth provider configurations
type OAuthConfig struct {
	GoogleConfig *oauth2.Config
	AppleConfig  *oauth2.Config // Future implementation
}

// GoogleUserInfo represents user data from Google OAuth
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// AppleUserInfo represents user data from Apple OAuth (future)
type AppleUserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// OAuthHandler handles OAuth authentication flows
type OAuthHandler struct {
	db     *db.DB
	config *OAuthConfig
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(database *db.DB) *OAuthHandler {
	// Web OAuth client for redirect flow
	googleConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_WEB_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthHandler{
		db: database,
		config: &OAuthConfig{
			GoogleConfig: googleConfig,
		},
	}
}

// GoogleLogin initiates Google OAuth flow
func (h *OAuthHandler) GoogleLogin(c *gin.Context) {
	// Generate random state for CSRF protection
	state := generateRandomState()

	// Store state in session/cookie for verification (production should use Redis/session store)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	url := h.config.GoogleConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles Google OAuth callback
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	// Verify state for CSRF protection
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil || c.Query("state") != stateCookie {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// Clear state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Exchange code for token
	code := c.Query("code")
	token, err := h.config.GoogleConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from Google
	userInfo, err := h.getGoogleUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	// Verify email is confirmed
	if !userInfo.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified with Google"})
		return
	}

	// Find or create user based on email (email is the canonical identifier)
	user, err := h.findOrCreateUserByEmail(userInfo.Email, "google", userInfo.ID, userInfo.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate user"})
		return
	}

	// Generate JWT token
	jwtToken, err := h.generateJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	})
}

// GoogleTokenAuth handles Google ID token authentication from mobile apps
// Flutter/mobile apps use google_sign_in package which returns an ID token
// This endpoint verifies the token and returns a JWT
func (h *OAuthHandler) GoogleTokenAuth(c *gin.Context) {
	var req struct {
		IDToken string `json:"id_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID token is required"})
		return
	}

	// Verify the ID token with Google
	userInfo, err := h.verifyGoogleIDToken(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid ID token"})
		return
	}

	// Verify email is confirmed
	if !userInfo.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified with Google"})
		return
	}

	// Find or create user based on email (same logic as web flow)
	user, err := h.findOrCreateUserByEmail(userInfo.Email, "google", userInfo.ID, userInfo.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate user"})
		return
	}

	// Generate JWT token
	jwtToken, err := h.generateJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	})
}

// AppleLogin initiates Apple OAuth flow (future implementation)
func (h *OAuthHandler) AppleLogin(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Apple Sign-In coming soon",
	})
}

// AppleCallback handles Apple OAuth callback (future implementation)
func (h *OAuthHandler) AppleCallback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Apple Sign-In coming soon",
	})
}

// getGoogleUserInfo fetches user information from Google
func (h *OAuthHandler) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

// User represents a user in the system
type User struct {
	ID       string
	Username string
	Email    string
}

// findOrCreateUserByEmail finds existing user by email or creates new one
// This implements email-based user linking across providers (Google, Apple, etc.)
func (h *OAuthHandler) findOrCreateUserByEmail(email, provider, providerUserID, name string) (*User, error) {
	tx, err := h.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Look for existing user by email (canonical identifier across all providers)
	var user User
	err = tx.QueryRow(`
		SELECT id, username, email 
		FROM users 
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Username, &user.Email)

	if err == sql.ErrNoRows {
		// No user exists with this email - create new user
		username := generateUsernameFromEmail(email)
		err = tx.QueryRow(`
			INSERT INTO users (username, email, auth_provider, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
			RETURNING id, username, email
		`, username, email, provider).Scan(&user.ID, &user.Username, &user.Email)

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Link OAuth provider to user (if not already linked)
	_, err = tx.Exec(`
		INSERT INTO oauth_providers (user_id, provider, provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (provider, provider_user_id) DO UPDATE
		SET updated_at = NOW()
	`, user.ID, provider, providerUserID, email)

	if err != nil {
		return nil, fmt.Errorf("failed to link OAuth provider: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &user, nil
}

// generateJWT creates a JWT token for authenticated user
func (h *OAuthHandler) generateJWT(userID, email string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET not configured")
	}

	expiryStr := os.Getenv("JWT_EXPIRY")
	if expiryStr == "" {
		expiryStr = "24h"
	}

	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		expiry = 24 * time.Hour
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(expiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// generateRandomState generates a random state string for CSRF protection
func generateRandomState() string {
	// In production, use crypto/rand for secure random generation
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// generateUsernameFromEmail creates a username from email
func generateUsernameFromEmail(email string) string {
	// Simple implementation - extract local part of email
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}

// verifyGoogleIDToken verifies a Google ID token from mobile apps
// This uses Google's tokeninfo endpoint to validate the token
// Accepts tokens from multiple client IDs (web, Android, iOS)
func (h *OAuthHandler) verifyGoogleIDToken(idToken string) (*GoogleUserInfo, error) {
	// Get all allowed client IDs (comma-separated)
	allowedClientIDs := os.Getenv("GOOGLE_ALLOWED_CLIENT_IDS")
	if allowedClientIDs == "" {
		return nil, fmt.Errorf("GOOGLE_ALLOWED_CLIENT_IDS not configured")
	}

	// Call Google's tokeninfo endpoint to verify the token
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token verification failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse token info response
	var tokenInfo struct {
		Aud           string `json:"aud"` // Audience (should match one of our client IDs)
		Sub           string `json:"sub"` // User ID
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"` // "true" or "false" as string
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Exp           string `json:"exp"` // Expiration timestamp
	}

	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	// Verify the token is for one of our app's client IDs (web, Android, or iOS)
	if !isAllowedClientID(tokenInfo.Aud, allowedClientIDs) {
		return nil, fmt.Errorf("token audience mismatch: got %s", tokenInfo.Aud)
	}

	// Convert email_verified from string to bool
	emailVerified := tokenInfo.EmailVerified == "true"

	return &GoogleUserInfo{
		ID:            tokenInfo.Sub,
		Email:         tokenInfo.Email,
		VerifiedEmail: emailVerified,
		Name:          tokenInfo.Name,
		Picture:       tokenInfo.Picture,
	}, nil
}

// isAllowedClientID checks if a client ID is in the allowed list
func isAllowedClientID(clientID, allowedList string) bool {
	// Split comma-separated list and check each
	for _, allowed := range splitAndTrim(allowedList, ",") {
		if clientID == allowed {
			return true
		}
	}
	return false
}

// splitAndTrim splits a string by delimiter and trims whitespace
func splitAndTrim(s, delimiter string) []string {
	parts := []string{}
	for _, part := range splitString(s, delimiter) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by delimiter
func splitString(s, delimiter string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for _, c := range s {
		if string(c) == delimiter {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
