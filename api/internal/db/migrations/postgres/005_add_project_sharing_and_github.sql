-- Add project sharing and GitHub integration fields

-- Add GitHub fields to projects table
ALTER TABLE projects ADD COLUMN github_repo_url TEXT;
ALTER TABLE projects ADD COLUMN github_owner TEXT;
ALTER TABLE projects ADD COLUMN github_repo_name TEXT;
ALTER TABLE projects ADD COLUMN github_branch TEXT DEFAULT 'main';
ALTER TABLE projects ADD COLUMN github_sync_enabled BOOLEAN DEFAULT false NOT NULL;
ALTER TABLE projects ADD COLUMN github_last_sync TIMESTAMPTZ;

-- Project members table for sharing
CREATE TABLE project_members (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('viewer', 'editor', 'admin')) DEFAULT 'viewer',
    added_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (added_by) REFERENCES users(id),
    UNIQUE(project_id, user_id)
);

CREATE INDEX idx_project_members_project_id ON project_members(project_id);
CREATE INDEX idx_project_members_user_id ON project_members(user_id);

-- Project invitations table
CREATE TABLE project_invitations (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('viewer', 'editor', 'admin')) DEFAULT 'viewer',
    invited_by BIGINT NOT NULL,
    accepted BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by) REFERENCES users(id),
    UNIQUE(project_id, email)
);

CREATE INDEX idx_project_invitations_email ON project_invitations(email);
CREATE INDEX idx_project_invitations_project_id ON project_invitations(project_id);
