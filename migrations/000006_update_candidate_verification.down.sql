-- Drop japan_work_experiences table
DROP TABLE IF EXISTS japan_work_experiences;

-- Remove columns from account_verifications
ALTER TABLE account_verifications
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name,
DROP COLUMN IF EXISTS profile_picture_url,
DROP COLUMN IF EXISTS occupation,
DROP COLUMN IF EXISTS phone,
DROP COLUMN IF EXISTS website_url,
DROP COLUMN IF EXISTS intro,
DROP COLUMN IF EXISTS japan_experience_duration,
DROP COLUMN IF EXISTS japanese_certificate_url;
