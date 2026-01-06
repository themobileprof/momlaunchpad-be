BEGIN;

DROP INDEX IF EXISTS idx_feature_usage_period;
DROP INDEX IF EXISTS idx_feature_usage_user_feature;
DROP TABLE IF EXISTS feature_usage;

ALTER TABLE plan_features
DROP COLUMN IF EXISTS quota_period,
DROP COLUMN IF EXISTS quota_limit;

COMMIT;
