-- Fix due_date format to be compatible with Ent's time.Time type
-- Postgres version

-- In Postgres, due_date is already stored as TIMESTAMPTZ or similar
-- This migration ensures consistency with the SQLite migration
-- Clear any TEXT-formatted dates that might exist from improper inserts

UPDATE tasks SET due_date = NULL
WHERE due_date IS NOT NULL
  AND pg_typeof(due_date)::text != 'timestamp with time zone';
