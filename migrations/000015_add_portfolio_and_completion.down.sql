-- ============================================================================
-- Migration: 000015_add_portfolio_and_completion (DOWN)
-- Purpose: Rollback portfolio_url column addition
-- ============================================================================

ALTER TABLE account_verifications
    DROP COLUMN IF EXISTS portfolio_url;
