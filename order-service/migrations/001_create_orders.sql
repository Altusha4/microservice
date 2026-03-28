-- Migration: 001_create_orders
-- Creates the orders table and idempotency_keys table for the order-service.

CREATE TABLE IF NOT EXISTS orders (
    id          TEXT        PRIMARY KEY,
    customer_id TEXT        NOT NULL,
    item_name   TEXT        NOT NULL,
    amount      BIGINT      NOT NULL CHECK (amount > 0),
    status      TEXT        NOT NULL DEFAULT 'Pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Stores idempotency keys to prevent duplicate order creation.
-- key is the client-supplied Idempotency-Key header value.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key      TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE
);
