-- ============================================================================
-- Migration: 000013_create_onboarding_tables
-- Purpose: Create tables for candidate onboarding wizard data
-- ============================================================================

-- A. LPK Reference Table (seeded from list-lpk.csv)
-- Stores all known LPK (Lembaga Pelatihan Kerja) training centers
CREATE TABLE IF NOT EXISTS lpk_list (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- B. Candidate Interests (Step 1: Special Interest Survey)
-- Stores selected job interests as individual rows for extensibility
-- interest_key values: 'teacher', 'translator', 'admin', 'none'
CREATE TABLE IF NOT EXISTS candidate_interests (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    interest_key TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, interest_key)
);

-- C. Candidate Company Preferences (Step 3: Company Type)
-- Stores preferred company ownership types
-- preference_key values: 'pma', 'joint_venture', 'local'
CREATE TABLE IF NOT EXISTS candidate_company_preferences (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, preference_key)
);

-- D. LPK Selection & Onboarding Completion (Step 2 data + completion flag)
-- Added to account_verifications for single source of truth
ALTER TABLE account_verifications
    ADD COLUMN IF NOT EXISTS lpk_id INTEGER REFERENCES lpk_list(id),
    ADD COLUMN IF NOT EXISTS lpk_other_name TEXT,
    ADD COLUMN IF NOT EXISTS lpk_none BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;

-- ============================================================================
-- Indexes for performance
-- ============================================================================

-- Fast lookup by user_id for interests and preferences
CREATE INDEX IF NOT EXISTS idx_candidate_interests_user_id ON candidate_interests(user_id);
CREATE INDEX IF NOT EXISTS idx_candidate_company_prefs_user_id ON candidate_company_preferences(user_id);

-- Simple B-tree index on LPK names for autocomplete (ILIKE will still work)
CREATE INDEX IF NOT EXISTS idx_lpk_list_name ON lpk_list(name);

-- Onboarding completion status lookup
CREATE INDEX IF NOT EXISTS idx_av_onboarding_completed ON account_verifications(onboarding_completed_at) WHERE onboarding_completed_at IS NOT NULL;

-- ============================================================================
-- Constraints
-- ============================================================================

-- Ensure LPK selection is mutually exclusive
-- Only one of: lpk_id, lpk_other_name, or lpk_none can be set
ALTER TABLE account_verifications
    ADD CONSTRAINT chk_lpk_exclusive CHECK (
        (lpk_id IS NOT NULL AND lpk_other_name IS NULL AND lpk_none = FALSE) OR
        (lpk_id IS NULL AND lpk_other_name IS NOT NULL AND lpk_none = FALSE) OR
        (lpk_id IS NULL AND lpk_other_name IS NULL AND lpk_none = TRUE) OR
        (lpk_id IS NULL AND lpk_other_name IS NULL AND lpk_none = FALSE) -- None selected yet
    );
