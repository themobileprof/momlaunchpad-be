CREATE TABLE IF NOT EXISTS doctor_visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    visit_date TIMESTAMP NOT NULL,
    visit_type VARCHAR(100) NOT NULL,
    provider_name VARCHAR(255),
    facility_name VARCHAR(255),
    chief_complaint TEXT,
    clinical_notes TEXT,
    diagnosis TEXT,
    treatment_plan TEXT,
    follow_up_instructions TEXT,
    blood_pressure_systolic INTEGER,
    blood_pressure_diastolic INTEGER,
    weight_kg DECIMAL(6, 2),
    heart_rate_bpm INTEGER,
    temperature_celsius DECIMAL(4, 1),
    fundal_height_cm DECIMAL(5, 1),
    fetal_heart_rate_bpm INTEGER,
    gestational_age_weeks INTEGER,
    medications JSONB NOT NULL DEFAULT '[]'::jsonb,
    lab_results JSONB NOT NULL DEFAULT '[]'::jsonb,
    next_appointment_at TIMESTAMP,
    next_appointment_notes TEXT,
    recorded_by VARCHAR(20) NOT NULL DEFAULT 'user',
    provider_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doctor_visits_user_id ON doctor_visits(user_id);
CREATE INDEX IF NOT EXISTS idx_doctor_visits_visit_date ON doctor_visits(user_id, visit_date DESC);
CREATE INDEX IF NOT EXISTS idx_doctor_visits_provider_user ON doctor_visits(provider_user_id)
    WHERE provider_user_id IS NOT NULL;
