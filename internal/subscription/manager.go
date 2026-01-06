package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) HasFeature(ctx context.Context, userID string, featureKey string) (bool, error) {
	if m.db == nil {
		return false, errors.New("db not initialized")
	}

	const q = `
SELECT EXISTS (
    SELECT 1
    FROM subscriptions s
    JOIN plans p ON p.id = s.plan_id AND p.active = TRUE
    JOIN plan_features pf ON pf.plan_id = p.id
    JOIN features f ON f.id = pf.feature_id
    WHERE s.user_id = $1
      AND s.status = 'active'
      AND (s.ends_at IS NULL OR s.ends_at > NOW())
      AND f.feature_key = $2
);`

	row := m.db.QueryRowContext(ctx, q, userID, featureKey)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("check feature access: %w", err)
	}
	return exists, nil
}

// CheckQuota verifies if user is within quota limits for a feature
func (m *Manager) CheckQuota(ctx context.Context, userID string, featureCode string) (bool, error) {
	if m.db == nil {
		return false, errors.New("db not initialized")
	}

	// Query: get quota limit, period, and current usage
	const q = `
SELECT 
    pf.quota_limit,
    pf.quota_period,
    COALESCE(fu.usage_count, 0) as usage_count
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id AND p.active = TRUE
JOIN plan_features pf ON pf.plan_id = p.id
JOIN features f ON f.id = pf.feature_id
LEFT JOIN feature_usage fu ON fu.user_id = s.user_id 
    AND fu.feature_key = f.feature_key
    AND fu.period_end > NOW()
WHERE s.user_id = $1
  AND s.status = 'active'
  AND (s.ends_at IS NULL OR s.ends_at > NOW())
  AND f.feature_key = $2;`

	var quotaLimit sql.NullInt64
	var quotaPeriod string
	var usageCount int

	err := m.db.QueryRowContext(ctx, q, userID, featureCode).Scan(&quotaLimit, &quotaPeriod, &usageCount)
	if err == sql.ErrNoRows {
		// No active subscription or feature not available
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check quota: %w", err)
	}

	// Unlimited quota (NULL quota_limit)
	if !quotaLimit.Valid || quotaPeriod == "unlimited" {
		return true, nil
	}

	// Check if within quota
	return usageCount < int(quotaLimit.Int64), nil
}

// IncrementUsage increments the usage counter for a feature
func (m *Manager) IncrementUsage(ctx context.Context, userID string, featureCode string) error {
	if m.db == nil {
		return errors.New("db not initialized")
	}

	// Get quota period for the feature
	const getPeriodQuery = `
SELECT pf.quota_period
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id AND p.active = TRUE
JOIN plan_features pf ON pf.plan_id = p.id
JOIN features f ON f.id = pf.feature_id
WHERE s.user_id = $1
  AND s.status = 'active'
  AND (s.ends_at IS NULL OR s.ends_at > NOW())
  AND f.feature_key = $2;`

	var quotaPeriod string
	err := m.db.QueryRowContext(ctx, getPeriodQuery, userID, featureCode).Scan(&quotaPeriod)
	if err != nil {
		return fmt.Errorf("get quota period: %w", err)
	}

	// Calculate period bounds
	now := time.Now()
	periodStart, periodEnd := calculatePeriodBounds(now, quotaPeriod)

	// Upsert usage record
	const upsertQuery = `
INSERT INTO feature_usage (user_id, feature_key, usage_count, period_start, period_end, updated_at)
VALUES ($1, $2, 1, $3, $4, NOW())
ON CONFLICT (user_id, feature_key, period_start)
DO UPDATE SET 
    usage_count = feature_usage.usage_count + 1,
    updated_at = NOW();`

	_, err = m.db.ExecContext(ctx, upsertQuery, userID, featureCode, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}

	return nil
}

// calculatePeriodBounds returns start and end timestamps for a quota period
func calculatePeriodBounds(now time.Time, period string) (time.Time, time.Time) {
	switch period {
	case "daily":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 1)
		return start, end
	case "weekly":
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 7)
		return start, end
	case "monthly":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		return start, end
	default:
		// Unlimited - use a far future date
		return now, now.AddDate(100, 0, 0)
	}
}

// QuotaInfo contains detailed quota information
type QuotaInfo struct {
	QuotaLimit  *int      `json:"quota_limit"`  // nil = unlimited
	QuotaPeriod string    `json:"quota_period"` // daily/weekly/monthly/unlimited
	UsageCount  int       `json:"usage_count"`
	PeriodEnd   time.Time `json:"period_end"`
}

// GetQuotaInfo returns detailed quota information for a user/feature
func (m *Manager) GetQuotaInfo(ctx context.Context, userID string, featureCode string) (*QuotaInfo, error) {
	if m.db == nil {
		return nil, errors.New("db not initialized")
	}

	const q = `
SELECT 
    pf.quota_limit,
    pf.quota_period,
    COALESCE(fu.usage_count, 0) as usage_count,
    COALESCE(fu.period_end, NOW() + INTERVAL '1 day') as period_end
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id AND p.active = TRUE
JOIN plan_features pf ON pf.plan_id = p.id
JOIN features f ON f.id = pf.feature_id
LEFT JOIN feature_usage fu ON fu.user_id = s.user_id 
    AND fu.feature_key = f.feature_key
    AND fu.period_end > NOW()
WHERE s.user_id = $1
  AND s.status = 'active'
  AND (s.ends_at IS NULL OR s.ends_at > NOW())
  AND f.feature_key = $2;`

	var info QuotaInfo
	var quotaLimit sql.NullInt64

	err := m.db.QueryRowContext(ctx, q, userID, featureCode).
		Scan(&quotaLimit, &info.QuotaPeriod, &info.UsageCount, &info.PeriodEnd)

	if err == sql.ErrNoRows {
		return nil, errors.New("feature not available")
	}
	if err != nil {
		return nil, fmt.Errorf("get quota info: %w", err)
	}

	if quotaLimit.Valid {
		limit := int(quotaLimit.Int64)
		info.QuotaLimit = &limit
	}

	return &info, nil
}

// UserFeature represents a feature available to a user
type UserFeature struct {
	FeatureKey  string `json:"feature_key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	QuotaLimit  *int   `json:"quota_limit"`
	QuotaPeriod string `json:"quota_period"`
}

// GetUserFeatures returns all features available to a user
func (m *Manager) GetUserFeatures(ctx context.Context, userID string) ([]UserFeature, error) {
	if m.db == nil {
		return nil, errors.New("db not initialized")
	}

	const q = `
SELECT 
    f.feature_key,
    f.name,
    f.description,
    pf.quota_limit,
    pf.quota_period
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id AND p.active = TRUE
JOIN plan_features pf ON pf.plan_id = p.id
JOIN features f ON f.id = pf.feature_id
WHERE s.user_id = $1
  AND s.status = 'active'
  AND (s.ends_at IS NULL OR s.ends_at > NOW())
ORDER BY f.feature_key;`

	rows, err := m.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("query user features: %w", err)
	}
	defer rows.Close()

	var features []UserFeature
	for rows.Next() {
		var f UserFeature
		var quotaLimit sql.NullInt64

		if err := rows.Scan(&f.FeatureKey, &f.Name, &f.Description, &quotaLimit, &f.QuotaPeriod); err != nil {
			return nil, fmt.Errorf("scan feature: %w", err)
		}

		if quotaLimit.Valid {
			limit := int(quotaLimit.Int64)
			f.QuotaLimit = &limit
		}

		features = append(features, f)
	}

	return features, rows.Err()
}

// Subscription represents a user's subscription
type Subscription struct {
	ID       int        `json:"id"`
	PlanID   int        `json:"plan_id"`
	PlanCode string     `json:"plan_code"`
	PlanName string     `json:"plan_name"`
	Status   string     `json:"status"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`
}

// GetActiveSubscription returns a user's active subscription
func (m *Manager) GetActiveSubscription(ctx context.Context, userID string) (*Subscription, error) {
	if m.db == nil {
		return nil, errors.New("db not initialized")
	}

	const q = `
SELECT s.id, s.plan_id, p.code, p.name, s.status, s.starts_at, s.ends_at
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
WHERE s.user_id = $1
  AND s.status = 'active'
  AND (s.ends_at IS NULL OR s.ends_at > NOW())
ORDER BY s.starts_at DESC
LIMIT 1;`

	var sub Subscription
	err := m.db.QueryRowContext(ctx, q, userID).
		Scan(&sub.ID, &sub.PlanID, &sub.PlanCode, &sub.PlanName, &sub.Status, &sub.StartsAt, &sub.EndsAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active subscription: %w", err)
	}

	return &sub, nil
}

// Plan represents a subscription plan
type Plan struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

// ListPlans returns all subscription plans
func (m *Manager) ListPlans(ctx context.Context) ([]Plan, error) {
	if m.db == nil {
		return nil, errors.New("db not initialized")
	}

	const q = `SELECT id, code, name, description, active FROM plans ORDER BY id;`

	rows, err := m.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query plans: %w", err)
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Active); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, p)
	}

	return plans, rows.Err()
}

// UpdateUserPlan changes a user's subscription plan
func (m *Manager) UpdateUserPlan(ctx context.Context, userID, planCode string) error {
	if m.db == nil {
		return errors.New("db not initialized")
	}

	// Get plan ID
	var planID int
	err := m.db.QueryRowContext(ctx, `SELECT id FROM plans WHERE code = $1 AND active = TRUE`, planCode).Scan(&planID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("plan not found: %s", planCode)
	}
	if err != nil {
		return fmt.Errorf("query plan: %w", err)
	}

	// Cancel existing subscription
	_, err = m.db.ExecContext(ctx, `
		UPDATE subscriptions 
		SET status = 'canceled', ends_at = NOW() 
		WHERE user_id = $1 AND status = 'active'
	`, userID)
	if err != nil {
		return fmt.Errorf("cancel existing subscription: %w", err)
	}

	// Create new subscription
	_, err = m.db.ExecContext(ctx, `
		INSERT INTO subscriptions (user_id, plan_id, status, starts_at)
		VALUES ($1, $2, 'active', NOW())
	`, userID, planID)
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}

	return nil
}

// ResetQuota resets quota usage for a user/feature
func (m *Manager) ResetQuota(ctx context.Context, userID, featureCode string) error {
	if m.db == nil {
		return errors.New("db not initialized")
	}

	_, err := m.db.ExecContext(ctx, `
		DELETE FROM feature_usage 
		WHERE user_id = $1 AND feature_key = $2
	`, userID, featureCode)

	if err != nil {
		return fmt.Errorf("reset quota: %w", err)
	}

	return nil
}

// QuotaStats represents aggregated quota statistics
type QuotaStats struct {
	TotalUsers     int     `json:"total_users"`
	TotalUsage     int     `json:"total_usage"`
	AverageUsage   float64 `json:"average_usage"`
	UsersAtLimit   int     `json:"users_at_limit"`
	UsersOverLimit int     `json:"users_over_limit"`
}

// GetQuotaStats returns system-wide quota statistics (stub for now)
func (m *Manager) GetQuotaStats(ctx context.Context, featureCode, planCode, period string) (*QuotaStats, error) {
	if m.db == nil {
		return nil, errors.New("db not initialized")
	}

	// TODO: Implement full stats query
	// For now, return basic stats
	return &QuotaStats{
		TotalUsers:   0,
		TotalUsage:   0,
		AverageUsage: 0,
	}, nil
}

// GrantFeature grants a feature to a user (stub for now)
func (m *Manager) GrantFeature(ctx context.Context, userID, featureKey string, expiresAt *int64) error {
	// TODO: Implement feature grant logic
	// This would require a new table for user_feature_grants
	return errors.New("not implemented yet")
}
