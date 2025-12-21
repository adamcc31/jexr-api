-- ============================================================================
-- Rollback Migration: 000012_add_hr_candidate_data
-- Purpose: Safely remove HR-grade candidate fields from account_verifications
-- ============================================================================

-- Drop indexes first
DROP INDEX IF EXISTS idx_av_main_job_fields;
DROP INDEX IF EXISTS idx_av_preferred_locations;
DROP INDEX IF EXISTS idx_av_preferred_industries;
DROP INDEX IF EXISTS idx_av_birth_date;
DROP INDEX IF EXISTS idx_av_marital_status;
DROP INDEX IF EXISTS idx_av_expected_salary;
DROP INDEX IF EXISTS idx_av_available_start_date;
DROP INDEX IF EXISTS idx_av_japanese_speaking_level;

-- Drop columns (only the ones added in this migration)
ALTER TABLE account_verifications
DROP COLUMN IF EXISTS birth_date,
DROP COLUMN IF EXISTS domicile_city,
DROP COLUMN IF EXISTS marital_status,
DROP COLUMN IF EXISTS children_count,
DROP COLUMN IF EXISTS main_job_fields,
DROP COLUMN IF EXISTS golden_skill,
DROP COLUMN IF EXISTS japanese_speaking_level,
DROP COLUMN IF EXISTS expected_salary,
DROP COLUMN IF EXISTS japan_return_date,
DROP COLUMN IF EXISTS available_start_date,
DROP COLUMN IF EXISTS preferred_locations,
DROP COLUMN IF EXISTS preferred_industries,
DROP COLUMN IF EXISTS supporting_certificates_url;
