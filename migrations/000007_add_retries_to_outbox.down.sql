ALTER TABLE outbox_events DROP COLUMN IF EXISTS retries;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS error_reason;
