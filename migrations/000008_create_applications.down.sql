-- Rollback applications table
DROP INDEX IF EXISTS idx_applications_created_at;
DROP INDEX IF EXISTS idx_applications_status;
DROP INDEX IF EXISTS idx_applications_candidate;
DROP INDEX IF EXISTS idx_applications_job_id;
DROP TABLE IF EXISTS applications;
