-- Rollback initial schema

DROP TRIGGER IF EXISTS update_reminders_updated_at ON reminders;
DROP TRIGGER IF EXISTS update_user_facts_updated_at ON user_facts;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS savings_entries;
DROP TABLE IF EXISTS languages;
DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS user_facts;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS users;
