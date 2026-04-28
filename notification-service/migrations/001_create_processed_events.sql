-- #######################################
-- IDEMPOTENCY TABLE
-- #######################################
-- Stores event_id of every successfully processed message.
-- Insert with ON CONFLICT DO NOTHING gives us idempotent
-- consumption — duplicate deliveries become no-ops.

CREATE TABLE IF NOT EXISTS processed_events (
    event_id     TEXT        PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);