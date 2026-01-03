-- Add EDD and savings goal to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS expected_delivery_date DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS savings_goal DECIMAL(10,2) DEFAULT 0.00;
