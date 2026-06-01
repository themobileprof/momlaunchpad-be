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

// NewFromURL creates a new database connection from a connection string
func NewFromURL(connectionString string) (*DB, error) {
	sqlDB, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Optimized connection pool settings for high throughput
	sqlDB.SetMaxOpenConns(50)                 // Increased from 25 for better concurrency
	sqlDB.SetMaxIdleConns(25)                 // Increased from 5 to reduce connection churn
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Keep connections alive
	sqlDB.SetConnMaxIdleTime(2 * time.Minute) // Close idle connections after 2 min

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
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
	ID                    string     `json:"id"`
	Email                 string     `json:"email"`
	PasswordHash          string     `json:"-"` // Don't expose password hash
	Name                  *string    `json:"name"`
	Language              string     `json:"language"`
	Currency              string     `json:"currency"`
	PregnancyWeek         *int       `json:"pregnancy_week"`
	PregnancyStartDate    *time.Time `json:"pregnancy_start_date"`
	ExpectedDeliveryDate  *time.Time `json:"expected_delivery_date"`
	IsFirstPregnancy      *bool      `json:"is_first_pregnancy"`
	PrimaryConcern        *string    `json:"primary_concern"`
	DietPreference        *string    `json:"diet_preference"`
	JourneyStage          *string    `json:"journey_stage"`
	JourneyStageSince     *time.Time `json:"journey_stage_since"`
	BabyBirthDate         *time.Time `json:"baby_birth_date"`
	LossDate              *time.Time `json:"loss_date"`
	ProfilePhotoURL       *string    `json:"profile_photo_url"`
	Country               *string    `json:"country"`
	CountryCode           *string    `json:"country_code"`
	StateProvince         *string    `json:"state_province"`
	City                  *string    `json:"city"`
	CommunityOnboardingAt *time.Time `json:"community_onboarding_completed_at,omitempty"`
	SavingsGoal           *float64   `json:"savings_goal"`
	IsAdmin               bool       `json:"is_admin"`
	OnboardingCompletedAt *time.Time `json:"onboarding_completed_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// Conversation represents a chat session
type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     *string   `json:"title"`
	IsStarred bool      `json:"is_starred"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message represents a chat message
type Message struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// UserFact represents a long-term memory fact
type UserFact struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Reminder represents a calendar reminder
type Reminder struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	Title             string    `json:"title"`
	Description       *string   `json:"description"`
	ReminderTime      time.Time `json:"reminder_time"`
	IsCompleted       bool      `json:"is_completed"`
	CommunityEventID  *string   `json:"community_event_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DoctorVisit represents a prenatal/medical visit record (micro EMR).
type DoctorVisit struct {
	ID                     string
	UserID                 string
	VisitDate              time.Time
	VisitType              string
	ProviderName           *string
	FacilityName           *string
	ChiefComplaint         *string
	ClinicalNotes          *string
	Diagnosis              *string
	TreatmentPlan          *string
	FollowUpInstructions   *string
	BloodPressureSystolic  *int
	BloodPressureDiastolic *int
	WeightKg               *float64
	HeartRateBpm           *int
	TemperatureCelsius     *float64
	FundalHeightCm         *float64
	FetalHeartRateBpm      *int
	GestationalAgeWeeks    *int
	Medications            []byte
	LabResults             []byte
	NextAppointmentAt      *time.Time
	NextAppointmentNotes   *string
	RecordedBy             string
	ProviderUserID         *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// VitalReading represents a standalone vital signs log entry.
type VitalReading struct {
	ID                     string
	UserID                 string
	RecordedAt             time.Time
	BloodPressureSystolic  *int
	BloodPressureDiastolic *int
	WeightKg               *float64
	HeartRateBpm           *int
	TemperatureCelsius     *float64
	FundalHeightCm         *float64
	FetalHeartRateBpm      *int
	GestationalAgeWeeks    *int
	Notes                  *string
	Source                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Language represents a supported language
type Language struct {
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	NativeName     string    `json:"native_name"`
	IsEnabled      bool      `json:"is_enabled"`
	IsExperimental bool      `json:"is_experimental"`
	CreatedAt      time.Time `json:"created_at"`
}

// SavingsEntry represents a savings record
type SavingsEntry struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Amount      float64   `json:"amount"`
	Description *string   `json:"description"`
	EntryDate   time.Time `json:"entry_date"`
	CreatedAt   time.Time `json:"created_at"`
}

// SystemSetting represents a system-wide configuration setting
type SystemSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description *string   `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
