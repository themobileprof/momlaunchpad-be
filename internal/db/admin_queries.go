package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ============================================================================
// PLAN MANAGEMENT
// ============================================================================

// Plan represents a subscription plan
type Plan struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreatePlan creates a new subscription plan
func (db *DB) CreatePlan(ctx context.Context, code, name, description string) (*Plan, error) {
	query := `
		INSERT INTO plans (code, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, code, name, description, active, created_at
	`

	plan := &Plan{}
	err := db.QueryRowContext(ctx, query, code, name, description).Scan(
		&plan.ID, &plan.Code, &plan.Name, &plan.Description, &plan.Active, &plan.CreatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	return plan, nil
}

// UpdatePlan updates an existing plan
func (db *DB) UpdatePlan(ctx context.Context, planID int, name, description string, active *bool) error {
	query := `
		UPDATE plans 
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = COALESCE(NULLIF($3, ''), description),
		    active = COALESCE($4, active)
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, planID, name, description, active)
	if err != nil {
		return fmt.Errorf("failed to update plan: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeactivatePlan soft-deletes a plan by setting active = false
func (db *DB) DeactivatePlan(ctx context.Context, planID int) error {
	query := `UPDATE plans SET active = FALSE WHERE id = $1`

	result, err := db.ExecContext(ctx, query, planID)
	if err != nil {
		return fmt.Errorf("failed to deactivate plan: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetAllPlans returns all plans
func (db *DB) GetAllPlans(ctx context.Context) ([]Plan, error) {
	query := `
		SELECT id, code, name, description, active, created_at
		FROM plans
		ORDER BY created_at
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query plans: %w", err)
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Active, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, p)
	}

	return plans, nil
}

// ============================================================================
// FEATURE MANAGEMENT
// ============================================================================

// Feature represents a feature
type Feature struct {
	ID          int       `json:"id"`
	FeatureKey  string    `json:"feature_key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetAllFeatures returns all features
func (db *DB) GetAllFeatures(ctx context.Context) ([]Feature, error) {
	query := `
		SELECT id, feature_key, name, description, created_at
		FROM features
		ORDER BY created_at
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query features: %w", err)
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var f Feature
		if err := rows.Scan(&f.ID, &f.FeatureKey, &f.Name, &f.Description, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	return features, nil
}

// CreateFeature creates a new feature
func (db *DB) CreateFeature(ctx context.Context, featureKey, name, description string) (*Feature, error) {
	query := `
		INSERT INTO features (feature_key, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, feature_key, name, description, created_at
	`

	feature := &Feature{}
	err := db.QueryRowContext(ctx, query, featureKey, name, description).Scan(
		&feature.ID, &feature.FeatureKey, &feature.Name, &feature.Description, &feature.CreatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to create feature: %w", err)
	}

	return feature, nil
}

// UpdateFeature updates an existing feature
func (db *DB) UpdateFeature(ctx context.Context, featureID int, name, description string) error {
	query := `
		UPDATE features 
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = COALESCE(NULLIF($3, ''), description)
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, featureID, name, description)
	if err != nil {
		return fmt.Errorf("failed to update feature: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteFeature removes a feature
func (db *DB) DeleteFeature(ctx context.Context, featureID int) error {
	query := `DELETE FROM features WHERE id = $1`

	result, err := db.ExecContext(ctx, query, featureID)
	if err != nil {
		return fmt.Errorf("failed to delete feature: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// PlanFeature represents a feature assigned to a plan with quota
type PlanFeature struct {
	FeatureID   int    `json:"feature_id"`
	FeatureKey  string `json:"feature_key"`
	FeatureName string `json:"feature_name"`
	QuotaLimit  *int   `json:"quota_limit"`
	QuotaPeriod string `json:"quota_period"`
}

// AssignFeatureToPlan assigns a feature to a plan with quota settings
func (db *DB) AssignFeatureToPlan(ctx context.Context, planID, featureID int, quotaLimit *int, quotaPeriod string) error {
	query := `
		INSERT INTO plan_features (plan_id, feature_id, quota_limit, quota_period)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (plan_id, feature_id) 
		DO UPDATE SET quota_limit = $3, quota_period = $4
	`

	_, err := db.ExecContext(ctx, query, planID, featureID, quotaLimit, quotaPeriod)
	if err != nil {
		return fmt.Errorf("failed to assign feature to plan: %w", err)
	}

	return nil
}

// RemoveFeatureFromPlan removes a feature from a plan
func (db *DB) RemoveFeatureFromPlan(ctx context.Context, planID, featureID int) error {
	query := `DELETE FROM plan_features WHERE plan_id = $1 AND feature_id = $2`

	_, err := db.ExecContext(ctx, query, planID, featureID)
	if err != nil {
		return fmt.Errorf("failed to remove feature from plan: %w", err)
	}

	return nil
}

// GetPlanFeatures returns all features for a plan with quotas
func (db *DB) GetPlanFeatures(ctx context.Context, planID int) ([]PlanFeature, error) {
	query := `
		SELECT f.id, f.feature_key, f.name, pf.quota_limit, pf.quota_period
		FROM plan_features pf
		JOIN features f ON f.id = pf.feature_id
		WHERE pf.plan_id = $1
		ORDER BY f.name
	`

	rows, err := db.QueryContext(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to query plan features: %w", err)
	}
	defer rows.Close()

	var features []PlanFeature
	for rows.Next() {
		var pf PlanFeature
		if err := rows.Scan(&pf.FeatureID, &pf.FeatureKey, &pf.FeatureName, &pf.QuotaLimit, &pf.QuotaPeriod); err != nil {
			return nil, fmt.Errorf("failed to scan plan feature: %w", err)
		}
		features = append(features, pf)
	}

	return features, nil
}

// ============================================================================
// LANGUAGE MANAGEMENT
// ============================================================================

// Note: Language struct is defined in db.go

// GetAllLanguages returns all languages
func (db *DB) GetAllLanguages(ctx context.Context) ([]Language, error) {
	query := `
		SELECT code, name, native_name, is_enabled, is_experimental, created_at
		FROM languages
		ORDER BY name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query languages: %w", err)
	}
	defer rows.Close()

	var languages []Language
	for rows.Next() {
		var l Language
		if err := rows.Scan(&l.Code, &l.Name, &l.NativeName, &l.IsEnabled, &l.IsExperimental, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}
		languages = append(languages, l)
	}

	return languages, nil
}

// CreateLanguage creates a new language
func (db *DB) CreateLanguage(ctx context.Context, code, name, nativeName string, isEnabled, isExperimental bool) (*Language, error) {
	query := `
		INSERT INTO languages (code, name, native_name, is_enabled, is_experimental)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING code, name, native_name, is_enabled, is_experimental, created_at
	`

	lang := &Language{}
	err := db.QueryRowContext(ctx, query, code, name, nativeName, isEnabled, isExperimental).Scan(
		&lang.Code, &lang.Name, &lang.NativeName, &lang.IsEnabled, &lang.IsExperimental, &lang.CreatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to create language: %w", err)
	}

	return lang, nil
}

// UpdateLanguage updates an existing language
func (db *DB) UpdateLanguage(ctx context.Context, code, name, nativeName string, isEnabled, isExperimental *bool) (*Language, error) {
	query := `
		UPDATE languages 
		SET name = COALESCE(NULLIF($2, ''), name),
		    native_name = COALESCE(NULLIF($3, ''), native_name),
		    is_enabled = COALESCE($4, is_enabled),
		    is_experimental = COALESCE($5, is_experimental)
		WHERE code = $1
		RETURNING code, name, native_name, is_enabled, is_experimental, created_at
	`

	lang := &Language{}
	err := db.QueryRowContext(ctx, query, code, name, nativeName, isEnabled, isExperimental).Scan(
		&lang.Code, &lang.Name, &lang.NativeName, &lang.IsEnabled, &lang.IsExperimental, &lang.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update language: %w", err)
	}

	return lang, nil
}

// DeleteLanguage removes a language
func (db *DB) DeleteLanguage(ctx context.Context, code string) error {
	query := `DELETE FROM languages WHERE code = $1`

	result, err := db.ExecContext(ctx, query, code)
	if err != nil {
		return fmt.Errorf("failed to delete language: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ============================================================================
// ANALYTICS
// ============================================================================

// MessageAnalytics represents aggregated message analytics
type MessageAnalytics struct {
	Intent      string  `json:"intent"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	SampleQuery string  `json:"sample_query"`
}

// GetMessageAnalytics analyzes user messages by extracting common keywords/topics
func (db *DB) GetMessageAnalytics(ctx context.Context, since time.Time, limit int) ([]MessageAnalytics, error) {
	// Analyze user messages for common topics using keyword extraction
	// This is a simplified approach - a more sophisticated version would use NLP
	query := `
		WITH message_keywords AS (
			SELECT 
				CASE 
					WHEN content ILIKE '%nausea%' OR content ILIKE '%sick%' OR content ILIKE '%vomit%' THEN 'nausea_morning_sickness'
					WHEN content ILIKE '%kick%' OR content ILIKE '%movement%' OR content ILIKE '%moving%' THEN 'baby_movement'
					WHEN content ILIKE '%cramp%' OR content ILIKE '%pain%' OR content ILIKE '%hurt%' THEN 'pain_cramps'
					WHEN content ILIKE '%diet%' OR content ILIKE '%eat%' OR content ILIKE '%food%' OR content ILIKE '%nutrition%' THEN 'diet_nutrition'
					WHEN content ILIKE '%sleep%' OR content ILIKE '%tired%' OR content ILIKE '%fatigue%' THEN 'sleep_fatigue'
					WHEN content ILIKE '%doctor%' OR content ILIKE '%appointment%' OR content ILIKE '%checkup%' THEN 'medical_appointments'
					WHEN content ILIKE '%week%' OR content ILIKE '%trimester%' OR content ILIKE '%month%' THEN 'pregnancy_timeline'
					WHEN content ILIKE '%exercise%' OR content ILIKE '%workout%' OR content ILIKE '%yoga%' THEN 'exercise_fitness'
					WHEN content ILIKE '%anxiety%' OR content ILIKE '%stress%' OR content ILIKE '%worried%' THEN 'mental_health'
					WHEN content ILIKE '%weight%' OR content ILIKE '%gain%' THEN 'weight_changes'
					ELSE 'general_questions'
				END AS topic,
				content
			FROM messages
			WHERE role = 'user' AND created_at >= $1
		),
		topic_counts AS (
			SELECT 
				topic,
				COUNT(*) as count,
				(SELECT content FROM message_keywords mk2 WHERE mk2.topic = mk.topic LIMIT 1) as sample
			FROM message_keywords mk
			GROUP BY topic
		),
		total AS (
			SELECT SUM(count) as total_count FROM topic_counts
		)
		SELECT 
			tc.topic,
			tc.count,
			ROUND((tc.count::numeric / NULLIF(t.total_count, 0) * 100), 2) as percentage,
			tc.sample
		FROM topic_counts tc, total t
		ORDER BY tc.count DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query message analytics: %w", err)
	}
	defer rows.Close()

	var analytics []MessageAnalytics
	for rows.Next() {
		var a MessageAnalytics
		if err := rows.Scan(&a.Intent, &a.Count, &a.Percentage, &a.SampleQuery); err != nil {
			return nil, fmt.Errorf("failed to scan analytics: %w", err)
		}
		analytics = append(analytics, a)
	}

	return analytics, nil
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers        int            `json:"total_users"`
	ActiveUsers7Days  int            `json:"active_users_7_days"`
	ActiveUsers30Days int            `json:"active_users_30_days"`
	UsersByPlan       map[string]int `json:"users_by_plan"`
	UsersByLanguage   map[string]int `json:"users_by_language"`
}

// GetUserStats returns user statistics
func (db *DB) GetUserStats(ctx context.Context) (*UserStats, error) {
	stats := &UserStats{
		UsersByPlan:     make(map[string]int),
		UsersByLanguage: make(map[string]int),
	}

	// Total users
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers); err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Active users in last 7 days
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM messages WHERE created_at >= NOW() - INTERVAL '7 days'
	`).Scan(&stats.ActiveUsers7Days); err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}

	// Active users in last 30 days
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM messages WHERE created_at >= NOW() - INTERVAL '30 days'
	`).Scan(&stats.ActiveUsers30Days); err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}

	// Users by plan
	rows, err := db.QueryContext(ctx, `
		SELECT p.code, COUNT(s.user_id) 
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.status = 'active'
		GROUP BY p.code
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by plan: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var code string
		var count int
		if err := rows.Scan(&code, &count); err != nil {
			return nil, fmt.Errorf("failed to scan plan count: %w", err)
		}
		stats.UsersByPlan[code] = count
	}

	// Users by language
	rows, err = db.QueryContext(ctx, `
		SELECT preferred_language, COUNT(*) 
		FROM users 
		GROUP BY preferred_language
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by language: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lang string
		var count int
		if err := rows.Scan(&lang, &count); err != nil {
			return nil, fmt.Errorf("failed to scan language count: %w", err)
		}
		stats.UsersByLanguage[lang] = count
	}

	return stats, nil
}

// VoiceCall represents a voice call record
type VoiceCall struct {
	CallSID     string    `json:"call_sid"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	PhoneNumber string    `json:"phone_number"`
	Duration    int       `json:"duration_seconds"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetVoiceCallHistory returns voice call history (placeholder - requires voice_calls table)
func (db *DB) GetVoiceCallHistory(ctx context.Context, since time.Time, limit int) ([]VoiceCall, error) {
	// Note: This requires a voice_calls table to be added to the schema
	// For now, return empty slice
	// TODO: Add voice_calls table migration and implement this query
	return []VoiceCall{}, nil
}

// isDuplicateKeyError checks if an error is a duplicate key violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL duplicate key error code: 23505
	return err.Error() == "pq: duplicate key value violates unique constraint" ||
		contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
