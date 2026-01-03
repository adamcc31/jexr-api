-- ============================================================================
-- Migration: 000014_update_candidate_profile_schema (DOWN)
-- Purpose: Revert schema changes
-- ============================================================================

DROP TABLE IF EXISTS candidate_skills;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS work_experiences;
DROP TABLE IF EXISTS candidate_details;

ALTER TABLE candidate_profiles
    DROP COLUMN IF EXISTS highest_education,
    DROP COLUMN IF EXISTS major_field,
    DROP COLUMN IF EXISTS desired_job_position,
    DROP COLUMN IF EXISTS desired_job_position_other,
    DROP COLUMN IF EXISTS preferred_work_environment,
    DROP COLUMN IF EXISTS career_goals_3y,
    DROP COLUMN IF EXISTS main_concerns_returning,
    DROP COLUMN IF EXISTS special_message,
    DROP COLUMN IF EXISTS skills_other;
