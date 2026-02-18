-- Migration: Backfill project access for existing team members
-- This migration adds all team members to their team's existing projects

-- Add team members to all projects in their team (excluding members who already have access)
INSERT INTO project_members (project_id, user_id, role, granted_by, granted_at)
SELECT DISTINCT p.id, tm.user_id, 'member', p.owner_id, NOW()
FROM projects p
INNER JOIN team_members tm ON p.team_id = tm.team_id
WHERE tm.status = 'active'
  AND NOT EXISTS (
    SELECT 1 FROM project_members pm
    WHERE pm.project_id = p.id AND pm.user_id = tm.user_id
  )
ON CONFLICT (project_id, user_id) DO NOTHING;
