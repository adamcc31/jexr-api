-- Remove japanese_level column from account_verifications
ALTER TABLE account_verifications
DROP COLUMN IF EXISTS japanese_level;
