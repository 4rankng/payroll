-- Drop outbox tables; handler coordination and retry are now managed by asynq.
-- outbox_event_handlers must be dropped first (FK references outbox_events).
DROP TABLE IF EXISTS outbox_event_handlers;
DROP TABLE IF EXISTS outbox_events;
