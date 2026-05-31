ALTER TABLE community_events DROP CONSTRAINT IF EXISTS fk_community_events_type;
ALTER TABLE community_user_interests DROP CONSTRAINT IF EXISTS fk_community_user_interests_key;
ALTER TABLE community_user_badges DROP CONSTRAINT IF EXISTS fk_community_user_badges_type;

ALTER TABLE community_events DROP COLUMN IF EXISTS event_type;

ALTER TABLE community_user_badges
    ADD CONSTRAINT community_user_badges_badge_type_check
    CHECK (badge_type IN (
        'midwife', 'doctor', 'pediatrician',
        'lactation_consultant', 'community_moderator'
    ));

DROP TABLE IF EXISTS community_regions;
DROP TABLE IF EXISTS community_countries;
DROP TABLE IF EXISTS community_event_types;
DROP TABLE IF EXISTS community_badge_types;
DROP TABLE IF EXISTS community_interests;
DROP TABLE IF EXISTS community_interest_groups;
