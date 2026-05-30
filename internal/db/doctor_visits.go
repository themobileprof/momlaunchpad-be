package db

import (
	"context"
	"database/sql"
	"fmt"
)

const doctorVisitSelectColumns = `
	id, user_id, visit_date, visit_type, provider_name, facility_name,
	chief_complaint, clinical_notes, diagnosis, treatment_plan, follow_up_instructions,
	blood_pressure_systolic, blood_pressure_diastolic, weight_kg, heart_rate_bpm,
	temperature_celsius, fundal_height_cm, fetal_heart_rate_bpm, gestational_age_weeks,
	medications, lab_results, next_appointment_at, next_appointment_notes,
	recorded_by, provider_user_id, created_at, updated_at
`

func scanDoctorVisit(scanner interface {
	Scan(dest ...any) error
}, visit *DoctorVisit) error {
	return scanner.Scan(
		&visit.ID, &visit.UserID, &visit.VisitDate, &visit.VisitType,
		&visit.ProviderName, &visit.FacilityName,
		&visit.ChiefComplaint, &visit.ClinicalNotes, &visit.Diagnosis,
		&visit.TreatmentPlan, &visit.FollowUpInstructions,
		&visit.BloodPressureSystolic, &visit.BloodPressureDiastolic,
		&visit.WeightKg, &visit.HeartRateBpm, &visit.TemperatureCelsius,
		&visit.FundalHeightCm, &visit.FetalHeartRateBpm, &visit.GestationalAgeWeeks,
		&visit.Medications, &visit.LabResults,
		&visit.NextAppointmentAt, &visit.NextAppointmentNotes,
		&visit.RecordedBy, &visit.ProviderUserID,
		&visit.CreatedAt, &visit.UpdatedAt,
	)
}

// CreateDoctorVisit inserts a new visit record.
func (db *DB) CreateDoctorVisit(ctx context.Context, visit *DoctorVisit) error {
	if len(visit.Medications) == 0 {
		visit.Medications = []byte("[]")
	}
	if len(visit.LabResults) == 0 {
		visit.LabResults = []byte("[]")
	}
	if visit.RecordedBy == "" {
		visit.RecordedBy = "user"
	}

	query := `
		INSERT INTO doctor_visits (
			user_id, visit_date, visit_type, provider_name, facility_name,
			chief_complaint, clinical_notes, diagnosis, treatment_plan, follow_up_instructions,
			blood_pressure_systolic, blood_pressure_diastolic, weight_kg, heart_rate_bpm,
			temperature_celsius, fundal_height_cm, fetal_heart_rate_bpm, gestational_age_weeks,
			medications, lab_results, next_appointment_at, next_appointment_notes,
			recorded_by, provider_user_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24
		)
		RETURNING id, created_at, updated_at
	`

	return db.QueryRowContext(ctx, query,
		visit.UserID, visit.VisitDate, visit.VisitType,
		visit.ProviderName, visit.FacilityName,
		visit.ChiefComplaint, visit.ClinicalNotes, visit.Diagnosis,
		visit.TreatmentPlan, visit.FollowUpInstructions,
		visit.BloodPressureSystolic, visit.BloodPressureDiastolic,
		visit.WeightKg, visit.HeartRateBpm, visit.TemperatureCelsius,
		visit.FundalHeightCm, visit.FetalHeartRateBpm, visit.GestationalAgeWeeks,
		visit.Medications, visit.LabResults,
		visit.NextAppointmentAt, visit.NextAppointmentNotes,
		visit.RecordedBy, visit.ProviderUserID,
	).Scan(&visit.ID, &visit.CreatedAt, &visit.UpdatedAt)
}

// GetUserDoctorVisits returns all visits for a patient, newest first.
func (db *DB) GetUserDoctorVisits(ctx context.Context, userID string) ([]DoctorVisit, error) {
	query := `
		SELECT ` + doctorVisitSelectColumns + `
		FROM doctor_visits
		WHERE user_id = $1
		ORDER BY visit_date DESC, created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get doctor visits: %w", err)
	}
	defer rows.Close()

	visits := make([]DoctorVisit, 0)
	for rows.Next() {
		var visit DoctorVisit
		if err := scanDoctorVisit(rows, &visit); err != nil {
			return nil, fmt.Errorf("failed to scan doctor visit: %w", err)
		}
		visits = append(visits, visit)
	}

	return visits, nil
}

// GetDoctorVisitByID returns a single visit by ID.
func (db *DB) GetDoctorVisitByID(ctx context.Context, id string) (*DoctorVisit, error) {
	query := `
		SELECT ` + doctorVisitSelectColumns + `
		FROM doctor_visits
		WHERE id = $1
	`

	visit := &DoctorVisit{}
	err := scanDoctorVisit(db.QueryRowContext(ctx, query, id), visit)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get doctor visit: %w", err)
	}

	return visit, nil
}

// UpdateDoctorVisit updates an existing visit record.
func (db *DB) UpdateDoctorVisit(ctx context.Context, visit *DoctorVisit) error {
	if len(visit.Medications) == 0 {
		visit.Medications = []byte("[]")
	}
	if len(visit.LabResults) == 0 {
		visit.LabResults = []byte("[]")
	}

	query := `
		UPDATE doctor_visits SET
			visit_date = $1, visit_type = $2, provider_name = $3, facility_name = $4,
			chief_complaint = $5, clinical_notes = $6, diagnosis = $7, treatment_plan = $8,
			follow_up_instructions = $9, blood_pressure_systolic = $10, blood_pressure_diastolic = $11,
			weight_kg = $12, heart_rate_bpm = $13, temperature_celsius = $14,
			fundal_height_cm = $15, fetal_heart_rate_bpm = $16, gestational_age_weeks = $17,
			medications = $18, lab_results = $19, next_appointment_at = $20, next_appointment_notes = $21,
			recorded_by = $22, provider_user_id = $23, updated_at = CURRENT_TIMESTAMP
		WHERE id = $24
		RETURNING updated_at
	`

	err := db.QueryRowContext(ctx, query,
		visit.VisitDate, visit.VisitType, visit.ProviderName, visit.FacilityName,
		visit.ChiefComplaint, visit.ClinicalNotes, visit.Diagnosis, visit.TreatmentPlan,
		visit.FollowUpInstructions, visit.BloodPressureSystolic, visit.BloodPressureDiastolic,
		visit.WeightKg, visit.HeartRateBpm, visit.TemperatureCelsius,
		visit.FundalHeightCm, visit.FetalHeartRateBpm, visit.GestationalAgeWeeks,
		visit.Medications, visit.LabResults, visit.NextAppointmentAt, visit.NextAppointmentNotes,
		visit.RecordedBy, visit.ProviderUserID, visit.ID,
	).Scan(&visit.UpdatedAt)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update doctor visit: %w", err)
	}

	return nil
}

// DeleteDoctorVisit removes a visit record.
func (db *DB) DeleteDoctorVisit(ctx context.Context, id string) error {
	result, err := db.ExecContext(ctx, `DELETE FROM doctor_visits WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete doctor visit: %w", err)
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
