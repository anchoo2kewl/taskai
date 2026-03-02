-- Add first_name and last_name columns to users table
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name TEXT;

-- Backfill from existing name column
UPDATE users SET
  first_name = CASE
    WHEN name IS NOT NULL AND POSITION(' ' IN name) > 0 THEN SUBSTRING(name, 1, POSITION(' ' IN name) - 1)
    WHEN name IS NOT NULL THEN name
    ELSE NULL
  END,
  last_name = CASE
    WHEN name IS NOT NULL AND POSITION(' ' IN name) > 0 THEN SUBSTRING(name, POSITION(' ' IN name) + 1)
    ELSE NULL
  END;
