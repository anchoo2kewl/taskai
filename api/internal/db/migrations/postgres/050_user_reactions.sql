ALTER TABLE github_reactions ADD COLUMN IF NOT EXISTS github_reaction_id BIGINT;

CREATE TABLE IF NOT EXISTS user_reactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
    task_comment_id BIGINT REFERENCES task_comments(id) ON DELETE CASCADE,
    reaction VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_reactions_task
    ON user_reactions (user_id, task_id, reaction)
    WHERE task_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_reactions_comment
    ON user_reactions (user_id, task_comment_id, reaction)
    WHERE task_comment_id IS NOT NULL;
