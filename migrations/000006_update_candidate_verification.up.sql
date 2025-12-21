-- Add columns to account_verifications
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS first_name TEXT,
ADD COLUMN IF NOT EXISTS last_name TEXT,
ADD COLUMN IF NOT EXISTS profile_picture_url TEXT,
ADD COLUMN IF NOT EXISTS occupation TEXT,
ADD COLUMN IF NOT EXISTS phone TEXT,
ADD COLUMN IF NOT EXISTS website_url TEXT,
ADD COLUMN IF NOT EXISTS intro TEXT,
ADD COLUMN IF NOT EXISTS japan_experience_duration INTEGER, -- Stored in months
ADD COLUMN IF NOT EXISTS japanese_certificate_url TEXT,
ADD COLUMN IF NOT EXISTS cv_url TEXT; -- CV/Resume document URL

-- Update status constraint to include SUBMITTED
ALTER TABLE account_verifications DROP CONSTRAINT IF EXISTS account_verifications_status_check;
ALTER TABLE account_verifications ADD CONSTRAINT account_verifications_status_check 
    CHECK (status IN ('PENDING', 'SUBMITTED', 'VERIFIED', 'REJECTED'));

-- Create japan_work_experiences table
CREATE TABLE IF NOT EXISTS japan_work_experiences (
    id BIGSERIAL PRIMARY KEY,
    account_verification_id BIGINT NOT NULL REFERENCES account_verifications(id) ON DELETE CASCADE,
    company_name TEXT NOT NULL,
    job_title TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_japan_work_experiences_verification_id ON japan_work_experiences(account_verification_id);
