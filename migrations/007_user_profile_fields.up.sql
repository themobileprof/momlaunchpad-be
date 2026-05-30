ALTER TABLE users ADD COLUMN IF NOT EXISTS pregnancy_week SMALLINT
  CHECK (pregnancy_week IS NULL OR (pregnancy_week >= 1 AND pregnancy_week <= 42));
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_first_pregnancy BOOLEAN;
ALTER TABLE users ADD COLUMN IF NOT EXISTS primary_concern TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS diet_preference VARCHAR(50);

-- Backfill profile columns from existing user_facts.
UPDATE users u
SET pregnancy_week = CAST(f.value AS SMALLINT)
FROM user_facts f
WHERE f.user_id = u.id
  AND f.key = 'pregnancy_week'
  AND u.pregnancy_week IS NULL
  AND f.value ~ '^[0-9]+$';

UPDATE users u
SET is_first_pregnancy = (LOWER(f.value) = 'yes')
FROM user_facts f
WHERE f.user_id = u.id
  AND f.key = 'is_first_pregnancy'
  AND u.is_first_pregnancy IS NULL;

UPDATE users u
SET primary_concern = f.value
FROM user_facts f
WHERE f.user_id = u.id
  AND f.key = 'primary_concern'
  AND (u.primary_concern IS NULL OR u.primary_concern = '');

UPDATE users u
SET diet_preference = f.value
FROM user_facts f
WHERE f.user_id = u.id
  AND f.key = 'diet'
  AND (u.diet_preference IS NULL OR u.diet_preference = '');

-- Approximate pregnancy start from week when missing.
UPDATE users
SET pregnancy_start_date = (CURRENT_DATE - (pregnancy_week * 7))
WHERE pregnancy_week IS NOT NULL
  AND pregnancy_start_date IS NULL;

-- Derive week from EDD when still missing.
UPDATE users
SET pregnancy_week = GREATEST(1, LEAST(42, 40 - ((expected_delivery_date - CURRENT_DATE) / 7)))
WHERE pregnancy_week IS NULL
  AND expected_delivery_date IS NOT NULL;
