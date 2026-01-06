BEGIN;

CREATE TABLE IF NOT EXISTS plans (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS features (
    id SERIAL PRIMARY KEY,
    feature_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS plan_features (
    plan_id INTEGER NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_id INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    PRIMARY KEY (plan_id, feature_id)
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id INTEGER NOT NULL REFERENCES plans(id),
    status TEXT NOT NULL DEFAULT 'active',
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ,
    CHECK (status IN ('active','canceled','expired'))
);

INSERT INTO plans (code, name, description)
VALUES ('free', 'Free', 'Default free plan')
ON CONFLICT (code) DO NOTHING;

INSERT INTO features (feature_key, name, description)
VALUES ('chat', 'Chat Access', 'Allows chat usage')
ON CONFLICT (feature_key) DO NOTHING;

INSERT INTO plan_features (plan_id, feature_id)
SELECT p.id, f.id FROM plans p, features f
WHERE p.code = 'free' AND f.feature_key = 'chat'
ON CONFLICT DO NOTHING;

INSERT INTO subscriptions (user_id, plan_id, status)
SELECT u.id, p.id, 'active'
FROM users u
JOIN plans p ON p.code = 'free'
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions s WHERE s.user_id = u.id AND s.status = 'active'
);

CREATE OR REPLACE FUNCTION assign_free_subscription()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO subscriptions (user_id, plan_id, status)
    SELECT NEW.id, p.id, 'active' FROM plans p WHERE p.code = 'free';
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_assign_free_subscription ON users;
CREATE TRIGGER trg_assign_free_subscription
AFTER INSERT ON users
FOR EACH ROW EXECUTE PROCEDURE assign_free_subscription();

COMMIT;
