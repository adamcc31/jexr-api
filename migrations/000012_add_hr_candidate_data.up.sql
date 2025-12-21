-- ============================================================================
-- Migration: 000012_add_hr_candidate_data
-- Purpose: Add HR-grade candidate fields to account_verifications table
-- ============================================================================

-- A. Identity & Demographics
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS birth_date DATE,
ADD COLUMN IF NOT EXISTS domicile_city TEXT,
ADD COLUMN IF NOT EXISTS marital_status TEXT CHECK (marital_status IN ('SINGLE', 'MARRIED', 'DIVORCED')),
ADD COLUMN IF NOT EXISTS children_count INTEGER DEFAULT 0;

-- B. Core Competencies
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS main_job_fields TEXT[],
ADD COLUMN IF NOT EXISTS golden_skill TEXT,
ADD COLUMN IF NOT EXISTS japanese_speaking_level TEXT CHECK (japanese_speaking_level IN ('NATIVE', 'FLUENT', 'BASIC', 'PASSIVE'));

-- C. Expectations & Availability
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS expected_salary BIGINT,
ADD COLUMN IF NOT EXISTS japan_return_date DATE,
ADD COLUMN IF NOT EXISTS available_start_date DATE,
ADD COLUMN IF NOT EXISTS preferred_locations TEXT[],
ADD COLUMN IF NOT EXISTS preferred_industries TEXT[];

-- D. Supporting Documents
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS supporting_certificates_url TEXT[];

-- ============================================================================
-- Indexes for HR search performance
-- ============================================================================

-- GIN indexes for array fields (efficient containment queries)
CREATE INDEX IF NOT EXISTS idx_av_main_job_fields ON account_verifications USING GIN (main_job_fields);
CREATE INDEX IF NOT EXISTS idx_av_preferred_locations ON account_verifications USING GIN (preferred_locations);
CREATE INDEX IF NOT EXISTS idx_av_preferred_industries ON account_verifications USING GIN (preferred_industries);

-- B-Tree indexes for scalar filtering fields
CREATE INDEX IF NOT EXISTS idx_av_birth_date ON account_verifications (birth_date);
CREATE INDEX IF NOT EXISTS idx_av_marital_status ON account_verifications (marital_status);
CREATE INDEX IF NOT EXISTS idx_av_expected_salary ON account_verifications (expected_salary);
CREATE INDEX IF NOT EXISTS idx_av_available_start_date ON account_verifications (available_start_date);
CREATE INDEX IF NOT EXISTS idx_av_japanese_speaking_level ON account_verifications (japanese_speaking_level);
