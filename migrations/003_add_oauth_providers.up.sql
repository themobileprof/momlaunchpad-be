-- Add OAuth providers table for multi-provider support
-- Email is the canonical identifier across all providers (Google, Apple, etc.)

CREATE TABLE IF NOT EXISTS oauth_providers (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google', 'apple', etc.
    provider_user_id VARCHAR(255) NOT NULL, -- OAuth provider's unique ID for this user
    email VARCHAR(255) NOT NULL, -- Email from OAuth provider
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure one provider account can only be linked once
    UNIQUE(provider, provider_user_id)
);

-- Index for fast email lookups across providers
CREATE INDEX IF NOT EXISTS idx_oauth_email ON oauth_providers(email);

-- Add email to users table if not exists (for email-based linking)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'users' AND column_name = 'email') THEN
        ALTER TABLE users ADD COLUMN email VARCHAR(255) UNIQUE;
        CREATE INDEX idx_users_email ON users(email);
    END IF;
END $$;

-- Add provider type to users for tracking primary auth method
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'users' AND column_name = 'auth_provider') THEN
        ALTER TABLE users ADD COLUMN auth_provider VARCHAR(50) DEFAULT 'local';
    END IF;
END $$;

-- Make password nullable for OAuth-only users
DO $$ 
BEGIN
    ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
EXCEPTION
    WHEN OTHERS THEN NULL;
END $$;
