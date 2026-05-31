ALTER TABLE users ADD COLUMN IF NOT EXISTS journey_stage VARCHAR(20)
  CHECK (journey_stage IS NULL OR journey_stage IN ('ttc', 'pregnant', 'postpartum', 'miscarriage'));

ALTER TABLE users ADD COLUMN IF NOT EXISTS journey_stage_since DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS baby_birth_date DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS loss_date DATE;

-- Existing users with pregnancy data are treated as currently pregnant.
UPDATE users
SET journey_stage = 'pregnant',
    journey_stage_since = COALESCE(journey_stage_since, CURRENT_DATE)
WHERE journey_stage IS NULL
  AND (pregnancy_week IS NOT NULL OR expected_delivery_date IS NOT NULL);
