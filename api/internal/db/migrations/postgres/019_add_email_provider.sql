-- Email provider configuration (singleton table, admin-only)
CREATE TABLE IF NOT EXISTS email_provider (
    id BIGSERIAL PRIMARY KEY,
    provider TEXT NOT NULL DEFAULT 'brevo',
    api_key TEXT NOT NULL,
    sender_email TEXT NOT NULL,
    sender_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
    last_checked_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
