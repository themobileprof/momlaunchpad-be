package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/community"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// CommunityHandler serves the parenting community API.
type CommunityHandler struct {
	db        *db.DB
	processor *community.Processor
}

// NewCommunityHandler creates a community handler.
func NewCommunityHandler(database *db.DB, processor *community.Processor) *CommunityHandler {
	return &CommunityHandler{db: database, processor: processor}
}

// RegisterRoutes mounts authenticated community endpoints under /community.
func (h *CommunityHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/community")
	{
		g.GET("/interests", h.ListInterests)
		g.GET("/event-types", h.ListEventTypes)
		g.GET("/badge-types", h.ListBadgeTypes)
		g.GET("/locations/countries", h.ListCountries)
		g.GET("/locations/suggestions", h.ListLocationSuggestions)
		g.GET("/status", h.GetStatus)
		g.POST("/onboarding", h.CompleteOnboarding)
		g.GET("/feed", h.GetFeed)
		g.POST("/posts", h.CreatePost)
		g.GET("/posts/:id", h.GetPost)
		g.POST("/posts/:id/replies", h.CreateReply)
		g.GET("/posts/:id/replies", h.ListReplies)
		g.POST("/posts/:id/like", h.TogglePostLike)
		g.POST("/posts/:id/hide", h.HidePost)
		g.POST("/posts/:id/report", h.ReportPost)
		g.POST("/replies/:id/like", h.ToggleReplyLike)
		g.POST("/replies/:id/report", h.ReportReply)
		g.GET("/posts/:id/event", h.GetEvent)
		g.POST("/events/:id/interested", h.ToggleEventInterest)
		g.POST("/users/:id/follow", h.FollowUser)
		g.DELETE("/users/:id/follow", h.UnfollowUser)
		g.POST("/users/:id/block", h.BlockUser)
		g.GET("/notifications", h.ListNotifications)
		g.PUT("/notifications/:id/read", h.MarkNotificationRead)
	}
}

// ListInterests godoc
// @Summary      List interest groups
// @Description  Returns enabled interest groups with nested interests for onboarding and filters.
// @Tags         community
// @Produce      json
// @Success      200 {object} ListInterestsResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/interests [get]
func (h *CommunityHandler) ListInterests(c *gin.Context) {
	groups, err := h.db.ListEnabledInterestGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load interests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// ListBadgeTypes godoc
// @Summary      List badge types
// @Description  Returns enabled badge types shown on author profiles.
// @Tags         community
// @Produce      json
// @Success      200 {object} ListBadgeTypesResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/badge-types [get]
func (h *CommunityHandler) ListBadgeTypes(c *gin.Context) {
	types, err := h.db.ListEnabledBadgeTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load badge types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge_types": types})
}

// ListEventTypes godoc
// @Summary      List event types
// @Description  Returns enabled event taxonomy entries for event posts.
// @Tags         community
// @Produce      json
// @Success      200 {object} ListEventTypesResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/event-types [get]
func (h *CommunityHandler) ListEventTypes(c *gin.Context) {
	types, err := h.db.ListEnabledEventTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load event types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"event_types": types})
}

func interestKeySet(keys []string) map[string]bool {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[k] = true
	}
	return set
}

// ListCountries godoc
// @Summary      List countries
// @Description  Returns enabled ISO countries for the location picker.
// @Tags         community
// @Produce      json
// @Success      200 {object} ListCountriesResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/locations/countries [get]
func (h *CommunityHandler) ListCountries(c *gin.Context) {
	countries, err := h.db.ListEnabledCountries(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load countries"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"countries": countries})
}

// ListLocationSuggestions godoc
// @Summary      Location autocomplete suggestions
// @Description  Returns distinct state/province or city strings previously entered by users. Use field=state_province or field=city.
// @Tags         community
// @Produce      json
// @Param        country_code    query string true  "ISO 3166-1 alpha-2 country code" example(US)
// @Param        field           query string true  "Suggestion field" Enums(state_province, city)
// @Param        q               query string false "Prefix filter"
// @Param        state_province  query string false "Scope city suggestions to a state (when field=city)"
// @Param        limit           query int    false "Max results (default 10, max 50)"
// @Success      200 {object} ListLocationSuggestionsResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/locations/suggestions [get]
func (h *CommunityHandler) ListLocationSuggestions(c *gin.Context) {
	countryCode := c.Query("country_code")
	field := c.Query("field")
	query := c.Query("q")
	state := c.Query("state_province")

	suggestions, err := h.db.ListLocationSuggestions(
		c.Request.Context(),
		countryCode,
		field,
		query,
		state,
		parseLimit(c, 10),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load suggestions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// GetStatus godoc
// @Summary      Community onboarding status
// @Description  Returns whether the caller completed community onboarding plus saved location and interests.
// @Tags         community
// @Produce      json
// @Success      200 {object} CommunityStatusResponse
// @Failure      404 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/status [get]
func (h *CommunityHandler) GetStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	interests, _ := h.db.GetUserCommunityInterests(c.Request.Context(), userID)
	c.JSON(http.StatusOK, gin.H{
		"community_onboarding_completed": user.CommunityOnboardingAt != nil,
		"country":                        user.Country,
		"state_province":                 user.StateProvince,
		"city":                           user.City,
		"interests":                      interests,
	})
}

type communityOnboardingRequest struct {
	CountryCode   string   `json:"country_code" binding:"required"`
	StateProvince string   `json:"state_province" binding:"required"`
	City          string   `json:"city" binding:"required"`
	Interests     []string `json:"interests" binding:"required"`
}

// CompleteOnboarding godoc
// @Summary      Complete community onboarding
// @Description  Saves location (country code + state/city text) and 1–5 catalog interests. Required before posting.
// @Tags         community
// @Accept       json
// @Produce      json
// @Param        body body CommunityOnboardingRequest true "Location and interests"
// @Success      200 {object} CommunityOnboardingResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/onboarding [post]
func (h *CommunityHandler) CompleteOnboarding(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req communityOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	stateName := strings.TrimSpace(req.StateProvince)
	city := strings.TrimSpace(req.City)

	countryName, err := h.db.ResolveCountryName(ctx, countryCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid country"})
		return
	}

	if stateName == "" || city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Location is required"})
		return
	}
	if len(req.Interests) == 0 || len(req.Interests) > community.MaxUserInterests {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Select 1 to 5 interests"})
		return
	}

	seen := make(map[string]bool)
	var keys []string
	for _, k := range req.Interests {
		k = strings.TrimSpace(k)
		if k == "" || seen[k] {
			continue
		}
		ok, err := h.db.IsEnabledInterest(ctx, k)
		if err != nil || !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interest: " + k})
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Select at least one interest"})
		return
	}

	if err := h.db.CompleteCommunityOnboarding(ctx, userID, countryCode, countryName, stateName, city); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save location"})
		return
	}
	if err := h.db.SetUserCommunityInterests(ctx, userID, keys); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save interests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"community_onboarding_completed": true,
		"country":                        countryName,
		"state_province":                 stateName,
		"city":                           city,
		"interests":                      keys,
	})
}

// GetFeed godoc
// @Summary      Community feed
// @Description  Returns paginated posts. Pass next_cursor from a prior response as cursor for the next page.
// @Tags         community
// @Produce      json
// @Param        filter query string false "Feed filter" Enums(for_you, local, following) default(for_you)
// @Param        cursor query string false "RFC3339 created_at of last post from previous page"
// @Param        limit  query int    false "Page size (default 20, max 50)"
// @Success      200 {object} CommunityFeedResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/feed [get]
func (h *CommunityHandler) GetFeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	filter := c.DefaultQuery("filter", "for_you")
	limit := parseLimit(c, 20)

	var cursor *time.Time
	if raw := c.Query("cursor"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			cursor = &t
		}
	}

	posts, err := h.db.ListCommunityFeed(c.Request.Context(), userID, filter, limit, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load feed"})
		return
	}

	items := make([]gin.H, 0, len(posts))
	for _, p := range posts {
		items = append(items, postToJSON(p))
	}

	var nextCursor *string
	if len(posts) == limit {
		last := posts[len(posts)-1].CreatedAt.UTC().Format(time.RFC3339)
		nextCursor = &last
	}

	c.JSON(http.StatusOK, gin.H{"posts": items, "next_cursor": nextCursor})
}

type createPostRequest struct {
	Body        string              `json:"body" binding:"required"`
	IsAnonymous bool                `json:"is_anonymous"`
	Event       *createEventRequest `json:"event"`
}

type createEventRequest struct {
	EventType   string     `json:"event_type" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	Description *string    `json:"description"`
	Venue       *string    `json:"venue"`
	StartsAt    time.Time  `json:"starts_at" binding:"required"`
	EndsAt      *time.Time `json:"ends_at"`
}

// CreatePost godoc
// @Summary      Create a post
// @Description  Creates a community post. Optional nested event object creates a linked local event. Requires completed onboarding.
// @Tags         community
// @Accept       json
// @Produce      json
// @Param        body body CreateCommunityPostRequest true "Post content"
// @Success      201 {object} CommunityPostJSON
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts [post]
func (h *CommunityHandler) CreatePost(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req createPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Post body is required"})
		return
	}

	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if user.CommunityOnboardingAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Complete community onboarding first"})
		return
	}

	ctx := c.Request.Context()
	validCategories, err := h.db.ListEnabledInterestKeys(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load categories"})
		return
	}

	analysis := h.processor.AnalyzePost(ctx, body, validCategories)
	if req.Event != nil {
		ok, err := h.db.IsEnabledEventType(ctx, req.Event.EventType)
		if err != nil || !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type"})
			return
		}
		analysis.IsEvent = true
		if interestKeySet(validCategories)["events_meetups"] {
			analysis.Category = "events_meetups"
		}
	}

	post := &db.CommunityPost{
		UserID:           userID,
		Body:             body,
		IsAnonymous:      req.IsAnonymous,
		Category:         analysis.Category,
		Scope:            analysis.Scope,
		MedicalRelevance: analysis.MedicalRelevance,
		IsEvent:          analysis.IsEvent,
		SafetyFlag:       analysis.SafetyFlag,
		SpamScore:        analysis.SpamScore,
		Status:           analysis.Status,
		Country:          user.Country,
		StateProvince:    user.StateProvince,
		City:             user.City,
	}

	saved, err := h.db.CreateCommunityPost(ctx, post)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	if req.Event != nil && analysis.IsEvent {
		eventType := req.Event.EventType
		event := &db.CommunityEvent{
			PostID:        saved.ID,
			EventType:     &eventType,
			Title:         strings.TrimSpace(req.Event.Title),
			Description:   req.Event.Description,
			Venue:         req.Event.Venue,
			StartsAt:      req.Event.StartsAt,
			EndsAt:        req.Event.EndsAt,
			Country:       user.Country,
			StateProvince: user.StateProvince,
			City:          user.City,
		}
		_, _ = h.db.CreateCommunityEvent(ctx, event)
	}

	full, _ := h.db.GetCommunityPostByID(ctx, saved.ID, userID)
	if full == nil {
		full = saved
	}
	c.JSON(http.StatusCreated, postToJSON(*full))
}

// GetPost godoc
// @Summary      Get a post
// @Description  Returns a single post by ID with author enrichment for the caller.
// @Tags         community
// @Produce      json
// @Param        id path string true "Post ID"
// @Success      200 {object} CommunityPostJSON
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id} [get]
func (h *CommunityHandler) GetPost(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")
	post, err := h.db.GetCommunityPostByID(c.Request.Context(), postID, userID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load post"})
		return
	}
	c.JSON(http.StatusOK, postToJSON(*post))
}

type createReplyRequest struct {
	Body        string `json:"body" binding:"required"`
	IsAnonymous bool   `json:"is_anonymous"`
}

// CreateReply godoc
// @Summary      Reply to a post
// @Description  Creates a flat reply on a post and notifies the post author.
// @Tags         community
// @Accept       json
// @Produce      json
// @Param        id   path string true "Post ID"
// @Param        body body CreateCommunityReplyRequest true "Reply content"
// @Success      201 {object} CommunityReplyJSON
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id}/replies [post]
func (h *CommunityHandler) CreateReply(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")
	var req createReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reply body is required"})
		return
	}

	ctx := c.Request.Context()
	post, err := h.db.GetCommunityPostByID(ctx, postID, userID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load post"})
		return
	}

	reply := &db.CommunityReply{
		PostID:      postID,
		UserID:      userID,
		Body:        body,
		IsAnonymous: req.IsAnonymous,
		Status:      "active",
	}
	saved, err := h.db.CreateCommunityReply(ctx, reply)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reply"})
		return
	}

	if post.UserID != userID {
		_ = h.db.CreateCommunityNotification(ctx, &db.CommunityNotification{
			UserID:  post.UserID,
			Type:    "post_reply",
			Title:   "New reply on your post",
			Body:    truncate(body, 120),
			Payload: map[string]any{"post_id": postID, "reply_id": saved.ID},
		})
	}

	fullReply, _ := h.enrichReply(ctx, *saved, userID)
	c.JSON(http.StatusCreated, replyToJSON(fullReply))
}

// ListReplies godoc
// @Summary      List post replies
// @Description  Returns all active replies for a post.
// @Tags         community
// @Produce      json
// @Param        id path string true "Post ID"
// @Success      200 {object} ListCommunityRepliesResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id}/replies [get]
func (h *CommunityHandler) ListReplies(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")
	replies, err := h.db.ListCommunityReplies(c.Request.Context(), postID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load replies"})
		return
	}
	items := make([]gin.H, 0, len(replies))
	for _, r := range replies {
		items = append(items, replyToJSON(r))
	}
	c.JSON(http.StatusOK, gin.H{"replies": items})
}

// TogglePostLike godoc
// @Summary      Like or unlike a post
// @Description  Toggles the caller's like on a post and returns updated counts.
// @Tags         community
// @Produce      json
// @Param        id path string true "Post ID"
// @Success      200 {object} ToggleLikeResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id}/like [post]
func (h *CommunityHandler) TogglePostLike(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")
	liked, count, err := h.db.TogglePostLike(c.Request.Context(), postID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update like"})
		return
	}

	if liked {
		authorID, _ := h.db.GetPostAuthorID(c.Request.Context(), postID)
		if authorID != "" && authorID != userID {
			_ = h.db.CreateCommunityNotification(c.Request.Context(), &db.CommunityNotification{
				UserID:  authorID,
				Type:    "post_like",
				Title:   "Someone liked your post",
				Body:    "Your post received a new like.",
				Payload: map[string]any{"post_id": postID},
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"liked": liked, "like_count": count})
}

// ToggleReplyLike godoc
// @Summary      Like or unlike a reply
// @Description  Toggles the caller's like on a reply and returns updated counts.
// @Tags         community
// @Produce      json
// @Param        id path string true "Reply ID"
// @Success      200 {object} ToggleLikeResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/replies/{id}/like [post]
func (h *CommunityHandler) ToggleReplyLike(c *gin.Context) {
	userID := middleware.GetUserID(c)
	replyID := c.Param("id")
	liked, count, err := h.db.ToggleReplyLike(c.Request.Context(), replyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update like"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liked": liked, "like_count": count})
}

// HidePost godoc
// @Summary      Hide a post
// @Description  Hides a post from the caller's feed without affecting other users.
// @Tags         community
// @Produce      json
// @Param        id path string true "Post ID"
// @Success      200 {object} HidePostResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id}/hide [post]
func (h *CommunityHandler) HidePost(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")
	if err := h.db.HidePost(c.Request.Context(), userID, postID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hide post"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hidden": true})
}

type reportRequest struct {
	Reason  string  `json:"reason" binding:"required"`
	Details *string `json:"details"`
}

// ReportPost godoc
// @Summary      Report a post
// @Description  Submits a moderation report for a post.
// @Tags         community
// @Accept       json
// @Produce      json
// @Param        id   path string true "Post ID"
// @Param        body body CommunityReportRequest true "Report reason"
// @Success      201 {object} CommunityReportCreatedResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id}/report [post]
func (h *CommunityHandler) ReportPost(c *gin.Context) {
	h.createReport(c, "post", c.Param("id"))
}

// ReportReply godoc
// @Summary      Report a reply
// @Description  Submits a moderation report for a reply.
// @Tags         community
// @Accept       json
// @Produce      json
// @Param        id   path string true "Reply ID"
// @Param        body body CommunityReportRequest true "Report reason"
// @Success      201 {object} CommunityReportCreatedResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/replies/{id}/report [post]
func (h *CommunityHandler) ReportReply(c *gin.Context) {
	h.createReport(c, "reply", c.Param("id"))
}

func (h *CommunityHandler) createReport(c *gin.Context, targetType, targetID string) {
	userID := middleware.GetUserID(c)
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	report := &db.CommunityReport{
		ReporterID: userID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     strings.TrimSpace(req.Reason),
		Details:    req.Details,
	}
	saved, err := h.db.CreateCommunityReport(c.Request.Context(), report)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": saved.ID, "status": saved.Status})
}

// GetEvent godoc
// @Summary      Get event for a post
// @Description  Returns event metadata linked to an event post.
// @Tags         community
// @Produce      json
// @Param        id path string true "Post ID"
// @Success      200 {object} CommunityEventJSON
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/posts/{id}/event [get]
func (h *CommunityHandler) GetEvent(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")
	event, err := h.db.GetCommunityEventByPostID(c.Request.Context(), postID, userID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load event"})
		return
	}
	c.JSON(http.StatusOK, eventToJSON(*event))
}

// ToggleEventInterest godoc
// @Summary      Mark interest in an event
// @Description  Toggles the caller's interest in a community event.
// @Tags         community
// @Produce      json
// @Param        id path string true "Event ID"
// @Success      200 {object} ToggleEventInterestResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/events/{id}/interested [post]
func (h *CommunityHandler) ToggleEventInterest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	eventID := c.Param("id")
	interested, count, err := h.db.ToggleEventInterest(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update interest"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interested": interested, "interested_count": count})
}

// FollowUser godoc
// @Summary      Follow a user
// @Description  Follows another community member and sends them a notification.
// @Tags         community
// @Produce      json
// @Param        id path string true "User ID to follow"
// @Success      200 {object} FollowUserResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/users/{id}/follow [post]
func (h *CommunityHandler) FollowUser(c *gin.Context) {
	followerID := middleware.GetUserID(c)
	followingID := c.Param("id")
	if followerID == followingID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot follow yourself"})
		return
	}
	if err := h.db.FollowUser(c.Request.Context(), followerID, followingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow user"})
		return
	}
	_ = h.db.CreateCommunityNotification(c.Request.Context(), &db.CommunityNotification{
		UserID:  followingID,
		Type:    "new_follower",
		Title:   "Someone followed you",
		Body:    "You have a new follower in the community.",
		Payload: map[string]any{"follower_id": followerID},
	})
	c.JSON(http.StatusOK, gin.H{"following": true})
}

// UnfollowUser godoc
// @Summary      Unfollow a user
// @Description  Removes a follow relationship.
// @Tags         community
// @Produce      json
// @Param        id path string true "User ID to unfollow"
// @Success      200 {object} FollowUserResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/users/{id}/follow [delete]
func (h *CommunityHandler) UnfollowUser(c *gin.Context) {
	followerID := middleware.GetUserID(c)
	followingID := c.Param("id")
	if err := h.db.UnfollowUser(c.Request.Context(), followerID, followingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfollow user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"following": false})
}

// BlockUser godoc
// @Summary      Block a user
// @Description  Blocks another member; their content is hidden from the caller.
// @Tags         community
// @Produce      json
// @Param        id path string true "User ID to block"
// @Success      200 {object} BlockUserResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/users/{id}/block [post]
func (h *CommunityHandler) BlockUser(c *gin.Context) {
	blockerID := middleware.GetUserID(c)
	blockedID := c.Param("id")
	if blockerID == blockedID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot block yourself"})
		return
	}
	if err := h.db.BlockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocked": true})
}

// ListNotifications godoc
// @Summary      List notifications
// @Description  Returns recent in-app community notifications for the caller.
// @Tags         community
// @Produce      json
// @Param        limit query int false "Max results (default 30, max 50)"
// @Success      200 {object} ListCommunityNotificationsResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/notifications [get]
func (h *CommunityHandler) ListNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.db.ListCommunityNotifications(c.Request.Context(), userID, parseLimit(c, 30))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load notifications"})
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, n := range items {
		out = append(out, notificationToJSON(n))
	}
	c.JSON(http.StatusOK, gin.H{"notifications": out})
}

// MarkNotificationRead godoc
// @Summary      Mark notification read
// @Description  Marks a single notification as read for the caller.
// @Tags         community
// @Produce      json
// @Param        id path string true "Notification ID"
// @Success      200 {object} MarkNotificationReadResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /community/notifications/{id}/read [put]
func (h *CommunityHandler) MarkNotificationRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")
	if err := h.db.MarkNotificationRead(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"read": true})
}

func postToJSON(p db.CommunityPost) gin.H {
	authorName := "Anonymous Mom"
	if !p.IsAnonymous && p.AuthorName != nil && *p.AuthorName != "" {
		authorName = *p.AuthorName
	} else if p.IsAnonymous {
		authorName = "Anonymous Mom"
	}

	return gin.H{
		"id":                p.ID,
		"body":              p.Body,
		"is_anonymous":      p.IsAnonymous,
		"category":          p.Category,
		"scope":             p.Scope,
		"medical_relevance": p.MedicalRelevance,
		"is_event":          p.IsEvent,
		"like_count":        p.LikeCount,
		"reply_count":       p.ReplyCount,
		"liked_by_me":       p.LikedByMe,
		"country":           p.Country,
		"state_province":    p.StateProvince,
		"city":              p.City,
		"created_at":        p.CreatedAt.UTC().Format(time.RFC3339),
		"author": gin.H{
			"id":           nullableAuthorID(p),
			"display_name": authorName,
			"photo_url":    p.AuthorPhotoURL,
			"badges":       p.AuthorBadges,
		},
	}
}

func nullableAuthorID(p db.CommunityPost) *string {
	if p.IsAnonymous {
		return nil
	}
	id := p.UserID
	return &id
}

func replyToJSON(r db.CommunityReply) gin.H {
	authorName := "Anonymous Mom"
	if !r.IsAnonymous && r.AuthorName != nil && *r.AuthorName != "" {
		authorName = *r.AuthorName
	}
	return gin.H{
		"id":           r.ID,
		"post_id":      r.PostID,
		"body":         r.Body,
		"is_anonymous": r.IsAnonymous,
		"like_count":   r.LikeCount,
		"liked_by_me":  r.LikedByMe,
		"created_at":   r.CreatedAt.UTC().Format(time.RFC3339),
		"author": gin.H{
			"id":           nullableReplyAuthorID(r),
			"display_name": authorName,
			"photo_url":    r.AuthorPhotoURL,
			"badges":       r.AuthorBadges,
		},
	}
}

func nullableReplyAuthorID(r db.CommunityReply) *string {
	if r.IsAnonymous {
		return nil
	}
	id := r.UserID
	return &id
}

func eventToJSON(e db.CommunityEvent) gin.H {
	return gin.H{
		"id":               e.ID,
		"post_id":          e.PostID,
		"event_type":       e.EventType,
		"title":            e.Title,
		"description":      e.Description,
		"venue":            e.Venue,
		"starts_at":        e.StartsAt.UTC().Format(time.RFC3339),
		"ends_at":          formatOptionalTime(e.EndsAt),
		"country":          e.Country,
		"state_province":   e.StateProvince,
		"city":             e.City,
		"interested_count": e.InterestedCount,
		"interested_by_me": e.InterestedByMe,
	}
}

func notificationToJSON(n db.CommunityNotification) gin.H {
	return gin.H{
		"id":         n.ID,
		"type":       n.Type,
		"title":      n.Title,
		"body":       n.Body,
		"payload":    n.Payload,
		"read_at":    formatOptionalTime(n.ReadAt),
		"created_at": n.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *CommunityHandler) enrichReply(ctx context.Context, reply db.CommunityReply, viewerID string) (db.CommunityReply, error) {
	replies, err := h.db.ListCommunityReplies(ctx, reply.PostID, viewerID)
	if err != nil {
		return reply, err
	}
	for _, r := range replies {
		if r.ID == reply.ID {
			return r, nil
		}
	}
	return reply, nil
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func parseLimit(c *gin.Context, defaultLimit int) int {
	raw := c.DefaultQuery("limit", "")
	if raw == "" {
		return defaultLimit
	}
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil || n <= 0 {
		return defaultLimit
	}
	if n > 50 {
		return 50
	}
	return n
}
