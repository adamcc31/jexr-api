-- Add japanese_level column to account_verifications
ALTER TABLE account_verifications
ADD COLUMN IF NOT EXISTS japanese_level VARCHAR(3); -- N5, N4, N3, N2, N1
