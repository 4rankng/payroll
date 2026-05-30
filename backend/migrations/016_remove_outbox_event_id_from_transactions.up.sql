-- Remove the obsolete outbox_event_id column from transactions
ALTER TABLE `transactions`
  DROP COLUMN `outbox_event_id`;
