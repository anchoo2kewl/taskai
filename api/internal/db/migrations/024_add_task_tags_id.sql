-- Add ID column to task_tags table to support Ent ORM
-- SQLite doesn't support ALTER TABLE to add PRIMARY KEY, so we need to recreate the table

-- Create new table with ID
CREATE TABLE task_tags_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
