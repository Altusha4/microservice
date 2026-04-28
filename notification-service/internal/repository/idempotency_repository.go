package repository

import "context"

// IdempotencyRepository keeps track of already-processed event IDs.
// Implementation must be safe for concurrent use.
type IdempotencyRepository interface {
	// MarkProcessed atomically records the event_id.
	// Returns true if it was inserted (first time we see it),
	// false if it already existed (duplicate — caller should skip work).
	MarkProcessed(ctx context.Context, eventID string) (bool, error)
}
