-- Migration: Add conversations table
BEGIN;

-- Create conversations table
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    is_starred BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at);

-- TRUNCATE messages to delete existing test data as requested
TRUNCATE TABLE messages CASCADE;

-- Update messages table to link to conversations (NOT NULL now possible)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);

COMMIT;
