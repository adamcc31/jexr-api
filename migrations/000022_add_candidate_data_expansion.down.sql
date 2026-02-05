-- ============================================================================
-- Migration: 000022_add_candidate_data_expansion.down.sql
-- Purpose: Rollback Height, Weight, Religion, JLPT Issue Year, Interview Willingness
-- ============================================================================

ALTER TABLE account_verifications
DROP COLUMN IF EXISTS height_cm,
DROP COLUMN IF EXISTS weight_kg,
DROP COLUMN IF EXISTS religion,
DROP COLUMN IF EXISTS jlpt_certificate_issue_year,
DROP COLUMN IF EXISTS willing_to_interview_onsite;
