-- Extend company_profiles table with additional employer profile fields
-- Note: industry, description, website, location already exist from 000003

ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS logo_url TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS company_story TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS founded TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS founder TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS headquarters TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS employee_count TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS hide_company_details BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS gallery_image_1 TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS gallery_image_2 TEXT;
ALTER TABLE company_profiles ADD COLUMN IF NOT EXISTS gallery_image_3 TEXT;

-- Create index for faster public profile lookups
CREATE INDEX IF NOT EXISTS idx_company_profiles_id ON company_profiles(id);
