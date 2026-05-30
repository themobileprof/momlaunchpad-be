CREATE TABLE IF NOT EXISTS user_welcome_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cache_date DATE NOT NULL,
    message TEXT NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'gemini',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, cache_date)
);

CREATE INDEX IF NOT EXISTS idx_user_welcome_messages_lookup
    ON user_welcome_messages(user_id, cache_date DESC);
