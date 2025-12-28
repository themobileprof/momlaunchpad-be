package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	if cfg.MaxConnections > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{sqlDB}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// User represents a user in the database
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         *string
	Language     string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Message represents a chat message
type Message struct {
	ID        string
	UserID    string
	Role      string
	Content   string
	CreatedAt time.Time
}

// UserFact represents a long-term memory fact
type UserFact struct {
	ID         string
	UserID     string
	Key        string
	Value      string
	Confidence float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Reminder represents a calendar reminder
type Reminder struct {
	ID           string
	UserID       string
	Title        string
	Description  *string
	ReminderTime time.Time
	IsCompleted  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Language represents a supported language
type Language struct {
	Code           string
	Name           string
	NativeName     string
	IsEnabled      bool
	IsExperimental bool
	CreatedAt      time.Time
}
