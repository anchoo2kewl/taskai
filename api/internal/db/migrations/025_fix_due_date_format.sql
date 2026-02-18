-- Fix due_date format to be compatible with Ent's time.Time type
-- The old schema stored due_date as TEXT, but Ent expects proper datetime
-- SQLite datetime columns should store values in ISO8601 format or as Unix timestamps
-- This migration clears all existing due_date values to allow clean migration to Ent
-- Users can re-enter due dates through the new Ent-based system

UPDATE tasks SET due_date = NULL WHERE due_date IS NOT NULL;
