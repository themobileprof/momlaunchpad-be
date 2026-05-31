package api

import (
	"time"

	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request"`
}

// --- Catalog (public, enabled entries only) ---

// ListInterestsResponse lists interest groups for onboarding.
type ListInterestsResponse struct {
	Groups []db.CommunityInterestGroupWithItems `json:"groups"`
}

// ListBadgeTypesResponse lists assignable badge types visible in the app.
type ListBadgeTypesResponse struct {
	BadgeTypes []db.CommunityBadgeType `json:"badge_types"`
}

// ListEventTypesResponse lists event taxonomy entries for post creation.
type ListEventTypesResponse struct {
	EventTypes []db.CommunityEventType `json:"event_types"`
}

// ListCountriesResponse lists enabled countries for location pickers.
type ListCountriesResponse struct {
	Countries []db.CommunityCountry `json:"countries"`
}

// ListLocationSuggestionsResponse returns autocomplete strings for state/city fields.
type ListLocationSuggestionsResponse struct {
	Suggestions []string `json:"suggestions" example:"California,Los Angeles"`
}

// --- Onboarding & status ---

// CommunityStatusResponse describes the caller's community profile state.
type CommunityStatusResponse struct {
	CommunityOnboardingCompleted bool     `json:"community_onboarding_completed" example:"true"`
	Country                      *string  `json:"country" example:"United States"`
	StateProvince                *string  `json:"state_province" example:"California"`
	City                         *string  `json:"city" example:"Los Angeles"`
	Interests                    []string `json:"interests" example:"pregnancy,local_events"`
}

// CommunityOnboardingRequest completes community setup (location + interests).
type CommunityOnboardingRequest struct {
	CountryCode   string   `json:"country_code" binding:"required" example:"US"`
	StateProvince string   `json:"state_province" binding:"required" example:"California"`
	City          string   `json:"city" binding:"required" example:"Los Angeles"`
	Interests     []string `json:"interests" binding:"required" example:"pregnancy,local_events"`
}

// CommunityOnboardingResponse is returned after successful onboarding.
type CommunityOnboardingResponse struct {
	CommunityOnboardingCompleted bool     `json:"community_onboarding_completed" example:"true"`
	Country                      string   `json:"country" example:"United States"`
	StateProvince                string   `json:"state_province" example:"California"`
	City                         string   `json:"city" example:"Los Angeles"`
	Interests                    []string `json:"interests" example:"pregnancy,local_events"`
}

// --- Feed & posts ---

// CommunityAuthorJSON is the public author block on posts and replies.
type CommunityAuthorJSON struct {
	ID          *string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DisplayName string   `json:"display_name" example:"Jane D."`
	PhotoURL    *string  `json:"photo_url"`
	Badges      []string `json:"badges" example:"verified_expert"`
}

// CommunityPostJSON is a feed or detail post.
type CommunityPostJSON struct {
	ID               string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Body             string              `json:"body" example:"Looking for stroller recommendations."`
	IsAnonymous      bool                `json:"is_anonymous" example:"false"`
	Category         string              `json:"category" example:"gear_recommendations"`
	Scope            string              `json:"scope" example:"local"`
	MedicalRelevance string              `json:"medical_relevance" example:"none"`
	IsEvent          bool                `json:"is_event" example:"false"`
	LikeCount        int                 `json:"like_count" example:"3"`
	ReplyCount       int                 `json:"reply_count" example:"1"`
	LikedByMe        bool                `json:"liked_by_me" example:"false"`
	Country          *string             `json:"country" example:"United States"`
	StateProvince    *string             `json:"state_province" example:"California"`
	City             *string             `json:"city" example:"Los Angeles"`
	ImageURLs        []string            `json:"image_urls" example:"https://example.com/photo.jpg"`
	CreatedAt        string              `json:"created_at" example:"2026-05-30T12:00:00Z"`
	Author           CommunityAuthorJSON `json:"author"`
}

// CommunityFeedResponse is a paginated feed.
type CommunityFeedResponse struct {
	Posts      []CommunityPostJSON `json:"posts"`
	NextCursor *string             `json:"next_cursor" example:"2026-05-30T12:00:00Z"`
}

// CreateCommunityEventRequest is optional nested payload when creating an event post.
type CreateCommunityEventRequest struct {
	EventType   string     `json:"event_type" binding:"required" example:"playdate"`
	Title       string     `json:"title" binding:"required" example:"Park playdate"`
	Description *string    `json:"description" example:"Bring snacks and sunscreen."`
	Venue       *string    `json:"venue" example:"Central Park"`
	StartsAt    time.Time  `json:"starts_at" binding:"required" example:"2026-06-01T10:00:00Z"`
	EndsAt      *time.Time `json:"ends_at" example:"2026-06-01T12:00:00Z"`
}

// CreateCommunityPostRequest creates a feed post; include event for event posts.
type CreateCommunityPostRequest struct {
	Body        string                       `json:"body" binding:"required" example:"Anyone free Saturday morning?"`
	IsAnonymous bool                         `json:"is_anonymous" example:"false"`
	ImageURLs   []string                     `json:"image_urls" example:"https://example.com/photo.jpg"`
	Event       *CreateCommunityEventRequest `json:"event"`
}

// --- Replies ---

// CommunityReplyJSON is a reply on a post.
type CommunityReplyJSON struct {
	ID           string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	PostID       string              `json:"post_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Body         string              `json:"body" example:"We love the UPPAbaby Vista."`
	IsAnonymous  bool                `json:"is_anonymous" example:"false"`
	LikeCount    int                 `json:"like_count" example:"0"`
	LikedByMe    bool                `json:"liked_by_me" example:"false"`
	CreatedAt    string              `json:"created_at" example:"2026-05-30T12:05:00Z"`
	Author       CommunityAuthorJSON `json:"author"`
}

// ListCommunityRepliesResponse lists replies for a post.
type ListCommunityRepliesResponse struct {
	Replies []CommunityReplyJSON `json:"replies"`
}

// CreateCommunityReplyRequest creates a reply on a post.
type CreateCommunityReplyRequest struct {
	Body        string `json:"body" binding:"required" example:"Happy to share our experience."`
	IsAnonymous bool   `json:"is_anonymous" example:"false"`
}

// --- Likes, hide, report ---

// ToggleLikeResponse is returned by post/reply like toggles.
type ToggleLikeResponse struct {
	Liked     bool `json:"liked" example:"true"`
	LikeCount int  `json:"like_count" example:"4"`
}

// HidePostResponse confirms a post was hidden for the caller.
type HidePostResponse struct {
	Hidden bool `json:"hidden" example:"true"`
}

// CommunityReportRequest reports a post or reply for moderation.
type CommunityReportRequest struct {
	Reason  string  `json:"reason" binding:"required" example:"spam"`
	Details *string `json:"details" example:"Repeated promotional links."`
}

// CommunityReportCreatedResponse is returned after submitting a report.
type CommunityReportCreatedResponse struct {
	ID     string `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status string `json:"status" example:"open"`
}

// --- Events ---

// CommunityEventJSON is event metadata linked to a post.
type CommunityEventJSON struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440003"`
	PostID         string  `json:"post_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EventType      *string `json:"event_type" example:"playdate"`
	Title          string  `json:"title" example:"Park playdate"`
	Description    *string `json:"description"`
	Venue          *string `json:"venue" example:"Central Park"`
	StartsAt       string  `json:"starts_at" example:"2026-06-01T10:00:00Z"`
	EndsAt         *string `json:"ends_at"`
	Country        *string `json:"country" example:"United States"`
	StateProvince  *string `json:"state_province" example:"California"`
	City           *string `json:"city" example:"Los Angeles"`
	InterestedCount int    `json:"interested_count" example:"5"`
	InterestedByMe bool    `json:"interested_by_me" example:"false"`
}

// ToggleEventInterestResponse is returned when marking interest in an event.
type ToggleEventInterestResponse struct {
	Interested      bool `json:"interested" example:"true"`
	InterestedCount int  `json:"interested_count" example:"6"`
}

// --- Social graph ---

// FollowUserResponse confirms follow state.
type FollowUserResponse struct {
	Following bool `json:"following" example:"true"`
}

// BlockUserResponse confirms a user was blocked.
type BlockUserResponse struct {
	Blocked bool `json:"blocked" example:"true"`
}

// --- Notifications ---

// CommunityNotificationJSON is an in-app community notification.
type CommunityNotificationJSON struct {
	ID        string         `json:"id" example:"550e8400-e29b-41d4-a716-446655440004"`
	Type      string         `json:"type" example:"post_reply"`
	Title     string         `json:"title" example:"New reply on your post"`
	Body      string         `json:"body" example:"We love the UPPAbaby Vista."`
	Payload   map[string]any `json:"payload"`
	ReadAt    *string        `json:"read_at"`
	CreatedAt string         `json:"created_at" example:"2026-05-30T12:05:00Z"`
}

// ListCommunityNotificationsResponse lists notifications for the caller.
type ListCommunityNotificationsResponse struct {
	Notifications []CommunityNotificationJSON `json:"notifications"`
}

// MarkNotificationReadResponse confirms a notification was marked read.
type MarkNotificationReadResponse struct {
	Read bool `json:"read" example:"true"`
}

// --- Admin moderation ---

// ListCommunityReportsResponse lists moderation reports (admin).
type ListCommunityReportsResponse struct {
	Reports []db.CommunityReport `json:"reports"`
}

// UpdateCommunityReportRequest updates report workflow status (admin).
type UpdateCommunityReportRequest struct {
	Status string `json:"status" binding:"required" enums:"reviewed,dismissed,actioned" example:"reviewed"`
}

// UpdateCommunityReportResponse confirms report status change.
type UpdateCommunityReportResponse struct {
	Status string `json:"status" example:"reviewed"`
}

// UpdateCommunityPostStatusRequest changes post visibility (admin).
type UpdateCommunityPostStatusRequest struct {
	Status string `json:"status" binding:"required" enums:"active,hidden,removed,pending_review" example:"hidden"`
}

// UpdateCommunityPostStatusResponse confirms post status change.
type UpdateCommunityPostStatusResponse struct {
	Status string `json:"status" example:"hidden"`
}

// GrantCommunityBadgeRequest assigns a badge to a user (admin).
type GrantCommunityBadgeRequest struct {
	BadgeType string `json:"badge_type" binding:"required" example:"verified_expert"`
}

// GrantCommunityBadgeResponse confirms badge grant.
type GrantCommunityBadgeResponse struct {
	BadgeType string `json:"badge_type" example:"verified_expert"`
}

// RevokeCommunityBadgeResponse confirms badge removal.
type RevokeCommunityBadgeResponse struct {
	Revoked bool `json:"revoked" example:"true"`
}

// --- Admin catalog ---

// CatalogKeyLabelRequest upserts catalog entries keyed by path param (admin).
type CatalogKeyLabelRequest struct {
	Key         string  `json:"key"`
	Label       string  `json:"label" binding:"required" example:"Pregnancy"`
	SortOrder   *int    `json:"sort_order" example:"10"`
	IsEnabled   *bool   `json:"is_enabled" example:"true"`
	GroupKey    string  `json:"group_key" example:"life_stage"`
	Description *string `json:"description" example:"Expert-verified contributor."`
}

// CatalogCountryRequest upserts a country (admin).
type CatalogCountryRequest struct {
	Code      string `json:"code" binding:"required" example:"US"`
	Name      string `json:"name" binding:"required" example:"United States"`
	SortOrder *int   `json:"sort_order" example:"1"`
	IsEnabled *bool  `json:"is_enabled" example:"true"`
}

// CatalogRegionRequest creates or updates a region (admin).
type CatalogRegionRequest struct {
	ID          string `json:"id"`
	CountryCode string `json:"country_code" binding:"required" example:"US"`
	Code        string `json:"code" binding:"required" example:"CA"`
	Name        string `json:"name" binding:"required" example:"California"`
	SortOrder   *int   `json:"sort_order" example:"1"`
	IsEnabled   *bool  `json:"is_enabled" example:"true"`
}

// ListInterestGroupsAdminResponse lists all interest groups including disabled (admin).
type ListInterestGroupsAdminResponse struct {
	InterestGroups []db.CommunityInterestGroup `json:"interest_groups"`
}

// UpsertInterestGroupResponse returns the saved interest group (admin).
type UpsertInterestGroupResponse struct {
	InterestGroup db.CommunityInterestGroup `json:"interest_group"`
}

// ListInterestsAdminResponse lists all interests including disabled (admin).
type ListInterestsAdminResponse struct {
	Interests []db.CommunityInterestCatalogItem `json:"interests"`
}

// UpsertInterestResponse returns the saved interest (admin).
type UpsertInterestResponse struct {
	Interest db.CommunityInterestCatalogItem `json:"interest"`
}

// ListBadgeTypesAdminResponse lists all badge types including disabled (admin).
type ListBadgeTypesAdminResponse struct {
	BadgeTypes []db.CommunityBadgeType `json:"badge_types"`
}

// UpsertBadgeTypeResponse returns the saved badge type (admin).
type UpsertBadgeTypeResponse struct {
	BadgeType db.CommunityBadgeType `json:"badge_type"`
}

// ListEventTypesAdminResponse lists all event types including disabled (admin).
type ListEventTypesAdminResponse struct {
	EventTypes []db.CommunityEventType `json:"event_types"`
}

// UpsertEventTypeResponse returns the saved event type (admin).
type UpsertEventTypeResponse struct {
	EventType db.CommunityEventType `json:"event_type"`
}

// ListCountriesAdminResponse lists all countries including disabled (admin).
type ListCountriesAdminResponse struct {
	Countries []db.CommunityCountry `json:"countries"`
}

// UpsertCountryResponse returns the saved country (admin).
type UpsertCountryResponse struct {
	Country db.CommunityCountry `json:"country"`
}

// ListRegionsAdminResponse lists regions, optionally filtered by country (admin).
type ListRegionsAdminResponse struct {
	Regions []db.CommunityRegion `json:"regions"`
}

// CreateRegionResponse returns a newly created region (admin).
type CreateRegionResponse struct {
	Region db.CommunityRegion `json:"region"`
}

// UpdateRegionResponse returns an updated region (admin).
type UpdateRegionResponse struct {
	Region db.CommunityRegion `json:"region"`
}
