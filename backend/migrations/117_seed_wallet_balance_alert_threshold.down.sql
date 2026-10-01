-- Remove the seeded wallet alert threshold row.
--
-- Only correct as a rollback of a *fresh* seed: if an admin has since saved a
-- real threshold, deleting the row discards their configuration and the alert
-- silently reverts to the compiled-in default. Check the value before running.
DELETE FROM settings WHERE `key` = 'wallet_balance_alert_threshold_vnd';
