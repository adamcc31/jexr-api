-- Migration: Add job detail fields
-- These columns use TEXT type for flexibility (no ENUM types)

ALTER TABLE jobs
ADD COLUMN employment_type TEXT,
ADD COLUMN job_type TEXT,
ADD COLUMN experience_level TEXT,
ADD COLUMN qualifications TEXT;

-- Create index for filtering active jobs
CREATE INDEX IF NOT EXISTS idx_jobs_company_status ON jobs(company_status);
