-- ============================================================================
-- Migration: 000014_update_candidate_profile_schema
-- Purpose: Update candidate profiles and add new normalized tables
-- ============================================================================

-- 1. Alter candidate_profiles
-- Non-destructive additions to the existing table
ALTER TABLE candidate_profiles
    ADD COLUMN IF NOT EXISTS highest_education TEXT,
    ADD COLUMN IF NOT EXISTS major_field TEXT,
    ADD COLUMN IF NOT EXISTS desired_job_position TEXT, -- ID/Code if master exists, else text
    ADD COLUMN IF NOT EXISTS desired_job_position_other TEXT, -- Text input for 'Others'
    ADD COLUMN IF NOT EXISTS preferred_work_environment TEXT,
    ADD COLUMN IF NOT EXISTS career_goals_3y TEXT,
    ADD COLUMN IF NOT EXISTS main_concerns_returning TEXT[],
    ADD COLUMN IF NOT EXISTS special_message TEXT,
    ADD COLUMN IF NOT EXISTS skills_other TEXT; -- JSON or newline separated string

-- 2. Create candidate_details
-- For narrative and long-form content that doesn't need high-freq filtering
CREATE TABLE IF NOT EXISTS candidate_details (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    soft_skills_description TEXT,
    applied_work_values TEXT,
    major_achievements TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Create work_experiences
-- Unified table handling both Local and Overseas experience
-- Replaces any country-specific legacy tables for backing storage
CREATE TABLE IF NOT EXISTS work_experiences (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    country_code VARCHAR(2) NOT NULL, -- ISO-2 Country Code (e.g., 'ID', 'JP')
    experience_type VARCHAR(20) NOT NULL CHECK (experience_type IN ('LOCAL', 'OVERSEAS')),
    company_name TEXT NOT NULL,
    job_title TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE, -- NULL indicates 'Present'
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_work_experiences_user_id ON work_experiences(user_id);

-- 4. Create skills (Master table)
-- Searchable master list of skills
CREATE TABLE IF NOT EXISTS skills (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    category VARCHAR(50) NOT NULL -- e.g., 'COMPUTER', 'LANGUAGE', 'TECHNICAL'
);

CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);

-- 5. Create candidate_skills (Pivot table)
-- Normalizes the many-to-many relationship between candidates and skills
CREATE TABLE IF NOT EXISTS candidate_skills (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_candidate_skills_user_id ON candidate_skills(user_id);
