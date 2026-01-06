-- MomLaunchpad Complete Schema Migration
-- Combines all migrations into a single optimized file

BEGIN;

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- Nullable for OAuth-only users
    display_name VARCHAR(255),
    preferred_language VARCHAR(10) DEFAULT 'en',
    pregnancy_start_date DATE,
    expected_delivery_date DATE,
    savings_goal DECIMAL(10,2) DEFAULT 0.00,
    is_admin BOOLEAN DEFAULT FALSE,
    auth_provider VARCHAR(50) DEFAULT 'local',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Messages table (chat history)
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- User facts table (long-term memory)
CREATE TABLE IF NOT EXISTS user_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_user_facts_user_id ON user_facts(user_id);

-- Reminders table (calendar)
CREATE TABLE IF NOT EXISTS reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    reminder_time TIMESTAMP NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reminders_user_id ON reminders(user_id);
CREATE INDEX IF NOT EXISTS idx_reminders_time ON reminders(reminder_time);

-- Languages table (multilingual support)
CREATE TABLE IF NOT EXISTS languages (
    code VARCHAR(10) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    native_name VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_experimental BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Savings entries table (optional - manual only)
CREATE TABLE IF NOT EXISTS savings_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    description TEXT,
    entry_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_savings_user_id ON savings_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_savings_date ON savings_entries(entry_date);

-- ============================================================================
-- OAUTH SUPPORT
-- ============================================================================

-- OAuth providers table
CREATE TABLE IF NOT EXISTS oauth_providers (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google', 'apple', etc.
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_email ON oauth_providers(email);

-- ============================================================================
-- SUBSCRIPTION & QUOTA SYSTEM
-- ============================================================================

-- Plans table
CREATE TABLE IF NOT EXISTS plans (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Features table
CREATE TABLE IF NOT EXISTS features (
    id SERIAL PRIMARY KEY,
    feature_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plan-Feature mapping with quotas
CREATE TABLE IF NOT EXISTS plan_features (
    plan_id INTEGER NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_id INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    quota_limit INTEGER, -- NULL = unlimited
    quota_period TEXT DEFAULT 'monthly' CHECK (quota_period IN ('daily', 'weekly', 'monthly', 'unlimited')),
    PRIMARY KEY (plan_id, feature_id)
);

-- User subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id INTEGER NOT NULL REFERENCES plans(id),
    status TEXT NOT NULL DEFAULT 'active',
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ,
    CHECK (status IN ('active','canceled','expired'))
);

-- Feature usage tracking
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

CREATE INDEX IF NOT EXISTS idx_feature_usage_user_feature ON feature_usage(user_id, feature_key);
CREATE INDEX IF NOT EXISTS idx_feature_usage_period ON feature_usage(period_end);

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Default languages
INSERT INTO languages (code, name, native_name, is_enabled) VALUES
    ('en', 'English', 'English', TRUE),
    ('es', 'Spanish', 'Español', TRUE),
    ('fr', 'French', 'Français', TRUE)
ON CONFLICT (code) DO NOTHING;

-- Default plans
INSERT INTO plans (code, name, description) VALUES
    ('free', 'Free', 'Default free plan with limited features'),
    ('premium', 'Premium', 'Full access to all features and higher quotas')
ON CONFLICT (code) DO NOTHING;

-- Default features
INSERT INTO features (feature_key, name, description) VALUES
    ('chat', 'Chat Access', 'AI chat support'),
    ('calendar', 'Calendar', 'Reminders and scheduling'),
    ('savings', 'Savings Tracker', 'Track savings progress'),
    -- pairing with close healthcare providers
    ('healthcare_integration', 'Healthcare Integration', 'Connect with healthcare providers'),
    -- Call a real number for assistance
    ('phone_support', 'Phone Support', 'Access to phone support services')
ON CONFLICT (feature_key) DO NOTHING;

-- Free plan features (chat with quota, others unlimited)
INSERT INTO plan_features (plan_id, feature_id, quota_limit, quota_period)
SELECT p.id, f.id, 
    CASE WHEN f.feature_key = 'chat' THEN 100 ELSE NULL END,
    CASE WHEN f.feature_key = 'chat' THEN 'monthly' ELSE 'unlimited' END
FROM plans p, features f
WHERE p.code = 'free' AND f.feature_key IN ('chat', 'calendar')
ON CONFLICT (plan_id, feature_id) DO NOTHING;

-- Premium plan features (all unlimited)
INSERT INTO plan_features (plan_id, feature_id, quota_limit, quota_period)
SELECT p.id, f.id, NULL, 'unlimited'
FROM plans p, features f
WHERE p.code = 'premium'
ON CONFLICT (plan_id, feature_id) DO NOTHING;

-- ============================================================================
-- TRIGGERS & FUNCTIONS
-- ============================================================================

-- NOTE: updated_at timestamps are managed in application code for:
--   - Explicit control and visibility
--   - Easier testing and debugging
--   - Simpler database schema
-- PostgreSQL doesn't have MySQL's ON UPDATE CURRENT_TIMESTAMP syntax

-- Auto-assign free subscription to new users
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
FOR EACH ROW EXECUTE FUNCTION assign_free_subscription();

COMMIT;
