-- 087: per-account OTP brute-force / lockout state for the email-OTP 2FA feature.
-- These two columns back the per-account attempt counter and lockout window
-- enforced by AuthService.VerifyLoginOTP. They live on `users` (not a separate
-- table) because email-OTP stores no per-user secret — the only persistent MFA
-- state is this lockout counter, mirroring the adjacent tokens_invalid_before
-- column added in 086.
--   otp_failed_attempts = consecutive failed verify attempts; reset to 0 on success.
--   otp_locked_until    = when non-null, OTP-gated login refuses until this instant.
ALTER TABLE users
    ADD COLUMN otp_failed_attempts INT NOT NULL DEFAULT 0
        COMMENT 'consecutive failed OTP verifies; reset on success'
        AFTER tokens_invalid_before,
    ADD COLUMN otp_locked_until DATETIME(3) NULL
        COMMENT 'when set, OTP-gated login is refused until this instant'
        AFTER otp_failed_attempts;
