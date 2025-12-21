-- Remove unique constraint
ALTER TABLE company_profiles DROP CONSTRAINT IF EXISTS company_profiles_user_id_key;
