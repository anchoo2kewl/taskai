-- Add auth_provider column to distinguish password vs OAuth users.
-- Default 'password' keeps all existing rows valid.
ALTER TABLE users ADD COLUMN auth_provider TEXT NOT NULL DEFAULT 'password';

-- Link table between users and OAuth providers.
CREATE TABLE IF NOT EXISTS oauth_providers (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id),
    UNIQUE(user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_oauth_providers_user_id ON oauth_providers(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_lookup  ON oauth_providers(provider, provider_user_id);
