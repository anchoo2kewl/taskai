-- Add ID column to task_tags table to support Ent ORM
-- Postgres version

-- Add id column with SERIAL type (auto-increment)
ALTER TABLE task_tags ADD COLUMN id BIGSERIAL;

-- Set id as primary key (will auto-generate values for existing rows)
-- First, we need to populate id for existing rows
-- SERIAL will auto-populate, but we need to make it the primary key

-- Drop existing primary key constraint if it exists (task_id, tag_id)
-- Note: In Postgres, we need to find the constraint name first
-- For safety, we'll recreate the table like in SQLite version

-- Create new table with ID as primary key
CREATE TABLE task_tags_new (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Copy data from old table
INSERT INTO task_tags_new (task_id, tag_id, created_at)
SELECT task_id, tag_id, created_at FROM task_tags;

-- Drop old table
DROP TABLE task_tags;

-- Rename new table
ALTER TABLE task_tags_new RENAME TO task_tags;

-- Recreate indexes
CREATE INDEX idx_task_tags_task_id ON task_tags(task_id);
CREATE INDEX idx_task_tags_tag_id ON task_tags(tag_id);
