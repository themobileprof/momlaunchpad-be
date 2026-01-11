-- Symptoms tracking table
CREATE TABLE IF NOT EXISTS symptoms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symptom_type VARCHAR(100) NOT NULL, -- e.g., 'swelling', 'nausea', 'headache'
    description TEXT NOT NULL, -- User's description
    severity VARCHAR(20), -- 'mild', 'moderate', 'severe'
    frequency VARCHAR(50), -- 'once', 'occasionally', 'daily', 'constant'
    onset_time VARCHAR(100), -- When it started: 'yesterday', '3 days ago', 'this morning'
    associated_symptoms TEXT[], -- Related symptoms mentioned
    is_resolved BOOLEAN DEFAULT FALSE,
    reported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_symptoms_user_id ON symptoms(user_id);
CREATE INDEX IF NOT EXISTS idx_symptoms_reported_at ON symptoms(reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_symptoms_type ON symptoms(symptom_type);
CREATE INDEX IF NOT EXISTS idx_symptoms_user_reported ON symptoms(user_id, reported_at DESC);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_symptoms_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER symptoms_updated_at_trigger
    BEFORE UPDATE ON symptoms
    FOR EACH ROW
    EXECUTE FUNCTION update_symptoms_updated_at();
