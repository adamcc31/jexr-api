-- ============================================================================
-- Migration: 000013_create_onboarding_tables (DOWN)
-- Purpose: Rollback onboarding tables and columns
-- ============================================================================

-- Remove constraint first
ALTER TABLE account_verifications DROP CONSTRAINT IF EXISTS chk_lpk_exclusive;

-- Remove indexes
DROP INDEX IF EXISTS idx_av_onboarding_completed;
DROP INDEX IF EXISTS idx_lpk_list_name;
DROP INDEX IF EXISTS idx_candidate_company_prefs_user_id;
DROP INDEX IF EXISTS idx_candidate_interests_user_id;

-- Remove columns from account_verifications
ALTER TABLE account_verifications
    DROP COLUMN IF EXISTS onboarding_completed_at,
    DROP COLUMN IF EXISTS lpk_none,
    DROP COLUMN IF EXISTS lpk_other_name,
    DROP COLUMN IF EXISTS lpk_id;

-- Drop tables (order matters due to FK references)
DROP TABLE IF EXISTS candidate_company_preferences;
DROP TABLE IF EXISTS candidate_interests;
DROP TABLE IF EXISTS lpk_list;
