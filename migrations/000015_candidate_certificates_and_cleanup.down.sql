-- ============================================================================
-- Migration: 000015_candidate_certificates_and_cleanup (DOWN)
-- Purpose: Rollback certificate table and changes
-- ============================================================================

-- Remove proficiency_level column
ALTER TABLE candidate_skills DROP COLUMN IF EXISTS proficiency_level;

-- Drop candidate_certificates table
DROP TABLE IF EXISTS candidate_certificates;

-- Remove deprecation comments (optional, comments don't affect functionality)
COMMENT ON TABLE japan_work_experiences IS NULL;
COMMENT ON COLUMN candidate_profiles.skills IS NULL;

-- Note: We do NOT delete migrated work_experiences data as it may have been modified
-- Manual cleanup required if full rollback is needed
