-- Rollback OAuth providers support

DROP TABLE IF EXISTS oauth_providers;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'users' AND column_name = 'email') THEN
        DROP INDEX IF EXISTS idx_users_email;
        ALTER TABLE users DROP COLUMN email;
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'users' AND column_name = 'auth_provider') THEN
        ALTER TABLE users DROP COLUMN auth_provider;
    END IF;
END $$;

-- Restore NOT NULL constraint on password_hash if needed
DO $$ 
BEGIN
    ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
EXCEPTION
    WHEN OTHERS THEN NULL;
END $$;
