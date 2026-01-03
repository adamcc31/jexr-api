-- Revert changes: Remove ON UPDATE CASCADE

-- 1. company_profiles
ALTER TABLE company_profiles DROP CONSTRAINT IF EXISTS fk_company_user;
ALTER TABLE company_profiles ADD CONSTRAINT fk_company_user 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 2. candidate_profiles
ALTER TABLE candidate_profiles DROP CONSTRAINT IF EXISTS fk_user;
ALTER TABLE candidate_profiles ADD CONSTRAINT fk_user 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 3. account_verifications (user_id)
ALTER TABLE account_verifications DROP CONSTRAINT IF EXISTS account_verifications_user_id_fkey;
ALTER TABLE account_verifications ADD CONSTRAINT account_verifications_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 4. account_verifications (verified_by)
ALTER TABLE account_verifications DROP CONSTRAINT IF EXISTS account_verifications_verified_by_fkey;
ALTER TABLE account_verifications ADD CONSTRAINT account_verifications_verified_by_fkey 
    FOREIGN KEY (verified_by) REFERENCES users(id);

-- 5. applications (candidate_user_id)
ALTER TABLE applications DROP CONSTRAINT IF EXISTS applications_candidate_user_id_fkey;
ALTER TABLE applications ADD CONSTRAINT applications_candidate_user_id_fkey 
    FOREIGN KEY (candidate_user_id) REFERENCES users(id) ON DELETE CASCADE;
