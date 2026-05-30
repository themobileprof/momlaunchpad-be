package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/profile"
)

const profileFactConfidence = 1.0

var profileFactKeys = map[string]bool{
	"pregnancy_week":     true,
	"is_first_pregnancy": true,
	"primary_concern":    true,
	"diet":               true,
}

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
	PregnancyWeek        *int              `json:"pregnancy_week,omitempty"`
	ExpectedDeliveryDate *time.Time        `json:"expected_delivery_date,omitempty"`
	PregnancyStartDate   *time.Time        `json:"pregnancy_start_date,omitempty"`
	IsFirstPregnancy     *bool             `json:"is_first_pregnancy,omitempty"`
	PrimaryConcern       *string           `json:"primary_concern,omitempty"`
	DietPreference       *string           `json:"diet_preference,omitempty"`
	LearnedFacts         map[string]string `json:"learned_facts,omitempty"`
	Facts                map[string]string `json:"facts,omitempty"`
}

// ProfileSaveRequest captures editable profile fields.
type ProfileSaveRequest struct {
	Name                 string     `json:"name"`
	Language             string     `json:"language"`
	PregnancyWeek        *int       `json:"pregnancy_week"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`
	IsFirstPregnancy     *bool      `json:"is_first_pregnancy"`
	PrimaryConcern       *string    `json:"primary_concern"`
	DietPreference       *string    `json:"diet_preference"`
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

// UpdateProfile saves profile changes from the profile page.
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req ProfileSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, facts, err := h.saveProfile(c, userID, req, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, buildProfileResponse(user, facts))
}

// CompleteOnboarding saves onboarding answers and marks setup complete.
func (h *ProfileHandler) CompleteOnboarding(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req ProfileSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PregnancyWeek == nil && req.ExpectedDeliveryDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pregnancy_week or expected_delivery_date is required"})
		return
	}

	user, facts, err := h.saveProfile(c, userID, req, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, buildProfileResponse(user, facts))
}

func (h *ProfileHandler) saveProfile(
	c *gin.Context,
	userID string,
	req ProfileSaveRequest,
	markComplete bool,
) (*db.User, []db.UserFact, error) {
	ctx := c.Request.Context()
	now := time.Now()

	language := req.Language
	if language == "" {
		language = "en"
	}

	name := strings.TrimSpace(req.Name)
	var namePtr *string
	if name != "" {
		namePtr = &name
	}

	update := db.UserProfileUpdate{
		Name:             namePtr,
		Language:         &language,
		IsFirstPregnancy: req.IsFirstPregnancy,
		PrimaryConcern:   trimOptionalString(req.PrimaryConcern),
		DietPreference:   trimOptionalString(req.DietPreference),
	}

	if req.PregnancyWeek != nil || req.ExpectedDeliveryDate != nil {
		week, edd, startDate, err := resolvePregnancyTiming(req.PregnancyWeek, req.ExpectedDeliveryDate, now)
		if err != nil {
			return nil, nil, err
		}
		update.PregnancyWeek = &week
		update.ExpectedDeliveryDate = &edd
		update.PregnancyStartDate = &startDate
	}

	if err := h.db.UpdateUserProfileDetails(ctx, userID, update); err != nil {
		return nil, nil, err
	}

	if err := h.syncProfileFacts(ctx, userID, update); err != nil {
		return nil, nil, err
	}

	if markComplete {
		if err := h.db.CompleteOnboarding(ctx, userID); err != nil {
			return nil, nil, err
		}
	}

	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	facts, err := h.db.GetUserFacts(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// Profile changes affect welcome personalization — invalidate today's cache.
	y, m, d := now.UTC().Date()
	cacheDate := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if err := h.db.DeleteWelcomeMessage(ctx, userID, cacheDate); err != nil {
		return nil, nil, fmt.Errorf("failed to invalidate welcome message: %w", err)
	}

	return user, facts, nil
}

func (h *ProfileHandler) syncProfileFacts(ctx context.Context, userID string, update db.UserProfileUpdate) error {
	if update.PregnancyWeek != nil {
		if _, err := h.db.SaveOrUpdateFact(ctx, userID, "pregnancy_week", strconv.Itoa(*update.PregnancyWeek), profileFactConfidence); err != nil {
			return err
		}
	}

	if update.IsFirstPregnancy != nil {
		value := "no"
		if *update.IsFirstPregnancy {
			value = "yes"
		}
		if _, err := h.db.SaveOrUpdateFact(ctx, userID, "is_first_pregnancy", value, profileFactConfidence); err != nil {
			return err
		}
	}

	if update.PrimaryConcern != nil && *update.PrimaryConcern != "" {
		if _, err := h.db.SaveOrUpdateFact(ctx, userID, "primary_concern", *update.PrimaryConcern, profileFactConfidence); err != nil {
			return err
		}
	}

	if update.DietPreference != nil && *update.DietPreference != "" {
		if _, err := h.db.SaveOrUpdateFact(ctx, userID, "diet", *update.DietPreference, profileFactConfidence); err != nil {
			return err
		}
	}

	return nil
}

func resolvePregnancyTiming(weekPtr *int, eddPtr *time.Time, now time.Time) (int, time.Time, time.Time, error) {
	switch {
	case weekPtr != nil && eddPtr != nil:
		week := clampWeek(*weekPtr)
		return week, *eddPtr, profile.PregnancyStartFromWeek(week, now), nil
	case weekPtr != nil:
		week := clampWeek(*weekPtr)
		edd := profile.EDDFromWeek(week, now)
		return week, edd, profile.PregnancyStartFromWeek(week, now), nil
	case eddPtr != nil:
		edd := *eddPtr
		week := profile.WeekFromEDD(edd, now)
		return week, edd, profile.PregnancyStartFromWeek(week, now), nil
	default:
		return 0, time.Time{}, time.Time{}, nil
	}
}

func buildProfileResponse(user *db.User, facts []db.UserFact) ProfileResponse {
	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	pregnancyWeek := user.PregnancyWeek
	if pregnancyWeek == nil && user.ExpectedDeliveryDate != nil {
		week := profile.WeekFromEDD(*user.ExpectedDeliveryDate, time.Now())
		pregnancyWeek = &week
	}

	allFacts := mapFacts(facts)
	learnedFacts := map[string]string{}
	for _, fact := range facts {
		if profileFactKeys[fact.Key] && fact.Confidence >= profileFactConfidence {
			continue
		}
		learnedFacts[fact.Key] = fact.Value
	}

	return ProfileResponse{
		Name:                 name,
		Language:             user.Language,
		OnboardingCompleted:  user.OnboardingCompletedAt != nil,
		PregnancyWeek:        pregnancyWeek,
		ExpectedDeliveryDate: user.ExpectedDeliveryDate,
		PregnancyStartDate:   user.PregnancyStartDate,
		IsFirstPregnancy:     user.IsFirstPregnancy,
		PrimaryConcern:       user.PrimaryConcern,
		DietPreference:       user.DietPreference,
		LearnedFacts:         learnedFacts,
		Facts:                allFacts,
	}
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

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
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
