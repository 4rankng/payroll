-- Create the wallet low-balance alert threshold row where it is missing.
--
-- Why
-- ---
-- The row is created by internal/seed/settings.go, which only runs when a
-- database is bootstrapped from scratch. Any environment upgraded from an older
-- release therefore has no row, and the consequences are user-visible:
--
--   * GetWalletBalanceAlertThreshold logs "setting not found" and falls back to
--     the compiled-in default, so the alert silently runs on 50,000,000 VND
--     whatever the admin believes is configured.
--   * The admin settings form marks the card unavailable, because the key is
--     absent from the settings list it renders, so the threshold cannot be
--     edited at all — the exact thing the panel exists for.
--
-- The other seed keys are deliberately NOT created here. They carry
-- placeholder company and bank data (name, address, phone, email, source
-- account), which would surface on payslips and statements as if an admin had
-- entered it. A missing row for those is created on first save, which is the
-- correct time to ask the admin for real values.
--
-- Safe to re-run and non-destructive: the INSERT is skipped whenever the key
-- already exists, so an admin who has already set a threshold keeps it.
--
-- 50000000 matches config.DefaultWalletBalanceAlertThreshold and the seed
-- default, so creating the row changes no behaviour — it only makes the value
-- visible and editable.

INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('wallet_balance_alert_threshold_vnd', '50000000', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;

-- Verify: the admin form can now read and edit the threshold.
SELECT `key`, `value`, `value_type`
FROM settings
WHERE `key` = 'wallet_balance_alert_threshold_vnd';
