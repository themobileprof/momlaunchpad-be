package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
)

const userSelectSQL = `
	SELECT id, email, password_hash, display_name, preferred_language, currency,
	       pregnancy_week, pregnancy_start_date, expected_delivery_date,
	       is_first_pregnancy, primary_concern, diet_preference,
	       journey_stage, journey_stage_since, baby_birth_date, loss_date,
	       profile_photo_url, country, country_code, state_province, city,
	       community_onboarding_completed_at,
	       savings_goal, is_admin, onboarding_completed_at, created_at, updated_at
	FROM users`

func scanUser(scanner interface {
	Scan(dest ...any) error
}) (*User, error) {
	user := &User{}
	err := scanner.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.Language, &user.Currency, &user.PregnancyWeek, &user.PregnancyStartDate,
		&user.ExpectedDeliveryDate, &user.IsFirstPregnancy, &user.PrimaryConcern,
		&user.DietPreference, &user.JourneyStage, &user.JourneyStageSince,
		&user.BabyBirthDate, &user.LossDate,
		&user.ProfilePhotoURL, &user.Country, &user.CountryCode, &user.StateProvince, &user.City,
		&user.CommunityOnboardingAt,
		&user.SavingsGoal, &user.IsAdmin, &user.OnboardingCompletedAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UserProfileUpdate contains editable profile fields.
type UserProfileUpdate struct {
	Name                 *string
	Language             *string
	PregnancyWeek        *int
	PregnancyStartDate   *time.Time
	ExpectedDeliveryDate *time.Time
	IsFirstPregnancy     *bool
	PrimaryConcern       *string
	DietPreference       *string
	JourneyStage         *string
	JourneyStageSince    *time.Time
	BabyBirthDate        *time.Time
	LossDate             *time.Time
	ProfilePhotoURL      *string
	Country              *string
	CountryCode          *string
	StateProvince        *string
	City                 *string
}

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (email, password_hash, display_name, preferred_language, is_admin)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	return db.QueryRowContext(ctx, query,
		user.Email, user.PasswordHash, user.Name, user.Language, user.IsAdmin,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// CreateOAuthUser creates a user authenticated via an external OAuth provider.
func (db *DB) CreateOAuthUser(ctx context.Context, user *User, authProvider string) error {
	query := `
		INSERT INTO users (email, password_hash, display_name, preferred_language, is_admin, auth_provider)
		VALUES ($1, NULL, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	return db.QueryRowContext(ctx, query,
		user.Email, user.Name, user.Language, user.IsAdmin, authProvider,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := userSelectSQL + `
		WHERE email = $1
	`

	user, err := scanUser(db.QueryRowContext(ctx, query, email))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := userSelectSQL + `
		WHERE id = $1
	`

	user, err := scanUser(db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// SaveMessage saves a chat message
func (db *DB) SaveMessage(ctx context.Context, userID, conversationID, role, content string) (*Message, error) {
	query := `
		INSERT INTO messages (user_id, conversation_id, role, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, conversation_id, role, content, created_at
	`

	msg := &Message{}
	err := db.QueryRowContext(ctx, query, userID, conversationID, role, content).Scan(
		&msg.ID, &msg.UserID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return msg, nil
}

// GetRecentMessages retrieves the N most recent messages for a user
func (db *DB) GetRecentMessages(ctx context.Context, userID string, limit int) ([]Message, error) {
	query := `
		SELECT id, user_id, role, content, created_at
		FROM messages
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	messages := make([]Message, 0, limit)
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SaveOrUpdateFact saves or updates a user fact
func (db *DB) SaveOrUpdateFact(ctx context.Context, userID, key, value string, confidence float64) (*UserFact, error) {
	query := `
		INSERT INTO user_facts (user_id, key, value, confidence)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, key) 
		DO UPDATE SET value = $3, confidence = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_facts.confidence <= $4
		RETURNING id, user_id, key, value, confidence, created_at, updated_at
	`

	fact := &UserFact{}
	err := db.QueryRowContext(ctx, query, userID, key, value, confidence).Scan(
		&fact.ID, &fact.UserID, &fact.Key, &fact.Value,
		&fact.Confidence, &fact.CreatedAt, &fact.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save fact: %w", err)
	}

	return fact, nil
}

// SymptomInsert captures a symptom logged from chat with source links and summary.
type SymptomInsert struct {
	UserID             string
	ConversationID     string
	MessageID          string
	SymptomType        string
	Description        string
	Summary            string
	Severity           string
	Frequency          string
	OnsetTime          string
	AssociatedSymptoms []string
}

// SaveSymptom saves a new symptom record linked to the source chat message when available.
func (db *DB) SaveSymptom(ctx context.Context, input SymptomInsert) (string, error) {
	query := `
		INSERT INTO symptoms (
			user_id, conversation_id, message_id, symptom_type, description, summary,
			severity, frequency, onset_time, associated_symptoms
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	convID := sql.NullString{String: input.ConversationID, Valid: input.ConversationID != ""}
	msgID := sql.NullString{String: input.MessageID, Valid: input.MessageID != ""}
	summary := sql.NullString{String: input.Summary, Valid: input.Summary != ""}

	var symptomID string
	err := db.QueryRowContext(
		ctx,
		query,
		input.UserID,
		convID,
		msgID,
		input.SymptomType,
		input.Description,
		summary,
		input.Severity,
		input.Frequency,
		input.OnsetTime,
		pq.Array(input.AssociatedSymptoms),
	).Scan(&symptomID)
	if err != nil {
		return "", fmt.Errorf("failed to save symptom: %w", err)
	}

	return symptomID, nil
}

// GetRecentSymptoms retrieves recent symptoms for a user
func (db *DB) GetRecentSymptoms(ctx context.Context, userID string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, symptom_type, description, summary, severity, frequency, onset_time, 
		       associated_symptoms, is_resolved, reported_at, resolved_at,
		       conversation_id, message_id
		FROM symptoms
		WHERE user_id = $1
		ORDER BY reported_at DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent symptoms: %w", err)
	}
	defer rows.Close()

	symptoms := make([]map[string]interface{}, 0)
	for rows.Next() {
		symptom, err := scanSymptomRow(rows)
		if err != nil {
			return nil, err
		}
		symptoms = append(symptoms, symptom)
	}

	return symptoms, nil
}

// GetSymptomHistory retrieves all symptoms for a user with optional filters
func (db *DB) GetSymptomHistory(ctx context.Context, userID string, symptomType string, limit int) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if symptomType != "" {
		query = `
			SELECT id, symptom_type, description, summary, severity, frequency, onset_time, 
			       associated_symptoms, is_resolved, reported_at, resolved_at,
			       conversation_id, message_id
			FROM symptoms
			WHERE user_id = $1 AND symptom_type = $2
			ORDER BY reported_at DESC
			LIMIT $3
		`
		args = []interface{}{userID, symptomType, limit}
	} else {
		query = `
			SELECT id, symptom_type, description, summary, severity, frequency, onset_time, 
			       associated_symptoms, is_resolved, reported_at, resolved_at,
			       conversation_id, message_id
			FROM symptoms
			WHERE user_id = $1
			ORDER BY reported_at DESC
			LIMIT $2
		`
		args = []interface{}{userID, limit}
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get symptom history: %w", err)
	}
	defer rows.Close()

	symptoms := make([]map[string]interface{}, 0)
	for rows.Next() {
		symptom, err := scanSymptomRow(rows)
		if err != nil {
			return nil, err
		}
		symptoms = append(symptoms, symptom)
	}

	return symptoms, nil
}

func scanSymptomRow(rows *sql.Rows) (map[string]interface{}, error) {
	var (
		id                 string
		symptomType        string
		description        string
		summary            sql.NullString
		severity           sql.NullString
		frequency          sql.NullString
		onsetTime          sql.NullString
		associatedSymptoms []string
		isResolved         bool
		reportedAt         time.Time
		resolvedAt         *time.Time
		conversationID     sql.NullString
		messageID          sql.NullString
	)

	err := rows.Scan(
		&id, &symptomType, &description, &summary, &severity, &frequency, &onsetTime,
		pq.Array(&associatedSymptoms), &isResolved, &reportedAt, &resolvedAt,
		&conversationID, &messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan symptom: %w", err)
	}

	return map[string]interface{}{
		"id":                  id,
		"symptom_type":        symptomType,
		"description":         description,
		"summary":             nullStringValue(summary),
		"severity":            nullStringValue(severity),
		"frequency":           nullStringValue(frequency),
		"onset_time":          nullStringValue(onsetTime),
		"associated_symptoms": associatedSymptoms,
		"is_resolved":         isResolved,
		"reported_at":         reportedAt,
		"resolved_at":         resolvedAt,
		"conversation_id":     nullStringValue(conversationID),
		"message_id":          nullStringValue(messageID),
	}, nil
}

func nullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// UpdateSymptomSummary stores a one-sentence AI summary for display in the health tracker.
func (db *DB) UpdateSymptomSummary(ctx context.Context, symptomID, summary string) error {
	query := `
		UPDATE symptoms
		SET summary = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := db.ExecContext(ctx, query, symptomID, summary)
	if err != nil {
		return fmt.Errorf("failed to update symptom summary: %w", err)
	}
	return nil
}

// MarkSymptomResolved marks a symptom as resolved.
func (db *DB) MarkSymptomResolved(ctx context.Context, symptomID, userID string) error {
	query := `
		UPDATE symptoms 
		SET is_resolved = true, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND is_resolved = false
	`

	result, err := db.ExecContext(ctx, query, symptomID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark symptom resolved: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check symptom resolve result: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetUserFacts retrieves all facts for a user
func (db *DB) GetUserFacts(ctx context.Context, userID string) ([]UserFact, error) {
	query := `
		SELECT id, user_id, key, value, confidence, created_at, updated_at
		FROM user_facts
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get facts: %w", err)
	}
	defer rows.Close()

	facts := make([]UserFact, 0)
	for rows.Next() {
		var fact UserFact
		if err := rows.Scan(&fact.ID, &fact.UserID, &fact.Key, &fact.Value,
			&fact.Confidence, &fact.CreatedAt, &fact.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// GetSystemSetting retrieves a system setting by key
func (db *DB) GetSystemSetting(ctx context.Context, key string) (*SystemSetting, error) {
	query := `
		SELECT key, value, description, updated_at
		FROM system_settings
		WHERE key = $1
	`

	var setting SystemSetting
	var description sql.NullString
	err := db.QueryRowContext(ctx, query, key).Scan(
		&setting.Key, &setting.Value, &description, &setting.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	if description.Valid {
		setting.Description = &description.String
	}

	return &setting, nil
}

// UpdateSystemSetting updates a system setting
func (db *DB) UpdateSystemSetting(ctx context.Context, key, value string) error {
	query := `
		UPDATE system_settings 
		SET value = $2
		WHERE key = $1
	`

	result, err := db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}

	return nil
}

// GetAllSystemSettings retrieves all system settings
func (db *DB) GetAllSystemSettings(ctx context.Context) ([]SystemSetting, error) {
	query := `
		SELECT key, value, description, updated_at
		FROM system_settings
		ORDER BY key
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	defer rows.Close()

	settings := make([]SystemSetting, 0)
	for rows.Next() {
		var setting SystemSetting
		var description sql.NullString
		if err := rows.Scan(&setting.Key, &setting.Value, &description, &setting.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}

		if description.Valid {
			setting.Description = &description.String
		}

		settings = append(settings, setting)
	}

	return settings, nil
}

// CreateReminder creates a new reminder
func (db *DB) CreateReminder(ctx context.Context, reminder *Reminder) error {
	query := `
		INSERT INTO reminders (user_id, title, description, reminder_time, is_completed, community_event_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	return db.QueryRowContext(ctx, query,
		reminder.UserID, reminder.Title, reminder.Description,
		reminder.ReminderTime, reminder.IsCompleted, reminder.CommunityEventID,
	).Scan(&reminder.ID, &reminder.CreatedAt, &reminder.UpdatedAt)
}

func scanReminder(scanner interface {
	Scan(dest ...any) error
}) (Reminder, error) {
	var reminder Reminder
	var communityEventID sql.NullString
	if err := scanner.Scan(
		&reminder.ID, &reminder.UserID, &reminder.Title, &reminder.Description,
		&reminder.ReminderTime, &reminder.IsCompleted, &communityEventID,
		&reminder.CreatedAt, &reminder.UpdatedAt,
	); err != nil {
		return Reminder{}, err
	}
	if communityEventID.Valid {
		id := communityEventID.String
		reminder.CommunityEventID = &id
	}
	return reminder, nil
}

// GetUserReminders retrieves all reminders for a user
func (db *DB) GetUserReminders(ctx context.Context, userID string) ([]Reminder, error) {
	query := `
		SELECT id, user_id, title, description, reminder_time, is_completed, community_event_id, created_at, updated_at
		FROM reminders
		WHERE user_id = $1
		ORDER BY reminder_time ASC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminders: %w", err)
	}
	defer rows.Close()

	reminders := make([]Reminder, 0)
	for rows.Next() {
		reminder, err := scanReminder(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

// GetEnabledLanguages retrieves all enabled languages
func (db *DB) GetEnabledLanguages(ctx context.Context) ([]Language, error) {
	query := `
		SELECT code, name, native_name, is_enabled, is_experimental, created_at
		FROM languages
		WHERE is_enabled = TRUE
		ORDER BY code
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get languages: %w", err)
	}
	defer rows.Close()

	languages := make([]Language, 0)
	for rows.Next() {
		var lang Language
		if err := rows.Scan(&lang.Code, &lang.Name, &lang.NativeName,
			&lang.IsEnabled, &lang.IsExperimental, &lang.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}
		languages = append(languages, lang)
	}

	return languages, nil
}

// GetReminderByID retrieves a reminder by ID
func (db *DB) GetReminderByID(ctx context.Context, id string) (*Reminder, error) {
	query := `
		SELECT id, user_id, title, description, reminder_time, is_completed, community_event_id, created_at, updated_at
		FROM reminders
		WHERE id = $1
	`

	reminder, err := scanReminder(db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return &reminder, nil
}

// UpdateReminder updates a reminder
func (db *DB) UpdateReminder(ctx context.Context, reminder *Reminder) error {
	query := `
		UPDATE reminders
		SET title = $1, description = $2, reminder_time = $3, is_completed = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`

	result, err := db.ExecContext(ctx, query,
		reminder.Title, reminder.Description, reminder.ReminderTime,
		reminder.IsCompleted, reminder.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update reminder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteReminder deletes a reminder
func (db *DB) DeleteReminder(ctx context.Context, id string) error {
	query := `DELETE FROM reminders WHERE id = $1`

	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateUserEDD updates a user's expected delivery date
func (db *DB) UpdateUserEDD(ctx context.Context, userID string, edd *time.Time) error {
	query := `
		UPDATE users
		SET expected_delivery_date = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := db.ExecContext(ctx, query, edd, userID)
	if err != nil {
		return fmt.Errorf("failed to update EDD: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateUserProfile updates display name and preferred language.
func (db *DB) UpdateUserProfile(ctx context.Context, userID string, name, language *string) error {
	return db.UpdateUserProfileDetails(ctx, userID, UserProfileUpdate{
		Name:     name,
		Language: language,
	})
}

// UpdateUserProfileDetails updates editable profile and pregnancy fields.
func (db *DB) UpdateUserProfileDetails(ctx context.Context, userID string, update UserProfileUpdate) error {
	query := `
		UPDATE users
		SET display_name = COALESCE($1, display_name),
		    preferred_language = COALESCE($2, preferred_language),
		    pregnancy_week = COALESCE($3, pregnancy_week),
		    pregnancy_start_date = COALESCE($4, pregnancy_start_date),
		    expected_delivery_date = COALESCE($5, expected_delivery_date),
		    is_first_pregnancy = COALESCE($6, is_first_pregnancy),
		    primary_concern = COALESCE($7, primary_concern),
		    diet_preference = COALESCE($8, diet_preference),
		    journey_stage = COALESCE($9, journey_stage),
		    journey_stage_since = COALESCE($10, journey_stage_since),
		    baby_birth_date = COALESCE($11, baby_birth_date),
		    loss_date = COALESCE($12, loss_date),
		    profile_photo_url = COALESCE($13, profile_photo_url),
		    country = COALESCE($14, country),
		    country_code = COALESCE($15, country_code),
		    state_province = COALESCE($16, state_province),
		    city = COALESCE($17, city),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $18
	`

	result, err := db.ExecContext(ctx, query,
		update.Name,
		update.Language,
		update.PregnancyWeek,
		update.PregnancyStartDate,
		update.ExpectedDeliveryDate,
		update.IsFirstPregnancy,
		update.PrimaryConcern,
		update.DietPreference,
		update.JourneyStage,
		update.JourneyStageSince,
		update.BabyBirthDate,
		update.LossDate,
		update.ProfilePhotoURL,
		update.Country,
		update.CountryCode,
		update.StateProvince,
		update.City,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// CompleteOnboarding marks onboarding as finished for a user.
func (db *DB) CompleteOnboarding(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET onboarding_completed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to complete onboarding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateUserSavingsGoal updates a user's savings goal
func (db *DB) UpdateUserSavingsGoal(ctx context.Context, userID string, goal float64) error {
	query := `
		UPDATE users
		SET savings_goal = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := db.ExecContext(ctx, query, goal, userID)
	if err != nil {
		return fmt.Errorf("failed to update savings goal: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateUserCurrency updates a user's preferred currency
func (db *DB) UpdateUserCurrency(ctx context.Context, userID, currency string) error {
	query := `
		UPDATE users
		SET currency = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := db.ExecContext(ctx, query, currency, userID)
	if err != nil {
		return fmt.Errorf("failed to update currency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// CreateSavingsEntry creates a new savings entry
func (db *DB) CreateSavingsEntry(ctx context.Context, entry *SavingsEntry) error {
	query := `
		INSERT INTO savings_entries (user_id, amount, description, entry_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	return db.QueryRowContext(ctx, query,
		entry.UserID, entry.Amount, entry.Description, entry.EntryDate,
	).Scan(&entry.ID, &entry.CreatedAt)
}

// GetUserSavingsEntries retrieves all savings entries for a user
func (db *DB) GetUserSavingsEntries(ctx context.Context, userID string) ([]SavingsEntry, error) {
	query := `
		SELECT id, user_id, amount, description, entry_date, created_at
		FROM savings_entries
		WHERE user_id = $1
		ORDER BY entry_date DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get savings entries: %w", err)
	}
	defer rows.Close()

	entries := make([]SavingsEntry, 0)
	for rows.Next() {
		var entry SavingsEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Amount, &entry.Description,
			&entry.EntryDate, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan savings entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetTotalSavings calculates the total savings for a user
func (db *DB) GetTotalSavings(ctx context.Context, userID string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM savings_entries
		WHERE user_id = $1
	`

	var total float64
	err := db.QueryRowContext(ctx, query, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total savings: %w", err)
	}

	return total, nil
}

// OAuth Provider Queries

// CreateOAuthProvider links an OAuth provider to a user
func (db *DB) CreateOAuthProvider(ctx context.Context, userID, provider, providerUserID, email string) error {
	query := `
		INSERT INTO oauth_providers (user_id, provider, provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (provider, provider_user_id) 
		DO UPDATE SET updated_at = NOW()
	`

	_, err := db.ExecContext(ctx, query, userID, provider, providerUserID, email)
	if err != nil {
		return fmt.Errorf("failed to create OAuth provider: %w", err)
	}

	return nil
}

// GetOAuthProvider retrieves OAuth provider info
func (db *DB) GetOAuthProvider(ctx context.Context, provider, providerUserID string) (string, error) {
	query := `
		SELECT user_id 
		FROM oauth_providers 
		WHERE provider = $1 AND provider_user_id = $2
	`

	var userID string
	err := db.QueryRowContext(ctx, query, provider, providerUserID).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	return userID, nil
}

// GetUserOAuthProviders lists all OAuth providers for a user
func (db *DB) GetUserOAuthProviders(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT provider 
		FROM oauth_providers 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query OAuth providers: %w", err)
	}
	defer rows.Close()

	var providers []string
	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// FindUserByEmailAcrossProviders finds a user by email regardless of auth method
// This enables email-based account linking across Google, Apple, and local auth
func (db *DB) FindUserByEmailAcrossProviders(ctx context.Context, email string) (*User, error) {
	return db.GetUserByEmail(ctx, email)
}
