-- Soft delete support for users: admin can mark a user as deleted
-- without destroying invite/activity history.
-- Email is anonymized on delete to free it up for re-invite.
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
