package cache

import "context"

// #######################################
// IDEMPOTENCY CACHE (Assignment 4)
// #######################################
//
// Sits in FRONT of the Postgres processed_events table.
// Two-tier idempotency:
//   1. Redis  — fast, checked first. Auto-expires after TTL.
//   2. Postgres — durable source of truth. Survives Redis restarts.
//
// On a duplicate, Redis answers in <1ms — no DB hit at all.

type IdempotencyCache interface {
	// Claim atomically marks event_id as processed in the cache.
	// Returns true if it was the first time, false if duplicate.
	// Errors are surfaced — caller decides if it falls back to DB.
	Claim(ctx context.Context, eventID string) (bool, error)

	// MarkSent records that the email for this payment has been
	// successfully delivered. Used to short-circuit handler-level
	// retries within the same consumer instance.
	MarkSent(ctx context.Context, paymentID string) error

	// IsSent reports whether MarkSent was previously called for paymentID.
	IsSent(ctx context.Context, paymentID string) (bool, error)
}
