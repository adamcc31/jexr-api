-- ============================================================================
-- Migration: 000015_add_portfolio_and_completion
-- Purpose: Add portfolio_url column and update verification completion logic
-- ============================================================================

-- 1. Add portfolio_url column (optional field, separate from cv_url)
ALTER TABLE account_verifications
    ADD COLUMN IF NOT EXISTS portfolio_url TEXT;

-- 2. Add comment for clarity
COMMENT ON COLUMN account_verifications.cv_url IS 'Mandatory: URL to uploaded CV/Resume document file';
COMMENT ON COLUMN account_verifications.portfolio_url IS 'Optional: URL to external portfolio (GitHub, LinkedIn, etc)';
