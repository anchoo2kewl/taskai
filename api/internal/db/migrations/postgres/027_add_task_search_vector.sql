-- Add full-text search vector to tasks table
-- Weight A for title (most important), B for description
ALTER TABLE tasks ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(description, '')), 'B')
    ) STORED;

-- GIN index for fast full-text search
CREATE INDEX idx_tasks_search_vector ON tasks USING GIN(search_vector);
