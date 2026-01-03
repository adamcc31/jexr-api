-- Migration: 000017_fix_orphan_fks
-- Purpose: Remove duplicate/default-named foreign keys that block ON UPDATE CASCADE.
-- The error "users" violates foreign key constraint "company_profiles_user_id_fkey" implies
-- that a default-named constraint exists and was NOT dropped by 000016 (which dropped 'fk_company_user').

-- 1. company_profiles
-- Drop the default auto-generated name if it exists
ALTER TABLE company_profiles DROP CONSTRAINT IF EXISTS company_profiles_user_id_fkey;

-- 2. candidate_profiles
-- Drop the default auto-generated name if it exists
ALTER TABLE candidate_profiles DROP CONSTRAINT IF EXISTS candidate_profiles_user_id_fkey;

-- Note: We generally don't need to re-add constraints here because 000016 already added
-- 'fk_company_user' and 'fk_user' with ON UPDATE CASCADE.
-- If 000016 was run successfully, those CASCADE constraints should be active.
-- This migration just removes the "zombie" blocking constraints.
