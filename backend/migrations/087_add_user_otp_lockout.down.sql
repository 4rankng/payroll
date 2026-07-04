-- Reverse of 087. Drops the email-OTP per-account lockout columns.
ALTER TABLE users DROP COLUMN otp_locked_until;
ALTER TABLE users DROP COLUMN otp_failed_attempts;
