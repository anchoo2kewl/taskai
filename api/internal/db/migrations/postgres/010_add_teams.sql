-- Add teams table
CREATE TABLE teams (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    owner_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_teams_owner_id ON teams(owner_id);

-- Add team_members table (many-to-many relationship with status)
CREATE TABLE team_members (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member', -- 'owner', 'admin', 'member'
    status TEXT NOT NULL DEFAULT 'active', -- 'active', 'inactive'
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(team_id, user_id)
);

CREATE INDEX idx_team_members_team_id ON team_members(team_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);

-- Add team_invitations table
CREATE TABLE team_invitations (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL,
    inviter_id BIGINT NOT NULL,
    invitee_email TEXT NOT NULL,
    invitee_id BIGINT, -- NULL until user exists
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'accepted', 'rejected', 'cancelled'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    responded_at TIMESTAMPTZ,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (invitee_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_team_invitations_team_id ON team_invitations(team_id);
CREATE INDEX idx_team_invitations_invitee_email ON team_invitations(invitee_email);
CREATE INDEX idx_team_invitations_invitee_id ON team_invitations(invitee_id);
CREATE INDEX idx_team_invitations_status ON team_invitations(status);

-- Add team_id to existing tables
ALTER TABLE projects ADD COLUMN team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE sprints ADD COLUMN team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE tags ADD COLUMN team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE;

-- Create indices for team_id columns
CREATE INDEX idx_projects_team_id ON projects(team_id);
CREATE INDEX idx_sprints_team_id ON sprints(team_id);
CREATE INDEX idx_tags_team_id ON tags(team_id);

-- Migrate existing data: Create a team for each user and assign their resources
-- Step 1: Create teams for all existing users
INSERT INTO teams (name, owner_id, created_at)
SELECT
    email || '''s Team' as name,
    id as owner_id,
    created_at
FROM users;

-- Step 2: Add each user to their own team as owner
INSERT INTO team_members (team_id, user_id, role, status, joined_at)
SELECT
    t.id as team_id,
    u.id as user_id,
    'owner' as role,
    'active' as status,
    u.created_at as joined_at
FROM users u
JOIN teams t ON t.owner_id = u.id;

-- Step 3: Update existing projects to belong to the owner's team
UPDATE projects
SET team_id = (
    SELECT t.id
    FROM teams t
    WHERE t.owner_id = projects.owner_id
    LIMIT 1
)
WHERE team_id IS NULL;

-- Step 4: Update existing sprints to belong to the user's team
UPDATE sprints
SET team_id = (
    SELECT t.id
    FROM teams t
    WHERE t.owner_id = sprints.user_id
    LIMIT 1
)
WHERE team_id IS NULL;

-- Step 5: Update existing tags to belong to the user's team
UPDATE tags
SET team_id = (
    SELECT t.id
    FROM teams t
    WHERE t.owner_id = tags.user_id
    LIMIT 1
)
WHERE team_id IS NULL;

-- Add trigger to update teams.updated_at
CREATE TRIGGER update_teams_timestamp
BEFORE UPDATE ON teams
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
