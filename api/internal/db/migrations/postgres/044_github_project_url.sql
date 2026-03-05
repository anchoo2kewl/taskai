-- Optional GitHub Projects V2 URL for precise project detection during sync.
-- Format: https://github.com/orgs/{org}/projects/{number}
--      or https://github.com/users/{user}/projects/{number}
ALTER TABLE projects ADD COLUMN IF NOT EXISTS github_project_url TEXT;
