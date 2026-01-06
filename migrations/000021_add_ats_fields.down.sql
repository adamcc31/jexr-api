-- ============================================================================
-- Migration Rollback: Remove ATS fields
-- ============================================================================

-- Drop indexes
DROP INDEX IF EXISTS idx_av_gender;
DROP INDEX IF EXISTS idx_cp_total_experience;
DROP INDEX IF EXISTS idx_av_status_verified_at;
DROP INDEX IF EXISTS idx_av_expected_salary;
DROP INDEX IF EXISTS idx_av_available_start_date;

-- Remove columns
ALTER TABLE account_verifications DROP COLUMN IF EXISTS gender;
ALTER TABLE candidate_profiles DROP COLUMN IF EXISTS total_experience_months;
