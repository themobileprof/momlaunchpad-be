package db

import (
	"context"
	"database/sql"
	"fmt"
)

const vitalReadingSelectColumns = `
	id, user_id, recorded_at, blood_pressure_systolic, blood_pressure_diastolic,
	weight_kg, heart_rate_bpm, temperature_celsius, fundal_height_cm,
	fetal_heart_rate_bpm, gestational_age_weeks, notes, source,
	created_at, updated_at
`

func scanVitalReading(scanner interface {
	Scan(dest ...any) error
}, reading *VitalReading) error {
	return scanner.Scan(
		&reading.ID, &reading.UserID, &reading.RecordedAt,
		&reading.BloodPressureSystolic, &reading.BloodPressureDiastolic,
		&reading.WeightKg, &reading.HeartRateBpm, &reading.TemperatureCelsius,
		&reading.FundalHeightCm, &reading.FetalHeartRateBpm, &reading.GestationalAgeWeeks,
		&reading.Notes, &reading.Source,
		&reading.CreatedAt, &reading.UpdatedAt,
	)
}

// CreateVitalReading inserts a new vital reading.
func (db *DB) CreateVitalReading(ctx context.Context, reading *VitalReading) error {
	if reading.Source == "" {
		reading.Source = "manual"
	}

	query := `
		INSERT INTO vital_readings (
			user_id, recorded_at, blood_pressure_systolic, blood_pressure_diastolic,
			weight_kg, heart_rate_bpm, temperature_celsius, fundal_height_cm,
			fetal_heart_rate_bpm, gestational_age_weeks, notes, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	return db.QueryRowContext(ctx, query,
		reading.UserID, reading.RecordedAt,
		reading.BloodPressureSystolic, reading.BloodPressureDiastolic,
		reading.WeightKg, reading.HeartRateBpm, reading.TemperatureCelsius,
		reading.FundalHeightCm, reading.FetalHeartRateBpm, reading.GestationalAgeWeeks,
		reading.Notes, reading.Source,
	).Scan(&reading.ID, &reading.CreatedAt, &reading.UpdatedAt)
}

// GetUserVitalReadings returns vital readings for a user, newest first.
func (db *DB) GetUserVitalReadings(ctx context.Context, userID string, limit int) ([]VitalReading, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT ` + vitalReadingSelectColumns + `
		FROM vital_readings
		WHERE user_id = $1
		ORDER BY recorded_at DESC, created_at DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get vital readings: %w", err)
	}
	defer rows.Close()

	readings := make([]VitalReading, 0)
	for rows.Next() {
		var reading VitalReading
		if err := scanVitalReading(rows, &reading); err != nil {
			return nil, fmt.Errorf("failed to scan vital reading: %w", err)
		}
		readings = append(readings, reading)
	}

	return readings, nil
}

// GetVitalReadingByID returns a single vital reading.
func (db *DB) GetVitalReadingByID(ctx context.Context, id string) (*VitalReading, error) {
	query := `
		SELECT ` + vitalReadingSelectColumns + `
		FROM vital_readings
		WHERE id = $1
	`

	reading := &VitalReading{}
	err := scanVitalReading(db.QueryRowContext(ctx, query, id), reading)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vital reading: %w", err)
	}

	return reading, nil
}

// DeleteVitalReading removes a vital reading.
func (db *DB) DeleteVitalReading(ctx context.Context, id string) error {
	result, err := db.ExecContext(ctx, `DELETE FROM vital_readings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete vital reading: %w", err)
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
