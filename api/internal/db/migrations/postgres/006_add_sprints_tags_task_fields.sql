-- Add sprints, tags, and enhanced task fields

-- Sprints table (shared across projects for the user)
CREATE TABLE sprints (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    goal TEXT,
    start_date DATE,
    end_date DATE,
    status TEXT NOT NULL CHECK(status IN ('planned', 'active', 'completed')) DEFAULT 'planned',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sprints_user_id ON sprints(user_id);
CREATE INDEX idx_sprints_status ON sprints(status);

-- Tags table (shared across projects for the user)
CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#3B82F6',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

CREATE INDEX idx_tags_user_id ON tags(user_id);

-- Add new fields to tasks table
ALTER TABLE tasks ADD COLUMN sprint_id BIGINT REFERENCES sprints(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN priority TEXT CHECK(priority IN ('low', 'medium', 'high', 'urgent')) DEFAULT 'medium';
ALTER TABLE tasks ADD COLUMN assignee_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN estimated_hours REAL;
ALTER TABLE tasks ADD COLUMN actual_hours REAL;

CREATE INDEX idx_tasks_sprint_id ON tasks(sprint_id);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);

-- Task tags junction table (many-to-many)
CREATE TABLE task_tags (
    task_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_tags_task_id ON task_tags(task_id);
CREATE INDEX idx_task_tags_tag_id ON task_tags(tag_id);

-- Trigger to update updated_at timestamp for sprints
CREATE TRIGGER update_sprints_timestamp
BEFORE UPDATE ON sprints
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
