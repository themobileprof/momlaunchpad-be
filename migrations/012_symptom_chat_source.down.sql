DROP INDEX IF EXISTS idx_symptoms_conversation_id;
ALTER TABLE symptoms
    DROP COLUMN IF EXISTS message_id,
    DROP COLUMN IF EXISTS conversation_id;
