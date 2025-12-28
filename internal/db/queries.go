package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
)

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

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, display_name, preferred_language, is_admin, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &User{}
	err := db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.Language, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
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
	query := `
		SELECT id, email, password_hash, display_name, preferred_language, is_admin, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.Language, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// SaveMessage saves a chat message
func (db *DB) SaveMessage(ctx context.Context, userID, role, content string) (*Message, error) {
	query := `
		INSERT INTO messages (user_id, role, content)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, role, content, created_at
	`

	msg := &Message{}
	err := db.QueryRowContext(ctx, query, userID, role, content).Scan(
		&msg.ID, &msg.UserID, &msg.Role, &msg.Content, &msg.CreatedAt,
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
		WHERE user_facts.confidence < $4
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

// CreateReminder creates a new reminder
func (db *DB) CreateReminder(ctx context.Context, reminder *Reminder) error {
	query := `
		INSERT INTO reminders (user_id, title, description, reminder_time, is_completed)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	return db.QueryRowContext(ctx, query,
		reminder.UserID, reminder.Title, reminder.Description,
		reminder.ReminderTime, reminder.IsCompleted,
	).Scan(&reminder.ID, &reminder.CreatedAt, &reminder.UpdatedAt)
}

// GetUserReminders retrieves all reminders for a user
func (db *DB) GetUserReminders(ctx context.Context, userID string) ([]Reminder, error) {
	query := `
		SELECT id, user_id, title, description, reminder_time, is_completed, created_at, updated_at
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
		var reminder Reminder
		if err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.Title, &reminder.Description,
			&reminder.ReminderTime, &reminder.IsCompleted, &reminder.CreatedAt, &reminder.UpdatedAt); err != nil {
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
		SELECT id, user_id, title, description, reminder_time, is_completed, created_at, updated_at
		FROM reminders
		WHERE id = $1
	`

	reminder := &Reminder{}
	err := db.QueryRowContext(ctx, query, id).Scan(
		&reminder.ID, &reminder.UserID, &reminder.Title, &reminder.Description,
		&reminder.ReminderTime, &reminder.IsCompleted, &reminder.CreatedAt, &reminder.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return reminder, nil
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
