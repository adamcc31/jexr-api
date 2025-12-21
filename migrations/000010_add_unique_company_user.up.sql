-- Add unique constraint to user_id in company_profiles to support ON CONFLICT
ALTER TABLE company_profiles ADD CONSTRAINT company_profiles_user_id_key UNIQUE (user_id);
