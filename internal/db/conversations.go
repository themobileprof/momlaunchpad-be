package db

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateConversation creates a new conversation
func (db *DB) CreateConversation(ctx context.Context, userID string, title *string) (*Conversation, error) {
	query := `
		INSERT INTO conversations (user_id, title)
		VALUES ($1, $2)
		RETURNING id, user_id, title, is_starred, created_at, updated_at
	`
	
	row := db.QueryRowContext(ctx, query, userID, title)
	
	var c Conversation
	if err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.IsStarred, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}
	
	return &c, nil
}

// GetConversations retrieves a paginated list of conversations for a user
func (db *DB) GetConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	query := `
		SELECT id, user_id, title, is_starred, created_at, updated_at
		FROM conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()
	
	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.IsStarred, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, c)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}
	
	return conversations, nil
}

// GetConversation retrieves a specific conversation by ID
func (db *DB) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	query := `
		SELECT id, user_id, title, is_starred, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`
	
	row := db.QueryRowContext(ctx, query, id)
	
	var c Conversation
	if err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.IsStarred, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	
	return &c, nil
}

// UpdateConversation updates conversation details
func (db *DB) UpdateConversation(ctx context.Context, id string, title *string, isStarred *bool) (*Conversation, error) {
	// Build dynamic query
	query := "UPDATE conversations SET updated_at = NOW()"
	var args []interface{}
	argCount := 0
	
	if title != nil {
		argCount++
		query += fmt.Sprintf(", title = $%d", argCount)
		args = append(args, *title)
	}
	
	if isStarred != nil {
		argCount++
		query += fmt.Sprintf(", is_starred = $%d", argCount)
		args = append(args, *isStarred)
	}
	
	argCount++
	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, user_id, title, is_starred, created_at, updated_at", argCount)
	args = append(args, id)
	
	row := db.QueryRowContext(ctx, query, args...)
	
	var c Conversation
	if err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.IsStarred, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}
	
	return &c, nil
}

// DeleteConversation deletes a conversation and its messages (via cascade)
func (db *DB) DeleteConversation(ctx context.Context, id string) error {
	query := "DELETE FROM conversations WHERE id = $1"
	
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// GetMessagesByConversation retrieves messages for a specific conversation
func (db *DB) GetMessagesByConversation(ctx context.Context, conversationID string, limit, offset int) ([]Message, error) {
	query := `
		SELECT id, user_id, conversation_id, role, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := db.QueryContext(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()
	
	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.UserID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, m)
	}
	
	return messages, nil
}
