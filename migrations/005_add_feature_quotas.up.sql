BEGIN;

-- Add quota columns to plan_features
ALTER TABLE plan_features
ADD COLUMN quota_limit INTEGER,
ADD COLUMN quota_period TEXT DEFAULT 'monthly' CHECK (quota_period IN ('daily', 'weekly', 'monthly', 'unlimited'));

-- Create feature_usage table to track consumption
CREATE TABLE IF NOT EXISTS feature_usage (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature_key TEXT NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, feature_key, period_start)
);

CREATE INDEX idx_feature_usage_user_feature ON feature_usage(user_id, feature_key);
CREATE INDEX idx_feature_usage_period ON feature_usage(period_end);

-- Update existing plan_features with default unlimited quota
UPDATE plan_features
SET quota_limit = NULL, quota_period = 'unlimited'
WHERE quota_limit IS NULL;

-- Add chat quota for free plan (example: 100 messages per month)
UPDATE plan_features pf
SET quota_limit = 100, quota_period = 'monthly'
FROM plans p
JOIN features f ON f.id = pf.feature_id
WHERE pf.plan_id = p.id
  AND p.code = 'free'
  AND f.feature_key = 'chat';

COMMIT;
