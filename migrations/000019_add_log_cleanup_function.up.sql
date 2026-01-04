-- Migration: 000019_add_log_cleanup_function.up.sql
-- Safe batch deletion of old security events
-- WARNING: Do NOT use unbounded DELETE in production PostgreSQL

-- Function for safe batch deletion of old security events
-- Uses LIMIT + loop pattern to avoid transaction log bloat and table locks
CREATE OR REPLACE FUNCTION cleanup_old_security_events(
    retention_days INTEGER DEFAULT 90,
    batch_size INTEGER DEFAULT 1000,
    max_iterations INTEGER DEFAULT 1000
)
RETURNS INTEGER AS $$
DECLARE
    total_deleted INTEGER := 0;
    rows_deleted INTEGER;
    iteration INTEGER := 0;
    cutoff_date TIMESTAMPTZ;
BEGIN
    -- Calculate cutoff date once
    cutoff_date := NOW() - (retention_days || ' days')::INTERVAL;
    
    -- Log start
    RAISE NOTICE 'Starting cleanup of security_events older than % (% days)', cutoff_date, retention_days;
    
    LOOP
        -- Delete a batch of old records
        DELETE FROM security_events
        WHERE id IN (
            SELECT id FROM security_events
            WHERE created_at < cutoff_date
            ORDER BY created_at ASC
            LIMIT batch_size
        );
        
        GET DIAGNOSTICS rows_deleted = ROW_COUNT;
        total_deleted := total_deleted + rows_deleted;
        iteration := iteration + 1;
        
        -- Exit if no more rows to delete or max iterations reached
        EXIT WHEN rows_deleted = 0;
        EXIT WHEN iteration >= max_iterations;
        
        -- Yield to other connections - small sleep between batches
        PERFORM pg_sleep(0.05);
        
        -- Log progress every 10 iterations
        IF iteration % 10 = 0 THEN
            RAISE NOTICE 'Cleanup progress: % rows deleted in % iterations', total_deleted, iteration;
        END IF;
    END LOOP;
    
    -- Log completion
    RAISE NOTICE 'Cleanup complete: % total rows deleted in % iterations', total_deleted, iteration;
    
    RETURN total_deleted;
END;
$$ LANGUAGE plpgsql;

-- Add comment
COMMENT ON FUNCTION cleanup_old_security_events IS 
'Safely deletes old security_events in batches to avoid transaction log bloat. 
Call with SELECT cleanup_old_security_events(90, 1000, 5000) for 90-day retention.
Schedule via pg_cron or application cron.';

-- Create index to speed up cleanup queries (if not exists)
CREATE INDEX IF NOT EXISTS idx_security_events_created_at 
    ON security_events(created_at);

-- Optional: Schedule with pg_cron (requires pg_cron extension)
-- Uncomment if pg_cron is available:
-- SELECT cron.schedule(
--     'cleanup_security_events_daily',
--     '0 3 * * *',  -- Daily at 3 AM UTC
--     $$SELECT cleanup_old_security_events(90, 1000, 5000)$$
-- );
