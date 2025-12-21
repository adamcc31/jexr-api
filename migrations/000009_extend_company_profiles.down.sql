-- Rollback company_profiles extension

DROP INDEX IF EXISTS idx_company_profiles_id;

ALTER TABLE company_profiles DROP COLUMN IF EXISTS logo_url;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS company_story;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS founded;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS founder;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS headquarters;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS employee_count;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS hide_company_details;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS gallery_image_1;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS gallery_image_2;
ALTER TABLE company_profiles DROP COLUMN IF EXISTS gallery_image_3;
