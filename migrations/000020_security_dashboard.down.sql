-- Migration: 000020_security_dashboard.down.sql
-- Rollback for Security Dashboard Tables

-- Drop triggers
DROP TRIGGER IF EXISTS security_users_updated_at ON security_users;
DROP FUNCTION IF EXISTS update_security_user_timestamp();

-- Drop RLS policies
DROP POLICY IF EXISTS "Service role full access" ON export_requests;
DROP POLICY IF EXISTS "Service role full access" ON hash_anchors;
DROP POLICY IF EXISTS "Service role full access" ON allowed_ip_ranges;
DROP POLICY IF EXISTS "Service role full access" ON break_glass_sessions;
DROP POLICY IF EXISTS "Service role full access" ON security_sessions;
DROP POLICY IF EXISTS "Service role full access" ON security_users;

-- Drop indexes
DROP INDEX IF EXISTS idx_export_requests_status;
DROP INDEX IF EXISTS idx_break_glass_active;
DROP INDEX IF EXISTS idx_security_sessions_token;
DROP INDEX IF EXISTS idx_security_sessions_user;
DROP INDEX IF EXISTS idx_security_events_subject_ip;
DROP INDEX IF EXISTS idx_security_events_severity;

-- Remove columns from security_events
ALTER TABLE security_events DROP COLUMN IF EXISTS row_hash;
ALTER TABLE security_events DROP COLUMN IF EXISTS previous_hash;
ALTER TABLE security_events DROP COLUMN IF EXISTS severity;

-- Drop tables in correct order (respecting foreign keys)
DROP TABLE IF EXISTS export_requests;
DROP TABLE IF EXISTS hash_anchors;
DROP TABLE IF EXISTS break_glass_sessions;
DROP TABLE IF EXISTS security_sessions;
DROP TABLE IF EXISTS allowed_ip_ranges;
DROP TABLE IF EXISTS security_users;

-- Drop enums
DROP TYPE IF EXISTS security_severity;
DROP TYPE IF EXISTS security_role;
