package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

type catalogKeyLabelRequest struct {
	Key         string  `json:"key"`
	Label       string  `json:"label" binding:"required"`
	SortOrder   *int    `json:"sort_order"`
	IsEnabled   *bool   `json:"is_enabled"`
	GroupKey    string  `json:"group_key"`
	Description *string `json:"description"`
}

type catalogCountryRequest struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	SortOrder *int   `json:"sort_order"`
	IsEnabled *bool  `json:"is_enabled"`
}

type catalogRegionRequest struct {
	ID          string `json:"id"`
	CountryCode string `json:"country_code" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	SortOrder   *int   `json:"sort_order"`
	IsEnabled   *bool  `json:"is_enabled"`
}

// ListInterestGroups godoc
// @Summary      List interest groups (admin)
// @Description  Returns all interest groups including disabled entries.
// @Tags         admin-community-catalog
// @Produce      json
// @Success      200 {object} ListInterestGroupsAdminResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/interest-groups [get]
func (h *AdminCommunityHandler) ListInterestGroups(c *gin.Context) {
	items, err := h.db.ListAllInterestGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list interest groups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interest_groups": items})
}

// UpsertInterestGroup godoc
// @Summary      Create or update interest group
// @Description  Upserts a group by path key. Set is_enabled=false to soft-disable (no DELETE route).
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        key  path string true "Stable group key"
// @Param        body body CatalogKeyLabelRequest true "Group fields"
// @Success      200 {object} UpsertInterestGroupResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/interest-groups/{key} [put]
func (h *AdminCommunityHandler) UpsertInterestGroup(c *gin.Context) {
	key := c.Param("key")
	var req catalogKeyLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := db.CommunityInterestGroup{
		Key:       key,
		Label:     req.Label,
		SortOrder: derefInt(req.SortOrder, 0),
		IsEnabled: derefBool(req.IsEnabled, true),
	}
	if err := h.db.UpsertInterestGroup(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save interest group"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interest_group": item})
}

// ListInterests godoc
// @Summary      List interests (admin)
// @Description  Returns all interests including disabled entries.
// @Tags         admin-community-catalog
// @Produce      json
// @Success      200 {object} ListInterestsAdminResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/interests [get]
func (h *AdminCommunityHandler) ListInterests(c *gin.Context) {
	items, err := h.db.ListAllInterests(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list interests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

// UpsertInterest godoc
// @Summary      Create or update interest
// @Description  Upserts an interest by path key. Requires group_key in body.
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        key  path string true "Stable interest key"
// @Param        body body CatalogKeyLabelRequest true "Interest fields"
// @Success      200 {object} UpsertInterestResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/interests/{key} [put]
func (h *AdminCommunityHandler) UpsertInterest(c *gin.Context) {
	key := c.Param("key")
	var req catalogKeyLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	groupKey := req.GroupKey
	if groupKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_key is required"})
		return
	}
	item := db.CommunityInterestCatalogItem{
		Key:       key,
		GroupKey:  groupKey,
		Label:     req.Label,
		SortOrder: derefInt(req.SortOrder, 0),
		IsEnabled: derefBool(req.IsEnabled, true),
	}
	if err := h.db.UpsertInterest(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save interest"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interest": item})
}

// ListBadgeTypes godoc
// @Summary      List badge types (admin)
// @Description  Returns all badge types including disabled entries.
// @Tags         admin-community-catalog
// @Produce      json
// @Success      200 {object} ListBadgeTypesAdminResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/badge-types [get]
func (h *AdminCommunityHandler) ListBadgeTypes(c *gin.Context) {
	items, err := h.db.ListAllBadgeTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list badge types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge_types": items})
}

// UpsertBadgeType godoc
// @Summary      Create or update badge type
// @Description  Upserts a badge type by path key.
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        key  path string true "Stable badge type key"
// @Param        body body CatalogKeyLabelRequest true "Badge type fields"
// @Success      200 {object} UpsertBadgeTypeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/badge-types/{key} [put]
func (h *AdminCommunityHandler) UpsertBadgeType(c *gin.Context) {
	key := c.Param("key")
	var req catalogKeyLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := db.CommunityBadgeType{
		Key:         key,
		Label:       req.Label,
		Description: req.Description,
		SortOrder:   derefInt(req.SortOrder, 0),
		IsEnabled:   derefBool(req.IsEnabled, true),
	}
	if err := h.db.UpsertBadgeType(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save badge type"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge_type": item})
}

// ListEventTypes godoc
// @Summary      List event types (admin)
// @Description  Returns all event types including disabled entries.
// @Tags         admin-community-catalog
// @Produce      json
// @Success      200 {object} ListEventTypesAdminResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/event-types [get]
func (h *AdminCommunityHandler) ListEventTypes(c *gin.Context) {
	items, err := h.db.ListAllEventTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list event types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"event_types": items})
}

// UpsertEventType godoc
// @Summary      Create or update event type
// @Description  Upserts an event type by path key.
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        key  path string true "Stable event type key"
// @Param        body body CatalogKeyLabelRequest true "Event type fields"
// @Success      200 {object} UpsertEventTypeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/event-types/{key} [put]
func (h *AdminCommunityHandler) UpsertEventType(c *gin.Context) {
	key := c.Param("key")
	var req catalogKeyLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := db.CommunityEventType{
		Key:         key,
		Label:       req.Label,
		Description: req.Description,
		SortOrder:   derefInt(req.SortOrder, 0),
		IsEnabled:   derefBool(req.IsEnabled, true),
	}
	if err := h.db.UpsertEventType(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save event type"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"event_type": item})
}

// ListCountries godoc
// @Summary      List countries (admin)
// @Description  Returns all countries including disabled entries.
// @Tags         admin-community-catalog
// @Produce      json
// @Success      200 {object} ListCountriesAdminResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/countries [get]
func (h *AdminCommunityHandler) ListCountries(c *gin.Context) {
	items, err := h.db.ListAllCountries(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list countries"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"countries": items})
}

// UpsertCountry godoc
// @Summary      Create or update country
// @Description  Upserts a country by ISO code path param (normalized to uppercase).
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        code path string true "ISO 3166-1 alpha-2 code" example(US)
// @Param        body body CatalogCountryRequest true "Country fields"
// @Success      200 {object} UpsertCountryResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/countries/{code} [put]
func (h *AdminCommunityHandler) UpsertCountry(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req catalogCountryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := db.CommunityCountry{
		Code:      code,
		Name:      req.Name,
		SortOrder: derefInt(req.SortOrder, 0),
		IsEnabled: derefBool(req.IsEnabled, true),
	}
	if err := h.db.UpsertCountry(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save country"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"country": item})
}

// ListRegions godoc
// @Summary      List regions (admin)
// @Description  Returns regions; optionally filter by country_code query param.
// @Tags         admin-community-catalog
// @Produce      json
// @Param        country_code query string false "ISO country code filter" example(US)
// @Success      200 {object} ListRegionsAdminResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/regions [get]
func (h *AdminCommunityHandler) ListRegions(c *gin.Context) {
	country := c.Query("country_code")
	items, err := h.db.ListAllRegions(c.Request.Context(), country)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list regions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"regions": items})
}

// CreateRegion godoc
// @Summary      Create region
// @Description  Creates a state/province catalog entry for autocomplete seeding.
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        body body CatalogRegionRequest true "Region fields"
// @Success      201 {object} CreateRegionResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/regions [post]
func (h *AdminCommunityHandler) CreateRegion(c *gin.Context) {
	var req catalogRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := db.CommunityRegion{
		CountryCode: strings.ToUpper(req.CountryCode),
		Code:        req.Code,
		Name:        req.Name,
		SortOrder:   derefInt(req.SortOrder, 0),
		IsEnabled:   derefBool(req.IsEnabled, true),
	}
	saved, err := h.db.UpsertRegion(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create region"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"region": saved})
}

// UpdateRegion godoc
// @Summary      Update region
// @Description  Updates an existing region by path id.
// @Tags         admin-community-catalog
// @Accept       json
// @Produce      json
// @Param        id   path string true "Region ID"
// @Param        body body CatalogRegionRequest true "Region fields"
// @Success      200 {object} UpdateRegionResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /admin/community/catalog/regions/{id} [put]
func (h *AdminCommunityHandler) UpdateRegion(c *gin.Context) {
	var req catalogRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := db.CommunityRegion{
		ID:          c.Param("id"),
		CountryCode: strings.ToUpper(req.CountryCode),
		Code:        req.Code,
		Name:        req.Name,
		SortOrder:   derefInt(req.SortOrder, 0),
		IsEnabled:   derefBool(req.IsEnabled, true),
	}
	saved, err := h.db.UpsertRegion(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update region"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": saved})
}

func derefInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func derefBool(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
