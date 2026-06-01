ALTER TABLE reminders
    ADD COLUMN IF NOT EXISTS community_event_id UUID REFERENCES community_events(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_reminders_user_community_event
    ON reminders(user_id, community_event_id)
    WHERE community_event_id IS NOT NULL;
