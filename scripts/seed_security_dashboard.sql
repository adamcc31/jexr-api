-- Security Dashboard Initial Setup Script
-- Run this AFTER migration 000020_security_dashboard.up.sql
-- This creates the initial SECURITY_ADMIN user for dashboard access

-- IMPORTANT: Change these values before running in production!
-- Password should be hashed with bcrypt (cost 10+)
-- Generate with: htpasswd -nbBC 10 "" "your-password" | tr -d ':\n' | sed 's/$2y/$2a/'

-- Step 1: Insert initial allowed IP ranges
-- Modify these to match your SOC/VPN network
INSERT INTO allowed_ip_ranges (cidr, description, is_active) VALUES
    ('10.0.0.0/8', 'Internal corporate network', true),
    ('172.16.0.0/12', 'VPN network range', true),
    ('192.168.0.0/16', 'Development network', true),
    ('180.242.207.37', 'master (master login)', true)
ON CONFLICT DO NOTHING;

INSERT INTO security_users (
    username,
    email,
    password_hash,
    role,
    totp_enabled,
    is_active
) VALUES (
    'secadmin',
    'security@jexpert.internal',
    '$2y$19$gKmY1v3WWLsE55BVoOqGjuCCDwrlqTQrnmKhK41D3NEBZKtYyIxei',
    'SECURITY_ADMIN',
    false,
    true
) ON CONFLICT (username) DO NOTHING;

-- Step 3: Create read-only observer for monitoring
INSERT INTO security_users (
    username,
    email,
    password_hash,
    role,
    totp_enabled,
    is_active
) VALUES (
    'secobserver',
    'soc-observer@jexpert.internal',
    '$2y$19$TVtZqbzDrHU.nG/fMfjjdeAfP07dI03Mr4Xufaw6EV3oXSoUJ8Ae2',
    'SECURITY_OBSERVER',
    false,
    true
) ON CONFLICT (username) DO NOTHING;

-- Step 4: Create analyst user
INSERT INTO security_users (
    username,
    email,
    password_hash,
    role,
    totp_enabled,
    is_active
) VALUES (
    'secanalyst',
    'soc-analyst@jexpert.internal',
    '$2y$19$JsANdO0iYdug9XahdXsL.e7u7lGZsazktW.FmruwSH5Rz6/ecNqG2',
    'SECURITY_ANALYST',
    false,
    true
) ON CONFLICT (username) DO NOTHING;

-- Verification queries
SELECT 'Allowed IP Ranges:' as info;
SELECT id, cidr, description, is_active FROM allowed_ip_ranges;

SELECT 'Security Users:' as info;
SELECT id, username, email, role, totp_enabled, is_active FROM security_users;

-- IMPORTANT NEXT STEPS:
-- 1. Each user must set up TOTP on first login via authenticator app
-- 2. Update password hashes with production-strength passwords
-- 3. Update allowed_ip_ranges with actual SOC/VPN CIDR blocks
-- 4. Configure S3 bucket with Object Lock for hash anchoring:
--    aws s3api create-bucket --bucket jexpert-security-anchors --region ap-southeast-1
--    aws s3api put-object-lock-configuration --bucket jexpert-security-anchors \
--        --object-lock-configuration '{"ObjectLockEnabled":"Enabled","Rule":{"DefaultRetention":{"Mode":"GOVERNANCE","Years":1}}}'
