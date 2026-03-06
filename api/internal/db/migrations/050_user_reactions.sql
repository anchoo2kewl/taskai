ALTER TABLE github_reactions ADD COLUMN github_reaction_id INTEGER;

CREATE TABLE IF NOT EXISTS user_reactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    task_comment_id INTEGER REFERENCES task_comments(id) ON DELETE CASCADE,
    reaction TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_reactions_task
    ON user_reactions (user_id, task_id, reaction)
    WHERE task_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_reactions_comment
    ON user_reactions (user_id, task_comment_id, reaction)
    WHERE task_comment_id IS NOT NULL;
