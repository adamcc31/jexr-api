-- Create company_profiles table if it doesn't exist (ensures foreign key constraints work)
CREATE TABLE IF NOT EXISTS company_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_name TEXT NOT NULL,
    industry TEXT,
    website TEXT,
    location TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_company_user FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_company_profiles_user_id ON company_profiles(user_id);

-- Create account_verifications table
CREATE TABLE IF NOT EXISTS account_verifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('ADMIN', 'EMPLOYER', 'CANDIDATE')),
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'VERIFIED', 'REJECTED')),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(id),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_account_verifications_user_id UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_account_verifications_status ON account_verifications(status);
CREATE INDEX IF NOT EXISTS idx_account_verifications_role ON account_verifications(role);

-- Create trigger function to handle new verifications
CREATE OR REPLACE FUNCTION handle_new_verification()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO account_verifications (user_id, role, status, submitted_at, updated_at)
    VALUES (
        NEW.user_id,
        CASE
            WHEN TG_TABLE_NAME = 'candidate_profiles' THEN 'CANDIDATE'
            WHEN TG_TABLE_NAME = 'company_profiles' THEN 'EMPLOYER'
            ELSE 'CANDIDATE' -- Default fallback, though should catch by TG_TABLE_NAME
        END,
        'PENDING',
        NOW(),
        NOW()
    )
    ON CONFLICT (user_id) DO UPDATE
    SET
        status = 'PENDING',
        submitted_at = NOW(),
        updated_at = NOW(),
        verified_at = NULL,
        verified_by = NULL
    WHERE account_verifications.status = 'REJECTED'; -- Only reset to PENDING if currently REJECTED

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for candidate_profiles
DROP TRIGGER IF EXISTS trigger_candidate_verification ON candidate_profiles;
CREATE TRIGGER trigger_candidate_verification
AFTER INSERT OR UPDATE ON candidate_profiles
FOR EACH ROW
EXECUTE FUNCTION handle_new_verification();

-- Trigger for company_profiles
DROP TRIGGER IF EXISTS trigger_company_verification ON company_profiles;
CREATE TRIGGER trigger_company_verification
AFTER INSERT OR UPDATE ON company_profiles
FOR EACH ROW
EXECUTE FUNCTION handle_new_verification();
