-- Migration: 000020_security_dashboard.up.sql
-- Security Dashboard Tables for Cyber Security Monitoring Console
-- This is a COMPLETELY SEPARATE authentication system from the main app

-- Security User Roles Enum
CREATE TYPE security_role AS ENUM (
    'SECURITY_OBSERVER',  -- Read-only logs
    'SECURITY_ANALYST',   -- Filter, correlate, export with approval
    'SECURITY_ADMIN'      -- System config, retention policy, alert rules, break-glass approval
);

-- Security Event Severity Enum (hard-coded, not user-provided)
CREATE TYPE security_severity AS ENUM (
    'INFO',
    'MEDIUM',
    'WARN',
    'HIGH',
    'CRITICAL'
);

-- Security Users Table (COMPLETELY SEPARATE from main users table)
-- These are security operators, not application users
CREATE TABLE security_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,  -- bcrypt hash
    role security_role NOT NULL DEFAULT 'SECURITY_OBSERVER',
    totp_secret VARCHAR(64),  -- Base32 encoded TOTP secret
    totp_enabled BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES security_users(id)
);

-- Allowed IP Ranges for Security Dashboard Access
-- This is the PRIMARY security control (not the hidden route)
CREATE TABLE allowed_ip_ranges (
    id SERIAL PRIMARY KEY,
    cidr CIDR NOT NULL,
    description VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES security_users(id)
);

-- Security Sessions (separate from main app sessions)
CREATE TABLE security_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    security_user_id UUID NOT NULL REFERENCES security_users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,  -- SHA-256 hash of session token
    ip_address INET NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    CONSTRAINT session_expiry CHECK (expires_at > created_at)
);

-- Break-Glass Sessions for DEVELOPER_ROOT elevation
-- Time-limited emergency access with mandatory justification
CREATE TABLE break_glass_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    security_user_id UUID NOT NULL REFERENCES security_users(id) ON DELETE CASCADE,
    justification TEXT NOT NULL CHECK (length(justification) >= 50),
    activated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    -- Maximum 60 minutes duration
    CONSTRAINT valid_duration CHECK (expires_at <= activated_at + INTERVAL '60 minutes')
);

-- Hash Anchors for Log Integrity Verification
-- References to externally-stored (S3 Object Lock) root hashes
CREATE TABLE hash_anchors (
    id SERIAL PRIMARY KEY,
    anchor_date DATE UNIQUE NOT NULL,
    root_hash VARCHAR(64) NOT NULL,  -- SHA-256 hex
    event_count INT NOT NULL,
    first_event_id BIGINT,
    last_event_id BIGINT,
    s3_key VARCHAR(255) NOT NULL,  -- e.g., 'security-anchors/2026-01-08.hash'
    verified_at TIMESTAMPTZ,
    verification_status VARCHAR(20) DEFAULT 'pending',  -- pending, verified, failed
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Export Requests (requires approval workflow)
CREATE TABLE export_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requested_by UUID NOT NULL REFERENCES security_users(id),
    filter_start_time TIMESTAMPTZ,
    filter_end_time TIMESTAMPTZ,
    filter_event_types TEXT[],
    filter_severity security_severity[],
    filter_ip TEXT,
    filter_subject TEXT,
    justification TEXT NOT NULL CHECK (length(justification) >= 20),
    status VARCHAR(20) DEFAULT 'pending',  -- pending, approved, rejected, expired
    approved_by UUID REFERENCES security_users(id),
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,
    download_count INT DEFAULT 0,
    download_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add severity column to security_events table
ALTER TABLE security_events ADD COLUMN IF NOT EXISTS severity security_severity;

-- Add previous_hash column for hash chaining
ALTER TABLE security_events ADD COLUMN IF NOT EXISTS previous_hash VARCHAR(64);

-- Add row_hash column for integrity verification
ALTER TABLE security_events ADD COLUMN IF NOT EXISTS row_hash VARCHAR(64);

-- Create indexes for security dashboard queries
CREATE INDEX IF NOT EXISTS idx_security_events_severity ON security_events(severity);
CREATE INDEX IF NOT EXISTS idx_security_events_subject_ip ON security_events(ip_address, created_at);
CREATE INDEX IF NOT EXISTS idx_security_sessions_user ON security_sessions(security_user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_security_sessions_token ON security_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_break_glass_active ON break_glass_sessions(security_user_id, expires_at) 
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_export_requests_status ON export_requests(status, created_at);

-- Enable Row Level Security on all security tables
ALTER TABLE security_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE break_glass_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE allowed_ip_ranges ENABLE ROW LEVEL SECURITY;
ALTER TABLE hash_anchors ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_requests ENABLE ROW LEVEL SECURITY;

-- RLS Policies: Only service_role can access these tables
CREATE POLICY "Service role full access" ON security_users
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access" ON security_sessions
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access" ON break_glass_sessions
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access" ON allowed_ip_ranges
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access" ON hash_anchors
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access" ON export_requests
    FOR ALL USING (auth.role() = 'service_role');

-- Function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_security_user_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER security_users_updated_at
    BEFORE UPDATE ON security_users
    FOR EACH ROW EXECUTE FUNCTION update_security_user_timestamp();

-- Comments for documentation
COMMENT ON TABLE security_users IS 'Security dashboard operators - SEPARATE from main app users';
COMMENT ON TABLE allowed_ip_ranges IS 'IP allowlist - PRIMARY access control for security dashboard';
COMMENT ON TABLE security_sessions IS 'Security dashboard sessions with 30-min timeout';
COMMENT ON TABLE break_glass_sessions IS 'Time-limited DEVELOPER_ROOT elevation with mandatory justification';
COMMENT ON TABLE hash_anchors IS 'References to S3 Object Lock anchored root hashes for tamper evidence';
COMMENT ON TABLE export_requests IS 'Security log export workflow with mandatory approval';
COMMENT ON COLUMN security_events.severity IS 'Derived from event_type via hard-coded mapping, not user-provided';
COMMENT ON COLUMN security_events.previous_hash IS 'SHA-256 hash of previous event for chain verification';
COMMENT ON COLUMN security_events.row_hash IS 'SHA-256 hash of this row for integrity verification';
