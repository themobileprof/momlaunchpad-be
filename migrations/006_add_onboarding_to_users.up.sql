ALTER TABLE users ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;

-- Existing users with pregnancy context already on file skip the walkthrough.
UPDATE users
SET onboarding_completed_at = COALESCE(updated_at, created_at, CURRENT_TIMESTAMP)
WHERE onboarding_completed_at IS NULL
  AND (
    expected_delivery_date IS NOT NULL
    OR EXISTS (
      SELECT 1 FROM user_facts
      WHERE user_facts.user_id = users.id
        AND user_facts.key = 'pregnancy_week'
    )
  );
