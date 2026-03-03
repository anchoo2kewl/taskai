-- GitHub bidirectional sync: push comments and swim lane status back to GitHub

-- Projects V2 GraphQL IDs needed for push operations
ALTER TABLE projects ADD COLUMN github_push_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE projects ADD COLUMN github_project_id TEXT;
ALTER TABLE projects ADD COLUMN github_status_field_id TEXT;

-- Track GitHub comment ID so we can deduplicate on import and update on push
ALTER TABLE task_comments ADD COLUMN github_comment_id BIGINT;
CREATE UNIQUE INDEX idx_task_comments_github_id ON task_comments(github_comment_id) WHERE github_comment_id IS NOT NULL;

-- Track which GitHub Projects V2 option ID corresponds to each swim lane
ALTER TABLE swim_lanes ADD COLUMN github_option_id TEXT;

-- Track the Projects V2 item ID per task (needed to push status updates)
ALTER TABLE tasks ADD COLUMN github_project_item_id TEXT;
