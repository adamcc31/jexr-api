-- Rollback: Remove application indexes
-- These are safe to drop as they don't affect data

DROP INDEX IF EXISTS idx_users_role_created;
DROP INDEX IF EXISTS idx_users_role;
-- Keep idx_users_email as it's implied by UNIQUE constraint
