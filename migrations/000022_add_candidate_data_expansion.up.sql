-- ============================================================================
-- Migration: 000022_add_candidate_data_expansion
-- Purpose: Add Height, Weight, Religion, JLPT Issue Year, Interview Willingness
-- ============================================================================

-- A. Physical Attributes (Height & Weight)
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS height_cm SMALLINT 
    CHECK (height_cm IS NULL OR (height_cm >= 50 AND height_cm <= 300)),
ADD COLUMN IF NOT EXISTS weight_kg NUMERIC(4,1)
    CHECK (weight_kg IS NULL OR (weight_kg >= 10.0 AND weight_kg <= 500.0));

-- B. Religion Field with Indonesian Standard Values
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS religion TEXT
    CHECK (religion IS NULL OR religion IN ('ISLAM', 'KRISTEN', 'KATOLIK', 'HINDU', 'BUDDHA', 'KONGHUCU', 'OTHER'));

-- C. JLPT Certificate Issue Year (associated with JLPT certificate upload)
-- NOTE: Upper bound (current year) validated at APPLICATION layer, not DB
-- PostgreSQL CHECK constraints must be immutable; CURRENT_DATE is not allowed
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS jlpt_certificate_issue_year SMALLINT
    CHECK (jlpt_certificate_issue_year IS NULL OR jlpt_certificate_issue_year >= 1984);

-- D. Interview Willingness (Onboarding context question)
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS willing_to_interview_onsite BOOLEAN;

-- ============================================================================
-- Comments for Documentation
-- ============================================================================

COMMENT ON COLUMN account_verifications.height_cm IS 'Candidate height in centimeters (50-300 valid range)';
COMMENT ON COLUMN account_verifications.weight_kg IS 'Candidate weight in kilograms with 1 decimal precision';
COMMENT ON COLUMN account_verifications.religion IS 'Candidate religion per Indonesian standard categories';
COMMENT ON COLUMN account_verifications.jlpt_certificate_issue_year IS 'Year when JLPT certificate was issued (1984 onwards, upper bound validated at app layer)';
COMMENT ON COLUMN account_verifications.willing_to_interview_onsite IS 'Candidate willingness to attend onsite interview at company office';
