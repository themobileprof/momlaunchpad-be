package db

import (
	"context"

	"github.com/themobileprof/momlaunchpad-be/internal/memory"
)

// MemoryAdapter adapts DB to implement memory.DBInterface
type MemoryAdapter struct {
	db *DB
}

// NewMemoryAdapter creates a new adapter
func NewMemoryAdapter(db *DB) *MemoryAdapter {
	return &MemoryAdapter{db: db}
}

// GetRecentMessages implements memory.DBInterface
func (a *MemoryAdapter) GetRecentMessages(userID string, limit int) ([]memory.Message, error) {
	ctx := context.Background()
	dbMessages, err := a.db.GetRecentMessages(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	// Convert DB messages to memory.Message format
	messages := make([]memory.Message, len(dbMessages))
	for i, msg := range dbMessages {
		messages[i] = memory.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
		}
	}

	return messages, nil
}
