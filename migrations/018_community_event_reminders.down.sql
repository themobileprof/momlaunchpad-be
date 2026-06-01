DROP INDEX IF EXISTS idx_reminders_user_community_event;
ALTER TABLE reminders DROP COLUMN IF EXISTS community_event_id;
