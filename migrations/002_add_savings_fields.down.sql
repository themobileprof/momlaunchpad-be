-- Remove savings fields from users table
ALTER TABLE users DROP COLUMN IF EXISTS expected_delivery_date;
ALTER TABLE users DROP COLUMN IF EXISTS savings_goal;
