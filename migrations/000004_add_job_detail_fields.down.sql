-- Rollback: Remove job detail fields

DROP INDEX IF EXISTS idx_jobs_company_status;

ALTER TABLE jobs
DROP COLUMN IF EXISTS employment_type,
DROP COLUMN IF EXISTS job_type,
DROP COLUMN IF EXISTS experience_level,
DROP COLUMN IF EXISTS qualifications;
