-- Add task search support
-- SQLite version: no-op (SQLite uses LIKE fallback in search handlers)
-- Postgres has tsvector + GIN index for full-text search
SELECT 1;
