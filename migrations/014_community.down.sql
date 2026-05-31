DROP TABLE IF EXISTS community_notifications;
DROP TABLE IF EXISTS community_hidden_posts;
DROP TABLE IF EXISTS community_blocks;
DROP TABLE IF EXISTS community_reports;
DROP TABLE IF EXISTS community_user_badges;
DROP TABLE IF EXISTS community_follows;
DROP TABLE IF EXISTS community_event_interests;
DROP TABLE IF EXISTS community_events;
DROP TABLE IF EXISTS community_reply_likes;
DROP TABLE IF EXISTS community_replies;
DROP TABLE IF EXISTS community_post_likes;
DROP TABLE IF EXISTS community_posts;
DROP TABLE IF EXISTS community_user_interests;

ALTER TABLE users
    DROP COLUMN IF EXISTS profile_photo_url,
    DROP COLUMN IF EXISTS country,
    DROP COLUMN IF EXISTS state_province,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS community_onboarding_completed_at;
