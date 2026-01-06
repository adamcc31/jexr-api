-- ============================================================================
-- Migration: 000015_candidate_certificates_and_cleanup
-- Purpose: Add certificate storage for TOEFL/IELTS/TOEIC, migrate legacy data,
--          add proficiency level to skills
-- ============================================================================

-- ============================================================================
-- PART A: Create candidate_certificates table
-- NOTE: JLPT stays in account_verifications (japanese_level, japanese_certificate_url)
-- ============================================================================

CREATE TABLE IF NOT EXISTS candidate_certificates (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    certificate_type VARCHAR(20) NOT NULL 
        CHECK (certificate_type IN ('TOEFL', 'IELTS', 'TOEIC', 'OTHER')),
    certificate_name TEXT, -- For 'OTHER' type, specify name
    score_total NUMERIC(5,1),
    score_details JSONB, -- Sub-scores like {reading: 25, listening: 28, ...}
    issued_date DATE,
    expires_date DATE,
    document_file_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for lookups and filtering
CREATE INDEX IF NOT EXISTS idx_candidate_certificates_user_id ON candidate_certificates(user_id);
CREATE INDEX IF NOT EXISTS idx_candidate_certificates_type ON candidate_certificates(certificate_type);

-- ============================================================================
-- PART B: Add proficiency level to candidate_skills pivot table
-- ============================================================================

ALTER TABLE candidate_skills
ADD COLUMN IF NOT EXISTS proficiency_level VARCHAR(20) 
    CHECK (proficiency_level IS NULL OR proficiency_level IN ('BEGINNER', 'INTERMEDIATE', 'ADVANCED', 'EXPERT'));

-- ============================================================================
-- PART C: Migrate japan_work_experiences data to unified work_experiences table
-- ============================================================================

INSERT INTO work_experiences (user_id, country_code, experience_type, company_name, job_title, start_date, end_date, description, created_at, updated_at)
SELECT 
    av.user_id,
    'JP' as country_code,
    'OVERSEAS' as experience_type,
    jwe.company_name,
    jwe.job_title,
    jwe.start_date,
    jwe.end_date,
    jwe.description,
    jwe.created_at,
    jwe.updated_at
FROM japan_work_experiences jwe
JOIN account_verifications av ON jwe.account_verification_id = av.id
ON CONFLICT DO NOTHING;

-- Mark legacy table as deprecated (keep for rollback safety)
COMMENT ON TABLE japan_work_experiences IS 'DEPRECATED: Migrated to work_experiences with country_code=JP. Use work_experiences instead.';

-- ============================================================================
-- PART D: Mark legacy skills column as deprecated
-- ============================================================================

COMMENT ON COLUMN candidate_profiles.skills IS 'DEPRECATED: Use skills + candidate_skills normalized tables instead.';
