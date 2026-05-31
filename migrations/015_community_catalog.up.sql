-- Configurable community reference data (managed via admin UI)

CREATE TABLE IF NOT EXISTS community_interest_groups (
    key VARCHAR(64) PRIMARY KEY,
    label VARCHAR(120) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS community_interests (
    key VARCHAR(64) PRIMARY KEY,
    group_key VARCHAR(64) NOT NULL REFERENCES community_interest_groups(key) ON DELETE RESTRICT,
    label VARCHAR(120) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_community_interests_group
    ON community_interests(group_key, sort_order);

CREATE TABLE IF NOT EXISTS community_badge_types (
    key VARCHAR(64) PRIMARY KEY,
    label VARCHAR(120) NOT NULL,
    description TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS community_event_types (
    key VARCHAR(64) PRIMARY KEY,
    label VARCHAR(120) NOT NULL,
    description TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS community_countries (
    code VARCHAR(8) PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS community_regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code VARCHAR(8) NOT NULL REFERENCES community_countries(code) ON DELETE CASCADE,
    code VARCHAR(32) NOT NULL,
    name VARCHAR(120) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (country_code, code)
);

CREATE INDEX IF NOT EXISTS idx_community_regions_country
    ON community_regions(country_code, sort_order);

-- Replace hard-coded badge CHECK with catalog FK
ALTER TABLE community_user_badges
    DROP CONSTRAINT IF EXISTS community_user_badges_badge_type_check;

ALTER TABLE community_user_interests
    DROP CONSTRAINT IF EXISTS community_user_interests_interest_key_fkey;

ALTER TABLE community_events
    ADD COLUMN IF NOT EXISTS event_type VARCHAR(64);

-- Seed interest groups
INSERT INTO community_interest_groups (key, label, sort_order) VALUES
    ('pregnancy', 'Pregnancy', 1),
    ('health', 'Health', 2),
    ('baby', 'Baby', 3),
    ('parenthood', 'Parenthood', 4),
    ('support', 'Support', 5),
    ('local', 'Local', 6),
    ('community', 'Community', 7)
ON CONFLICT (key) DO NOTHING;

INSERT INTO community_interests (key, group_key, label, sort_order) VALUES
    ('first_trimester', 'pregnancy', 'First Trimester', 1),
    ('second_trimester', 'pregnancy', 'Second Trimester', 2),
    ('third_trimester', 'pregnancy', 'Third Trimester', 3),
    ('pregnancy_health', 'health', 'Pregnancy Health', 1),
    ('mental_health', 'health', 'Mental Health', 2),
    ('nutrition', 'health', 'Nutrition', 3),
    ('fitness', 'health', 'Fitness', 4),
    ('newborn_care', 'baby', 'Newborn Care', 1),
    ('breastfeeding', 'baby', 'Breastfeeding', 2),
    ('baby_sleep', 'baby', 'Baby Sleep', 3),
    ('baby_health', 'baby', 'Baby Health', 4),
    ('first_time_moms', 'parenthood', 'First-Time Moms', 1),
    ('experienced_moms', 'parenthood', 'Experienced Moms', 2),
    ('dads_partners', 'parenthood', 'Dads & Partners', 3),
    ('single_parents', 'parenthood', 'Single Parents', 4),
    ('ask_midwife', 'support', 'Ask a Midwife', 1),
    ('ask_doctor', 'support', 'Ask a Doctor', 2),
    ('emotional_support', 'support', 'Emotional Support', 3),
    ('local_recommendations', 'local', 'Local Recommendations', 1),
    ('local_services', 'local', 'Local Services', 2),
    ('events_meetups', 'local', 'Events & Meetups', 3),
    ('introductions', 'community', 'Introductions', 1),
    ('success_stories', 'community', 'Success Stories', 2)
ON CONFLICT (key) DO NOTHING;

INSERT INTO community_badge_types (key, label, description, sort_order) VALUES
    ('midwife', 'Midwife', 'Verified midwife or doula', 1),
    ('doctor', 'Doctor', 'Verified medical doctor', 2),
    ('pediatrician', 'Pediatrician', 'Verified pediatric specialist', 3),
    ('lactation_consultant', 'Lactation Consultant', 'Verified lactation consultant', 4),
    ('community_moderator', 'Community Moderator', 'Official community moderator', 5)
ON CONFLICT (key) DO NOTHING;

INSERT INTO community_event_types (key, label, description, sort_order) VALUES
    ('antenatal_class', 'Antenatal class', 'Prenatal education sessions', 1),
    ('breastfeeding_workshop', 'Breastfeeding workshop', 'Breastfeeding support workshops', 2),
    ('hospital_seminar', 'Hospital seminar', 'Hospital-hosted seminars', 3),
    ('mom_meetup', 'Mom meetup', 'Informal parent meetups', 4),
    ('ngo_program', 'NGO program', 'Community or NGO-led programs', 5),
    ('other', 'Other', 'Other local events', 99)
ON CONFLICT (key) DO NOTHING;

INSERT INTO community_countries (code, name, sort_order) VALUES
    ('NG', 'Nigeria', 1),
    ('GH', 'Ghana', 2),
    ('KE', 'Kenya', 3),
    ('ZA', 'South Africa', 4),
    ('GB', 'United Kingdom', 5),
    ('US', 'United States', 6),
    ('CA', 'Canada', 7)
ON CONFLICT (code) DO NOTHING;

INSERT INTO community_regions (country_code, code, name, sort_order) VALUES
    ('NG', 'LA', 'Lagos', 1),
    ('NG', 'AB', 'Abuja', 2),
    ('NG', 'RV', 'Rivers', 3),
    ('US', 'CA', 'California', 1),
    ('US', 'NY', 'New York', 2),
    ('US', 'TX', 'Texas', 3),
    ('GB', 'ENG', 'England', 1),
    ('GB', 'SCT', 'Scotland', 2),
    ('CA', 'ON', 'Ontario', 1),
    ('CA', 'BC', 'British Columbia', 2)
ON CONFLICT (country_code, code) DO NOTHING;

-- FK constraints after seed data
ALTER TABLE community_user_badges
    ADD CONSTRAINT fk_community_user_badges_type
    FOREIGN KEY (badge_type) REFERENCES community_badge_types(key);

ALTER TABLE community_user_interests
    ADD CONSTRAINT fk_community_user_interests_key
    FOREIGN KEY (interest_key) REFERENCES community_interests(key);

ALTER TABLE community_events
    ADD CONSTRAINT fk_community_events_type
    FOREIGN KEY (event_type) REFERENCES community_event_types(key);
