-- GitHub sync tracking: token on projects, issue/milestone/label tracking on tasks/sprints/tags

ALTER TABLE projects ADD COLUMN IF NOT EXISTS github_token TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS github_issue_number INTEGER;
ALTER TABLE sprints ADD COLUMN IF NOT EXISTS github_milestone_number INTEGER;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS github_label_name TEXT;

-- Unique indexes prevent duplicate imports (NULLs are treated as distinct so regular tasks are unaffected)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_project_github_issue ON tasks(project_id, github_issue_number);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sprints_project_github_milestone ON sprints(project_id, github_milestone_number);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_project_github_label ON tags(project_id, github_label_name);
