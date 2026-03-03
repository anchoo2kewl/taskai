-- Add first_name and last_name columns to users table
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name TEXT;

-- Backfill from existing name column
UPDATE users SET
  first_name = CASE
    WHEN name IS NOT NULL AND INSTR(name, ' ') > 0 THEN SUBSTR(name, 1, INSTR(name, ' ') - 1)
    WHEN name IS NOT NULL THEN name
    ELSE NULL
  END,
  last_name = CASE
    WHEN name IS NOT NULL AND INSTR(name, ' ') > 0 THEN SUBSTR(name, INSTR(name, ' ') + 1)
    ELSE NULL
  END;
