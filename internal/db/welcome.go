package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WelcomeMessage is a cached daily welcome message for a user.
type WelcomeMessage struct {
	ID        string
	UserID    string
	CacheDate time.Time
	Message   string
	Source    string
	CreatedAt time.Time
}

// GetWelcomeMessage returns the cached message for a user on a given calendar date.
func (db *DB) GetWelcomeMessage(ctx context.Context, userID string, cacheDate time.Time) (*WelcomeMessage, error) {
	query := `
		SELECT id, user_id, cache_date, message, source, created_at
		FROM user_welcome_messages
		WHERE user_id = $1 AND cache_date = $2
	`

	msg := &WelcomeMessage{}
	err := db.QueryRowContext(ctx, query, userID, cacheDate.Format("2006-01-02")).Scan(
		&msg.ID, &msg.UserID, &msg.CacheDate, &msg.Message, &msg.Source, &msg.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get welcome message: %w", err)
	}

	return msg, nil
}

// SaveWelcomeMessage stores a daily welcome message (idempotent per user/date).
func (db *DB) SaveWelcomeMessage(ctx context.Context, userID string, cacheDate time.Time, message, source string) (*WelcomeMessage, error) {
	if source == "" {
		source = "gemini"
	}

	query := `
		INSERT INTO user_welcome_messages (user_id, cache_date, message, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, cache_date) DO NOTHING
		RETURNING id, user_id, cache_date, message, source, created_at
	`

	msg := &WelcomeMessage{}
	err := db.QueryRowContext(ctx, query, userID, cacheDate.Format("2006-01-02"), message, source).Scan(
		&msg.ID, &msg.UserID, &msg.CacheDate, &msg.Message, &msg.Source, &msg.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return db.GetWelcomeMessage(ctx, userID, cacheDate)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to save welcome message: %w", err)
	}

	return msg, nil
}
