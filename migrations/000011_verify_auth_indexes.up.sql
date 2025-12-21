-- Migration: Verify and create optimal indexes for users table
-- Purpose: Improve query performance for application-side lookups
--
-- NOTE: This does NOT improve Supabase Auth login performance.
-- Supabase Auth uses the auth.users schema (managed by Supabase).
-- These indexes are for YOUR application's queries on public.users:
--   - Admin dashboard user search
--   - Role-based user listings
--   - Email lookups in your application code

-- Index on email (should already exist via UNIQUE constraint from init migration)
-- Using IF NOT EXISTS to be safe for re-runs
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Index on role for filtering users by type
-- Useful for: admin dashboard, role-based statistics, user management
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Composite index for common admin queries (paginated user lists by role)
CREATE INDEX IF NOT EXISTS idx_users_role_created ON users(role, created_at DESC);

-- IMPORTANT: For Supabase Auth performance issues (the 163k rows problem),
-- you need to check indexing in the Supabase Dashboard:
--
-- Run this in Supabase SQL Editor to check auth.users indexes:
-- 
--   SELECT indexname, indexdef 
--   FROM pg_indexes 
--   WHERE tablename = 'users' 
--   AND schemaname = 'auth';
--
-- Supabase should have indexes on auth.users.email by default.
-- If you're seeing full table scans, contact Supabase support or check:
-- - Database health in Supabase Dashboard
-- - Any custom triggers on auth.users you may have added
-- - RLS policies that might prevent index usage
