-- Migration: Remove unique constraint from users.email
-- This allows multiple employees to have NULL or duplicate email addresses
-- Needed for employee user initialization where employees might not have email addresses

-- Drop the unique index on email column
ALTER TABLE users DROP INDEX idx_users_email;

-- Create a non-unique index for better query performance
CREATE INDEX idx_users_email_nonunique ON users(email);
