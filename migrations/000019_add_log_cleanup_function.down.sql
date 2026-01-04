-- Migration: 000019_add_log_cleanup_function.down.sql
-- Rollback for log cleanup function

-- Remove scheduled job if pg_cron was used
-- SELECT cron.unschedule('cleanup_security_events_daily');

-- Drop the cleanup function
DROP FUNCTION IF EXISTS cleanup_old_security_events(INTEGER, INTEGER, INTEGER);

-- Note: We don't drop the index as it may be useful for other queries
