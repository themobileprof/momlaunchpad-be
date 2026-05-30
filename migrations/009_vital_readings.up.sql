CREATE TABLE IF NOT EXISTS vital_readings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    blood_pressure_systolic INTEGER,
    blood_pressure_diastolic INTEGER,
    weight_kg DECIMAL(6, 2),
    heart_rate_bpm INTEGER,
    temperature_celsius DECIMAL(4, 1),
    fundal_height_cm DECIMAL(5, 1),
    fetal_heart_rate_bpm INTEGER,
    gestational_age_weeks INTEGER,
    notes TEXT,
    source VARCHAR(20) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vital_readings_user_id ON vital_readings(user_id);
CREATE INDEX IF NOT EXISTS idx_vital_readings_recorded_at ON vital_readings(user_id, recorded_at DESC);
