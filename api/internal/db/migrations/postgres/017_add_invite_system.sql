-- Add invite-only registration system
-- Each user gets 3 invites, admins get unlimited (enforced in code)

ALTER TABLE users ADD COLUMN invite_count INTEGER NOT NULL DEFAULT 3;

CREATE TABLE invites (
    id BIGSERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    inviter_id BIGINT NOT NULL REFERENCES users(id),
    invitee_id BIGINT REFERENCES users(id),
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invites_code ON invites(code);
CREATE INDEX idx_invites_inviter_id ON invites(inviter_id);
