-- Migration: Add password_hash column to users table
-- Date: 2026-02-13
-- Description: Implements password-based authentication for user login

-- Add password_hash column (nullable initially to allow migration)
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

-- For existing users: Set a temporary password hash
-- This hash corresponds to the password "ChangeMe123!" (bcrypt cost 10)
-- Users will need to reset their password on first login or be notified
UPDATE users 
SET password_hash = '$2a$10$N9qo8uLOickgx2ZMRZoMye/IQ8ugYqDJx7nZhELJ.vKjLj5cU7uPu'
WHERE password_hash IS NULL OR password_hash = '';

-- Make password_hash NOT NULL after setting defaults
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;

-- Add index on email for faster lookups (if not exists)
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Add comment explaining the column
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hash of user password (cost=10). Never exposed in API responses.';

-- Notes:
-- 1. All existing users have been set with temporary password: "ChangeMe123!"
-- 2. Consider implementing a password reset flow for production
-- 3. Notify users to change their password on next login
-- 4. For new deployments without existing users, the UPDATE can be skipped
