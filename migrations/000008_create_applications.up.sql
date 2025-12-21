-- Create applications table for job applications
CREATE TABLE IF NOT EXISTS applications (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    candidate_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_verification_id BIGINT REFERENCES account_verifications(id),
    cv_url TEXT NOT NULL,  -- Required for application submission
    cover_letter TEXT,
    status TEXT NOT NULL DEFAULT 'applied' CHECK (status IN ('applied', 'reviewed', 'accepted', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_application_job_candidate UNIQUE (job_id, candidate_user_id)
);

-- Indexes for efficient querying
CREATE INDEX idx_applications_job_id ON applications(job_id);
CREATE INDEX idx_applications_candidate ON applications(candidate_user_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_created_at ON applications(created_at DESC);
