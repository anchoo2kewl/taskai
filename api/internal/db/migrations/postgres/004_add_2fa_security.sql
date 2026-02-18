-- Add 2FA and security fields to users table

-- Add TOTP secret for 2FA
ALTER TABLE users ADD COLUMN totp_secret TEXT;

-- Add flag to indicate if 2FA is enabled
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN DEFAULT false NOT NULL;

-- Add backup codes (JSON array of hashed codes)
ALTER TABLE users ADD COLUMN backup_codes TEXT;

-- Add last password change timestamp
ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ;
