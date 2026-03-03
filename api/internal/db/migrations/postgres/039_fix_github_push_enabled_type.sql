-- Fix github_push_enabled column type from INTEGER to BOOLEAN
-- (migration 038 incorrectly used INTEGER; this corrects existing deployments)
ALTER TABLE projects ALTER COLUMN github_push_enabled DROP DEFAULT;
ALTER TABLE projects ALTER COLUMN github_push_enabled TYPE BOOLEAN USING (github_push_enabled != 0);
ALTER TABLE projects ALTER COLUMN github_push_enabled SET DEFAULT false;
