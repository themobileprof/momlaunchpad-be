package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/profile"
)

const onboardingFactConfidence = 1.0

// ProfileHandler handles user profile and onboarding endpoints.
type ProfileHandler struct {
	db *db.DB
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(database *db.DB) *ProfileHandler {
	return &ProfileHandler{db: database}
}

// ProfileResponse is the user's profile and known facts for personalization.
type ProfileResponse struct {
	Name                 string            `json:"name"`
	Language             string            `json:"language"`
	OnboardingCompleted  bool              `json:"onboarding_completed"`
	ExpectedDeliveryDate *time.Time        `json:"expected_delivery_date,omitempty"`
	PregnancyWeek        *int              `json:"pregnancy_week,omitempty"`
	Facts                map[string]string `json:"facts,omitempty"`
}

// CompleteOnboardingRequest captures first-time setup fields.
type CompleteOnboardingRequest struct {
	Name                 string     `json:"name"`
	Language             string     `json:"language"`
	PregnancyWeek        *int       `json:"pregnancy_week"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`
	IsFirstPregnancy     *bool      `json:"is_first_pregnancy"`
	PrimaryConcern       string     `json:"primary_concern"`
}

// GetProfile returns the current user's profile and personalization facts.
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	facts, err := h.db.GetUserFacts(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile facts"})
		return
	}

	c.JSON(http.StatusOK, buildProfileResponse(user, facts))
}

// CompleteOnboarding saves onboarding answers and marks setup complete.
func (h *ProfileHandler) CompleteOnboarding(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CompleteOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PregnancyWeek == nil && req.ExpectedDeliveryDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pregnancy_week or expected_delivery_date is required"})
		return
	}

	now := time.Now()
	var pregnancyWeek int
	var edd time.Time

	switch {
	case req.PregnancyWeek != nil && req.ExpectedDeliveryDate != nil:
		pregnancyWeek = clampWeek(*req.PregnancyWeek)
		edd = *req.ExpectedDeliveryDate
	case req.PregnancyWeek != nil:
		pregnancyWeek = clampWeek(*req.PregnancyWeek)
		edd = profile.EDDFromWeek(pregnancyWeek, now)
	case req.ExpectedDeliveryDate != nil:
		edd = *req.ExpectedDeliveryDate
		pregnancyWeek = profile.WeekFromEDD(edd, now)
	}

	name := req.Name
	language := req.Language
	if language == "" {
		language = "en"
	}

	var namePtr *string
	if name != "" {
		namePtr = &name
	}

	if err := h.db.UpdateUserProfile(c.Request.Context(), userID, namePtr, &language); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	if err := h.db.UpdateUserEDD(c.Request.Context(), userID, &edd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save due date"})
		return
	}

	weekValue := formatWeek(pregnancyWeek)
	if _, err := h.db.SaveOrUpdateFact(c.Request.Context(), userID, "pregnancy_week", weekValue, onboardingFactConfidence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save pregnancy week"})
		return
	}

	if req.IsFirstPregnancy != nil {
		value := "no"
		if *req.IsFirstPregnancy {
			value = "yes"
		}
		if _, err := h.db.SaveOrUpdateFact(c.Request.Context(), userID, "is_first_pregnancy", value, onboardingFactConfidence); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save pregnancy history"})
			return
		}
	}

	if req.PrimaryConcern != "" {
		if _, err := h.db.SaveOrUpdateFact(c.Request.Context(), userID, "primary_concern", req.PrimaryConcern, onboardingFactConfidence); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save primary concern"})
			return
		}
	}

	if err := h.db.CompleteOnboarding(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete onboarding"})
		return
	}

	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated profile"})
		return
	}

	facts, err := h.db.GetUserFacts(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated facts"})
		return
	}

	c.JSON(http.StatusOK, buildProfileResponse(user, facts))
}

func buildProfileResponse(user *db.User, facts []db.UserFact) ProfileResponse {
	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	resp := ProfileResponse{
		Name:                name,
		Language:            user.Language,
		OnboardingCompleted: user.OnboardingCompletedAt != nil,
		ExpectedDeliveryDate: user.ExpectedDeliveryDate,
		Facts:               mapFacts(facts),
	}

	if week := factValue(facts, "pregnancy_week"); week != "" {
		if parsed, ok := parseWeek(week); ok {
			resp.PregnancyWeek = &parsed
		}
	} else if user.ExpectedDeliveryDate != nil {
		week := profile.WeekFromEDD(*user.ExpectedDeliveryDate, time.Now())
		resp.PregnancyWeek = &week
	}

	return resp
}

func mapFacts(facts []db.UserFact) map[string]string {
	if len(facts) == 0 {
		return nil
	}
	out := make(map[string]string, len(facts))
	for _, fact := range facts {
		out[fact.Key] = fact.Value
	}
	return out
}

func factValue(facts []db.UserFact, key string) string {
	for _, fact := range facts {
		if fact.Key == key {
			return fact.Value
		}
	}
	return ""
}

func clampWeek(week int) int {
	if week < 1 {
		return 1
	}
	if week > 42 {
		return 42
	}
	return week
}

func formatWeek(week int) string {
	return strconv.Itoa(week)
}

func parseWeek(value string) (int, bool) {
	week, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return clampWeek(week), true
}
