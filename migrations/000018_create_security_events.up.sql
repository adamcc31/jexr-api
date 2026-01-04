CREATE TABLE security_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    service VARCHAR(100),
    environment VARCHAR(50),
    level VARCHAR(20),
    subject_type VARCHAR(20), -- 'email', 'ip', 'user_id'
    subject_value VARCHAR(255), -- Masked/hashed for PII protection
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(36),
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_security_events_created_at ON security_events(created_at);
CREATE INDEX idx_security_events_level ON security_events(level);
CREATE INDEX idx_security_events_type ON security_events(event_type);
CREATE INDEX idx_security_events_subject ON security_events(subject_type, subject_value);

-- Enable Row Level Security
ALTER TABLE security_events ENABLE ROW LEVEL SECURITY;

-- Policy: Only service_role can access (backend uses service_role key)
-- This prevents anonymous/authenticated frontend users from reading security logs
CREATE POLICY "Service role full access" ON security_events
    FOR ALL
    USING (auth.role() = 'service_role');

-- Retention: Events older than 90 days can be purged by scheduled job
COMMENT ON TABLE security_events IS 'Security audit log with 90-day retention policy';
