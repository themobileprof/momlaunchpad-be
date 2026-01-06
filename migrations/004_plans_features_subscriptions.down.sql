BEGIN;

DROP TRIGGER IF EXISTS trg_assign_free_subscription ON users;
DROP FUNCTION IF EXISTS assign_free_subscription();

DELETE FROM subscriptions WHERE plan_id IN (SELECT id FROM plans WHERE code = 'free');
DELETE FROM plan_features WHERE plan_id IN (SELECT id FROM plans WHERE code = 'free');
DELETE FROM features WHERE feature_key = 'chat';
DELETE FROM plans WHERE code = 'free';

DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plan_features;
DROP TABLE IF EXISTS features;
DROP TABLE IF EXISTS plans;

COMMIT;
