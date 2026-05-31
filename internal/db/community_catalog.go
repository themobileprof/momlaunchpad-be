package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const MaxCommunityInterests = 5

// CommunityInterestGroup is a configurable interest category group.
type CommunityInterestGroup struct {
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	SortOrder int       `json:"sort_order"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommunityInterestCatalogItem is a selectable interest.
type CommunityInterestCatalogItem struct {
	Key       string    `json:"key"`
	GroupKey  string    `json:"group_key"`
	Label     string    `json:"label"`
	SortOrder int       `json:"sort_order"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommunityInterestGroupWithItems is used for onboarding UI.
type CommunityInterestGroupWithItems struct {
	Key   string                         `json:"key"`
	Label string                         `json:"label"`
	Items []CommunityInterestCatalogItem `json:"items"`
}

// CommunityBadgeType is an expert/moderator badge definition.
type CommunityBadgeType struct {
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	Description *string   `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CommunityEventType is a configurable event taxonomy entry.
type CommunityEventType struct {
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	Description *string   `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CommunityCountry is a supported country for location pickers.
type CommunityCountry struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommunityRegion is a state/province within a country.
type CommunityRegion struct {
	ID          string    `json:"id"`
	CountryCode string    `json:"country_code"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	SortOrder   int       `json:"sort_order"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListEnabledInterestGroups returns grouped interests for the mobile app.
func (db *DB) ListEnabledInterestGroups(ctx context.Context) ([]CommunityInterestGroupWithItems, error) {
	groupRows, err := db.QueryContext(ctx, `
		SELECT key, label FROM community_interest_groups
		WHERE is_enabled = TRUE ORDER BY sort_order, label
	`)
	if err != nil {
		return nil, err
	}
	defer groupRows.Close()

	var groups []CommunityInterestGroupWithItems
	for groupRows.Next() {
		var g CommunityInterestGroupWithItems
		if err := groupRows.Scan(&g.Key, &g.Label); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := groupRows.Err(); err != nil {
		return nil, err
	}

	itemRows, err := db.QueryContext(ctx, `
		SELECT key, group_key, label FROM community_interests
		WHERE is_enabled = TRUE ORDER BY sort_order, label
	`)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	byGroup := make(map[string][]CommunityInterestCatalogItem)
	for itemRows.Next() {
		var item CommunityInterestCatalogItem
		if err := itemRows.Scan(&item.Key, &item.GroupKey, &item.Label); err != nil {
			return nil, err
		}
		byGroup[item.GroupKey] = append(byGroup[item.GroupKey], item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	for i := range groups {
		groups[i].Items = byGroup[groups[i].Key]
	}
	return groups, nil
}

// IsEnabledInterest returns true when key exists and is enabled.
func (db *DB) IsEnabledInterest(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM community_interests WHERE key = $1 AND is_enabled = TRUE)`,
		key,
	).Scan(&exists)
	return exists, err
}

// ListEnabledInterestKeys returns all enabled interest keys (for AI classification).
func (db *DB) ListEnabledInterestKeys(ctx context.Context) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT key FROM community_interests WHERE is_enabled = TRUE ORDER BY sort_order, key`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// IsEnabledBadgeType returns true when badge type exists and is enabled.
func (db *DB) IsEnabledBadgeType(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM community_badge_types WHERE key = $1 AND is_enabled = TRUE)`,
		key,
	).Scan(&exists)
	return exists, err
}

// ListEnabledBadgeTypes returns badge definitions for display.
func (db *DB) ListEnabledBadgeTypes(ctx context.Context) ([]CommunityBadgeType, error) {
	return db.listBadgeTypes(ctx, true)
}

func (db *DB) listBadgeTypes(ctx context.Context, enabledOnly bool) ([]CommunityBadgeType, error) {
	query := `
		SELECT key, label, description, sort_order, is_enabled, created_at, updated_at
		FROM community_badge_types
	`
	if enabledOnly {
		query += ` WHERE is_enabled = TRUE`
	}
	query += ` ORDER BY sort_order, label`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBadgeTypes(rows)
}

// ListEnabledEventTypes returns event types for create-event UI.
func (db *DB) ListEnabledEventTypes(ctx context.Context) ([]CommunityEventType, error) {
	return db.listEventTypes(ctx, true)
}

func (db *DB) listEventTypes(ctx context.Context, enabledOnly bool) ([]CommunityEventType, error) {
	query := `
		SELECT key, label, description, sort_order, is_enabled, created_at, updated_at
		FROM community_event_types
	`
	if enabledOnly {
		query += ` WHERE is_enabled = TRUE`
	}
	query += ` ORDER BY sort_order, label`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEventTypes(rows)
}

// IsEnabledEventType validates an event type key.
func (db *DB) IsEnabledEventType(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM community_event_types WHERE key = $1 AND is_enabled = TRUE)`,
		key,
	).Scan(&exists)
	return exists, err
}

// ListEnabledCountries returns countries for location pickers.
func (db *DB) ListEnabledCountries(ctx context.Context) ([]CommunityCountry, error) {
	return db.listCountries(ctx, true)
}

func (db *DB) listCountries(ctx context.Context, enabledOnly bool) ([]CommunityCountry, error) {
	query := `
		SELECT code, name, sort_order, is_enabled, created_at, updated_at
		FROM community_countries
	`
	if enabledOnly {
		query += ` WHERE is_enabled = TRUE`
	}
	query += ` ORDER BY sort_order, name`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CommunityCountry
	for rows.Next() {
		var c CommunityCountry
		if err := rows.Scan(&c.Code, &c.Name, &c.SortOrder, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// ListEnabledRegions returns regions for a country.
func (db *DB) ListEnabledRegions(ctx context.Context, countryCode string) ([]CommunityRegion, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, country_code, code, name, sort_order, is_enabled, created_at, updated_at
		FROM community_regions
		WHERE country_code = $1 AND is_enabled = TRUE
		ORDER BY sort_order, name
	`, countryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRegions(rows)
}

// ResolveCountryName returns the display name for a country code.
func (db *DB) ResolveCountryName(ctx context.Context, code string) (string, error) {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM community_countries WHERE code = $1 AND is_enabled = TRUE`,
		code,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return name, err
}

// ResolveRegionName returns the display name for a region in a country.
func (db *DB) ResolveRegionName(ctx context.Context, countryCode, regionCode string) (string, error) {
	var name string
	err := db.QueryRowContext(ctx, `
		SELECT name FROM community_regions
		WHERE country_code = $1 AND code = $2 AND is_enabled = TRUE
	`, countryCode, regionCode).Scan(&name)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return name, err
}

// --- Admin catalog CRUD ---

func (db *DB) ListAllInterestGroups(ctx context.Context) ([]CommunityInterestGroup, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT key, label, sort_order, is_enabled, created_at, updated_at
		FROM community_interest_groups ORDER BY sort_order, label
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CommunityInterestGroup
	for rows.Next() {
		var g CommunityInterestGroup
		if err := rows.Scan(&g.Key, &g.Label, &g.SortOrder, &g.IsEnabled, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (db *DB) UpsertInterestGroup(ctx context.Context, g CommunityInterestGroup) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO community_interest_groups (key, label, sort_order, is_enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO UPDATE SET
			label = EXCLUDED.label,
			sort_order = EXCLUDED.sort_order,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = CURRENT_TIMESTAMP
	`, g.Key, g.Label, g.SortOrder, g.IsEnabled)
	return err
}

func (db *DB) ListAllInterests(ctx context.Context) ([]CommunityInterestCatalogItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT key, group_key, label, sort_order, is_enabled, created_at, updated_at
		FROM community_interests ORDER BY group_key, sort_order, label
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CommunityInterestCatalogItem
	for rows.Next() {
		var item CommunityInterestCatalogItem
		if err := rows.Scan(&item.Key, &item.GroupKey, &item.Label, &item.SortOrder, &item.IsEnabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) UpsertInterest(ctx context.Context, item CommunityInterestCatalogItem) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO community_interests (key, group_key, label, sort_order, is_enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			group_key = EXCLUDED.group_key,
			label = EXCLUDED.label,
			sort_order = EXCLUDED.sort_order,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = CURRENT_TIMESTAMP
	`, item.Key, item.GroupKey, item.Label, item.SortOrder, item.IsEnabled)
	return err
}

func (db *DB) ListAllBadgeTypes(ctx context.Context) ([]CommunityBadgeType, error) {
	return db.listBadgeTypes(ctx, false)
}

func (db *DB) UpsertBadgeType(ctx context.Context, item CommunityBadgeType) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO community_badge_types (key, label, description, sort_order, is_enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			label = EXCLUDED.label,
			description = EXCLUDED.description,
			sort_order = EXCLUDED.sort_order,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = CURRENT_TIMESTAMP
	`, item.Key, item.Label, item.Description, item.SortOrder, item.IsEnabled)
	return err
}

func (db *DB) ListAllEventTypes(ctx context.Context) ([]CommunityEventType, error) {
	return db.listEventTypes(ctx, false)
}

func (db *DB) UpsertEventType(ctx context.Context, item CommunityEventType) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO community_event_types (key, label, description, sort_order, is_enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			label = EXCLUDED.label,
			description = EXCLUDED.description,
			sort_order = EXCLUDED.sort_order,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = CURRENT_TIMESTAMP
	`, item.Key, item.Label, item.Description, item.SortOrder, item.IsEnabled)
	return err
}

func (db *DB) ListAllCountries(ctx context.Context) ([]CommunityCountry, error) {
	return db.listCountries(ctx, false)
}

func (db *DB) UpsertCountry(ctx context.Context, item CommunityCountry) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO community_countries (code, name, sort_order, is_enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			sort_order = EXCLUDED.sort_order,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = CURRENT_TIMESTAMP
	`, strings.ToUpper(item.Code), item.Name, item.SortOrder, item.IsEnabled)
	return err
}

func (db *DB) ListAllRegions(ctx context.Context, countryCode string) ([]CommunityRegion, error) {
	query := `
		SELECT id, country_code, code, name, sort_order, is_enabled, created_at, updated_at
		FROM community_regions
	`
	args := []any{}
	if countryCode != "" {
		query += ` WHERE country_code = $1`
		args = append(args, countryCode)
	}
	query += ` ORDER BY country_code, sort_order, name`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRegions(rows)
}

func (db *DB) UpsertRegion(ctx context.Context, item CommunityRegion) (*CommunityRegion, error) {
	if item.ID != "" {
		_, err := db.ExecContext(ctx, `
			UPDATE community_regions
			SET country_code = $1, code = $2, name = $3, sort_order = $4, is_enabled = $5,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $6
		`, item.CountryCode, item.Code, item.Name, item.SortOrder, item.IsEnabled, item.ID)
		if err != nil {
			return nil, err
		}
		return &item, nil
	}

	out := &CommunityRegion{}
	err := db.QueryRowContext(ctx, `
		INSERT INTO community_regions (country_code, code, name, sort_order, is_enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (country_code, code) DO UPDATE SET
			name = EXCLUDED.name,
			sort_order = EXCLUDED.sort_order,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, country_code, code, name, sort_order, is_enabled, created_at, updated_at
	`, item.CountryCode, item.Code, item.Name, item.SortOrder, item.IsEnabled).Scan(
		&out.ID, &out.CountryCode, &out.Code, &out.Name, &out.SortOrder, &out.IsEnabled, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert region: %w", err)
	}
	return out, nil
}

func scanBadgeTypes(rows *sql.Rows) ([]CommunityBadgeType, error) {
	var items []CommunityBadgeType
	for rows.Next() {
		var item CommunityBadgeType
		var desc sql.NullString
		if err := rows.Scan(&item.Key, &item.Label, &desc, &item.SortOrder, &item.IsEnabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			item.Description = &desc.String
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanEventTypes(rows *sql.Rows) ([]CommunityEventType, error) {
	var items []CommunityEventType
	for rows.Next() {
		var item CommunityEventType
		var desc sql.NullString
		if err := rows.Scan(&item.Key, &item.Label, &desc, &item.SortOrder, &item.IsEnabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			item.Description = &desc.String
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanRegions(rows *sql.Rows) ([]CommunityRegion, error) {
	var items []CommunityRegion
	for rows.Next() {
		var item CommunityRegion
		if err := rows.Scan(&item.ID, &item.CountryCode, &item.Code, &item.Name, &item.SortOrder, &item.IsEnabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// BadgeLabelsMap returns key→label for enabled badges (for API responses).
func (db *DB) BadgeLabelsMap(ctx context.Context) (map[string]string, error) {
	types, err := db.ListEnabledBadgeTypes(ctx)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(types))
	for _, t := range types {
		labels[t.Key] = t.Label
	}
	return labels, nil
}

// ListLocationSuggestions returns autocomplete values for state/province or city.
func (db *DB) ListLocationSuggestions(ctx context.Context, countryCode, field, query, stateProvince string, limit int) ([]string, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	query = strings.TrimSpace(query)
	if countryCode == "" || len(query) < 1 {
		return nil, nil
	}
	prefix := query + "%"

	switch field {
	case "state_province":
		rows, err := db.QueryContext(ctx, `
			SELECT suggestion FROM (
				SELECT DISTINCT name AS suggestion
				FROM community_regions
				WHERE country_code = $1 AND is_enabled = TRUE AND name ILIKE $2
				UNION
				SELECT DISTINCT state_province AS suggestion
				FROM users
				WHERE country_code = $1
				  AND state_province IS NOT NULL
				  AND TRIM(state_province) <> ''
				  AND state_province ILIKE $2
			) s
			ORDER BY suggestion
			LIMIT $3
		`, countryCode, prefix, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStringRows(rows)
	case "city":
		args := []any{countryCode, prefix, limit}
		sql := `
			SELECT suggestion FROM (
				SELECT DISTINCT name AS suggestion
				FROM community_regions
				WHERE country_code = $1 AND is_enabled = TRUE AND name ILIKE $2
				UNION
				SELECT DISTINCT city AS suggestion
				FROM users
				WHERE country_code = $1
				  AND city IS NOT NULL
				  AND TRIM(city) <> ''
				  AND city ILIKE $2
		`
		if strings.TrimSpace(stateProvince) != "" {
			sql += ` AND state_province ILIKE $4`
			args = append(args, stateProvince+"%")
		}
		sql += `
			) s
			ORDER BY suggestion
			LIMIT $3
		`
		rows, err := db.QueryContext(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStringRows(rows)
	default:
		return nil, fmt.Errorf("unsupported field: %s", field)
	}
}

func scanStringRows(rows *sql.Rows) ([]string, error) {
	var items []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}
