-- Add agent_name column to track AI agent attribution
-- NULL means human-created, non-NULL means agent-created (e.g. "Claude Code")

ALTER TABLE task_comments ADD COLUMN agent_name VARCHAR(100);
ALTER TABLE tasks ADD COLUMN agent_name VARCHAR(100);
