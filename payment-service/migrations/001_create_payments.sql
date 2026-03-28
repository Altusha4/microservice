-- Migration: 001_create_payments
-- Creates the payments table for the payment-service.

CREATE TABLE IF NOT EXISTS payments (
    id             TEXT   PRIMARY KEY,
    order_id       TEXT   NOT NULL UNIQUE,
    transaction_id TEXT,
    amount         BIGINT NOT NULL CHECK (amount > 0),
    status         TEXT   NOT NULL DEFAULT 'Authorized'
);
