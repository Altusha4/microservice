package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// #######################################
// IDEMPOTENCY REPO
// #######################################

type IdempotencyRepo struct {
	db *sql.DB
}

func NewIdempotencyRepo(db *sql.DB) *IdempotencyRepo {
	return &IdempotencyRepo{db: db}
}

// MarkProcessed inserts the event_id. ON CONFLICT DO NOTHING ensures
// that a duplicate insert is a no-op rather than an error.
// We return whether a new row was actually inserted.
func (r *IdempotencyRepo) MarkProcessed(ctx context.Context, eventID string) (bool, error) {
	const query = `
		INSERT INTO processed_events (event_id)
		VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING`

	res, err := r.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return false, fmt.Errorf("mark processed: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return rows == 1, nil
}
