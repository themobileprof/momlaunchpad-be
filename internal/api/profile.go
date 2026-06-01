package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/profile"
	"github.com/themobileprof/momlaunchpad-be/internal/storage"
)

const profileFactConfidence = 1.0

var profileFactKeys = map[string]bool{
	"journey_stage":      true,
	"pregnancy_week":     true,
	"is_first_pregnancy": true,
	"primary_concern":    true,
	"diet":               true,
}

// ProfileHandler handles user profile and onboarding endpoints.
type ProfileHandler struct {
	db     *db.DB
	photos *storage.ProfilePhotoStore
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(database *db.DB, photos *storage.ProfilePhotoStore) *ProfileHandler {
	return &ProfileHandler{db: database, photos: photos}
}

// ProfileResponse is the user's profile and known facts for personalization.
type ProfileResponse struct {
	Name                         string            `json:"name"`
	Language                     string            `json:"language"`
	OnboardingCompleted          bool              `json:"onboarding_completed"`
	JourneyStage                 string            `json:"journey_stage,omitempty"`
	JourneyStageSince            *time.Time        `json:"journey_stage_since,omitempty"`
	BabyBirthDate                *time.Time        `json:"baby_birth_date,omitempty"`
	LossDate                     *time.Time        `json:"loss_date,omitempty"`
	PregnancyWeek                *int              `json:"pregnancy_week,omitempty"`
	ExpectedDeliveryDate         *time.Time        `json:"expected_delivery_date,omitempty"`
	PregnancyStartDate           *time.Time        `json:"pregnancy_start_date,omitempty"`
	IsFirstPregnancy             *bool             `json:"is_first_pregnancy,omitempty"`
	PrimaryConcern               *string           `json:"primary_concern,omitempty"`
	DietPreference               *string           `json:"diet_preference,omitempty"`
	ProfilePhotoURL              *string           `json:"profile_photo_url,omitempty"`
	Country                      *string           `json:"country,omitempty"`
	CountryCode                  *string           `json:"country_code,omitempty"`
	StateProvince                *string           `json:"state_province,omitempty"`
	City                         *string           `json:"city,omitempty"`
	CommunityOnboardingCompleted bool              `json:"community_onboarding_completed"`
	CommunityInterests           []string          `json:"community_interests,omitempty"`
	LearnedFacts                 map[string]string `json:"learned_facts,omitempty"`
	Facts                        map[string]string `json:"facts,omitempty"`
}

// ProfileSaveRequest captures editable profile fields.
type ProfileSaveRequest struct {
	Name                 string     `json:"name"`
	Language             string     `json:"language"`
	JourneyStage         string     `json:"journey_stage"`
	PregnancyWeek        *int       `json:"pregnancy_week"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`
	BabyBirthDate        *time.Time `json:"baby_birth_date"`
	LossDate             *time.Time `json:"loss_date"`
	IsFirstPregnancy     *bool      `json:"is_first_pregnancy"`
	PrimaryConcern       *string    `json:"primary_concern"`
	DietPreference       *string    `json:"diet_preference"`
	ProfilePhotoURL      *string    `json:"profile_photo_url"`
	Country              *string    `json:"country"`
	CountryCode          *string    `json:"country_code"`
	StateProvince        *string    `json:"state_province"`
	City                 *string    `json:"city"`
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

	c.JSON(http.StatusOK, h.buildProfileResponseWithCommunity(c.Request.Context(), user, facts))
}

// UploadProfilePhoto godoc
// @Summary      Upload profile photo
// @Description  Accepts multipart form field `photo` (JPEG/PNG/WebP, max 5MB). Replaces any previous uploaded photo.
// @Tags         profile
// @Accept       multipart/form-data
// @Produce      json
// @Param        photo formData file true "Profile photo"
// @Success      200 {object} ProfileResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /users/me/profile-photo [post]
func (h *ProfileHandler) UploadProfilePhoto(c *gin.Context) {
	if h.photos == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Photo upload is not configured"})
		return
	}

	userID := middleware.GetUserID(c)
	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Photo file is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read photo"})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, storage.MaxProfilePhotoBytes+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read photo"})
		return
	}
	if len(data) > storage.MaxProfilePhotoBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Photo must be 5MB or smaller"})
		return
	}

	ctx := c.Request.Context()
	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	path, err := h.photos.Save(userID, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.ProfilePhotoURL != nil && storage.IsManagedProfilePhotoPath(*user.ProfilePhotoURL) {
		_ = h.photos.DeleteByPublicPath(*user.ProfilePhotoURL)
	}

	publicURL := publicBaseURL(c) + path
	if err := h.db.UpdateUserProfileDetails(ctx, userID, db.UserProfileUpdate{
		ProfilePhotoURL: &publicURL,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile photo"})
		return
	}

	user, err = h.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile"})
		return
	}
	facts, err := h.db.GetUserFacts(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile facts"})
		return
	}

	c.JSON(http.StatusOK, h.buildProfileResponseWithCommunity(ctx, user, facts))
}

// DeleteProfilePhoto godoc
// @Summary      Remove profile photo
// @Description  Deletes the user's uploaded profile photo if stored on this server.
// @Tags         profile
// @Produce      json
// @Success      200 {object} ProfileResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /users/me/profile-photo [delete]
func (h *ProfileHandler) DeleteProfilePhoto(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.ProfilePhotoURL != nil && h.photos != nil &&
		storage.IsManagedProfilePhotoPath(*user.ProfilePhotoURL) {
		_ = h.photos.DeleteByPublicPath(*user.ProfilePhotoURL)
	}

	empty := ""
	if err := h.db.UpdateUserProfileDetails(ctx, userID, db.UserProfileUpdate{
		ProfilePhotoURL: &empty,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove profile photo"})
		return
	}

	user, err = h.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile"})
		return
	}
	facts, err := h.db.GetUserFacts(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile facts"})
		return
	}

	c.JSON(http.StatusOK, h.buildProfileResponseWithCommunity(ctx, user, facts))
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
		if strings.Contains(err.Error(), "invalid journey transition") ||
			strings.Contains(err.Error(), "journey_stage") ||
			strings.Contains(err.Error(), "baby_birth_date") ||
			strings.Contains(err.Error(), "pregnancy_week") ||
			strings.Contains(err.Error(), "profile photo") ||
			strings.Contains(err.Error(), "image URL") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.buildProfileResponseWithCommunity(c.Request.Context(), user, facts))
}

// CompleteOnboarding saves onboarding answers and marks setup complete.
func (h *ProfileHandler) CompleteOnboarding(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req ProfileSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stage, err := profile.NormalizeStage(req.JourneyStage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.JourneyStage = stage

	if err := profile.ValidateStageProfile(stage, stageSaveInput(req), true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, facts, err := h.saveProfile(c, userID, req, true)
	if err != nil {
		if strings.Contains(err.Error(), "invalid journey transition") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.buildProfileResponseWithCommunity(c.Request.Context(), user, facts))
}

func (h *ProfileHandler) saveProfile(
	c *gin.Context,
	userID string,
	req ProfileSaveRequest,
	markComplete bool,
) (*db.User, []db.UserFact, error) {
	ctx := c.Request.Context()
	now := time.Now()

	currentUser, err := h.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if currentUser == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

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
		BabyBirthDate:    req.BabyBirthDate,
		LossDate:         req.LossDate,
		ProfilePhotoURL:  trimOptionalString(req.ProfilePhotoURL),
		StateProvince:    trimOptionalString(req.StateProvince),
		City:             trimOptionalString(req.City),
	}

	if req.CountryCode != nil && strings.TrimSpace(*req.CountryCode) != "" {
		code := strings.ToUpper(strings.TrimSpace(*req.CountryCode))
		countryName, err := h.db.ResolveCountryName(ctx, code)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid country_code")
		}
		update.CountryCode = &code
		update.Country = &countryName
	} else {
		update.Country = trimOptionalString(req.Country)
	}

	if update.ProfilePhotoURL != nil && *update.ProfilePhotoURL != "" &&
		!storage.IsManagedProfilePhotoPath(*update.ProfilePhotoURL) {
		urls, err := validateHTTPSImageURLs([]string{*update.ProfilePhotoURL}, 1)
		if err != nil {
			return nil, nil, fmt.Errorf("profile photo: %s", err.Error())
		}
		if len(urls) == 0 {
			empty := ""
			update.ProfilePhotoURL = &empty
		} else {
			update.ProfilePhotoURL = &urls[0]
		}
	}

	stage := strings.TrimSpace(req.JourneyStage)
	if stage == "" && currentUser.JourneyStage != nil {
		stage = *currentUser.JourneyStage
	}
	if stage != "" {
		normalized, err := profile.NormalizeStage(stage)
		if err != nil {
			return nil, nil, err
		}

		currentStage := ""
		if currentUser.JourneyStage != nil {
			currentStage = *currentUser.JourneyStage
		}
		if currentStage != "" && currentStage != normalized && !profile.CanTransition(currentStage, normalized) {
			return nil, nil, fmt.Errorf("invalid journey transition from %s to %s", currentStage, normalized)
		}

		validateInput := mergedStageInput(currentUser, req)
		validateInput.Stage = normalized
		if markComplete || currentStage != normalized {
			if err := profile.ValidateStageProfile(normalized, validateInput, markComplete); err != nil {
				return nil, nil, err
			}
		}

		update.JourneyStage = &normalized
		if currentStage != normalized || currentUser.JourneyStageSince == nil {
			since := dateOnly(now)
			update.JourneyStageSince = &since
		}
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

	// Profile changes affect welcome personalization — invalidate cached messages.
	if err := h.db.DeleteWelcomeMessagesForUser(ctx, userID); err != nil {
		return nil, nil, fmt.Errorf("failed to invalidate welcome message: %w", err)
	}

	return user, facts, nil
}

func (h *ProfileHandler) syncProfileFacts(ctx context.Context, userID string, update db.UserProfileUpdate) error {
	if update.JourneyStage != nil {
		if _, err := h.db.SaveOrUpdateFact(ctx, userID, "journey_stage", *update.JourneyStage, profileFactConfidence); err != nil {
			return err
		}
	}

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
		// Week slider is the source of truth when both are supplied.
		week := clampWeek(*weekPtr)
		edd := profile.EDDFromWeek(week, now)
		return week, edd, profile.PregnancyStartFromWeek(week, now), nil
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
		Name:                         name,
		Language:                     user.Language,
		OnboardingCompleted:          user.OnboardingCompletedAt != nil,
		JourneyStage:                 journeyStageValue(user),
		JourneyStageSince:            user.JourneyStageSince,
		BabyBirthDate:                user.BabyBirthDate,
		LossDate:                     user.LossDate,
		PregnancyWeek:                pregnancyWeek,
		ExpectedDeliveryDate:         user.ExpectedDeliveryDate,
		PregnancyStartDate:           user.PregnancyStartDate,
		IsFirstPregnancy:             user.IsFirstPregnancy,
		PrimaryConcern:               user.PrimaryConcern,
		DietPreference:               user.DietPreference,
		ProfilePhotoURL:              user.ProfilePhotoURL,
		Country:                      user.Country,
		CountryCode:                  user.CountryCode,
		StateProvince:                user.StateProvince,
		City:                         user.City,
		CommunityOnboardingCompleted: user.CommunityOnboardingAt != nil,
		LearnedFacts:                 learnedFacts,
		Facts:                        allFacts,
	}
}

func (h *ProfileHandler) buildProfileResponseWithCommunity(ctx context.Context, user *db.User, facts []db.UserFact) ProfileResponse {
	resp := buildProfileResponse(user, facts)
	interests, _ := h.db.GetUserCommunityInterests(ctx, user.ID)
	resp.CommunityInterests = interests
	return resp
}

func journeyStageValue(user *db.User) string {
	if user.JourneyStage != nil && *user.JourneyStage != "" {
		return *user.JourneyStage
	}
	if user.PregnancyWeek != nil || user.ExpectedDeliveryDate != nil {
		return profile.StagePregnant
	}
	return ""
}

func stageSaveInput(req ProfileSaveRequest) profile.StageSaveInput {
	return profile.StageSaveInput{
		Stage:         req.JourneyStage,
		PregnancyWeek: req.PregnancyWeek,
		ExpectedDue:   req.ExpectedDeliveryDate,
		BabyBirthDate: req.BabyBirthDate,
		LossDate:      req.LossDate,
		IsFirstPreg:   req.IsFirstPregnancy,
	}
}

func mergedStageInput(current *db.User, req ProfileSaveRequest) profile.StageSaveInput {
	input := stageSaveInput(req)
	if input.PregnancyWeek == nil {
		input.PregnancyWeek = current.PregnancyWeek
	}
	if input.ExpectedDue == nil {
		input.ExpectedDue = current.ExpectedDeliveryDate
	}
	if input.BabyBirthDate == nil {
		input.BabyBirthDate = current.BabyBirthDate
	}
	if input.LossDate == nil {
		input.LossDate = current.LossDate
	}
	return input
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
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

func publicBaseURL(c *gin.Context) string {
	if v := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	return scheme + "://" + c.Request.Host
}
