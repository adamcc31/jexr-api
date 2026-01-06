-- ============================================================================
-- Migration: Add ATS (Applicant Tracking System) fields
-- Purpose: Add gender field and pre-computed experience column for ATS filtering
-- ============================================================================

-- Add gender field to account_verifications
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS gender VARCHAR(10);

-- Add computed total experience to candidate_profiles
ALTER TABLE candidate_profiles
ADD COLUMN IF NOT EXISTS total_experience_months INT DEFAULT 0;

-- ============================================================================
-- Performance Indexes for ATS Filtering
-- ============================================================================

-- Index for gender filtering
CREATE INDEX IF NOT EXISTS idx_av_gender ON account_verifications(gender);

-- Index for total experience filtering
CREATE INDEX IF NOT EXISTS idx_cp_total_experience ON candidate_profiles(total_experience_months);

-- Composite index for common ATS queries (status + verified_at for sorting)
CREATE INDEX IF NOT EXISTS idx_av_status_verified_at ON account_verifications(status, verified_at DESC);

-- Index for salary range filtering
CREATE INDEX IF NOT EXISTS idx_av_expected_salary ON account_verifications(expected_salary);

-- Index for available start date filtering
CREATE INDEX IF NOT EXISTS idx_av_available_start_date ON account_verifications(available_start_date);

-- ============================================================================
-- Backfill total_experience_months from existing work_experiences
-- ============================================================================

-- Update total_experience_months for all candidates based on their work_experiences
UPDATE candidate_profiles cp
SET total_experience_months = COALESCE(
    (
        SELECT SUM(
            EXTRACT(YEAR FROM AGE(COALESCE(we.end_date, CURRENT_DATE), we.start_date)) * 12 +
            EXTRACT(MONTH FROM AGE(COALESCE(we.end_date, CURRENT_DATE), we.start_date))
        )::INT
        FROM work_experiences we
        WHERE we.user_id = cp.user_id
    ), 0
);

-- ============================================================================
-- Add comment for documentation
-- ============================================================================
COMMENT ON COLUMN account_verifications.gender IS 'Candidate gender: MALE, FEMALE, or NULL';
COMMENT ON COLUMN candidate_profiles.total_experience_months IS 'Pre-computed total work experience in months. Updated when work_experiences change.';
