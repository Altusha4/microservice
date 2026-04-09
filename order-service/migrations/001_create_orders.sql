-- #######################################
-- ORDERS
-- #######################################

-- ##############################
-- Orders Table
-- ##############################

CREATE TABLE IF NOT EXISTS orders (
    id          TEXT        PRIMARY KEY,
    customer_id TEXT        NOT NULL,
    item_name   TEXT        NOT NULL,
    amount      BIGINT      NOT NULL CHECK (amount > 0),
    status      TEXT        NOT NULL DEFAULT 'Pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- #######################################
-- IDEMPOTENCY
-- #######################################

-- ##############################
-- Idempotency Keys Table
-- ##############################

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key      TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE
);

