package domain

import "time"

// Order statuses
const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

// Order is the core domain entity for an order.
// It has zero dependency on HTTP, JSON, or any framework.
type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64 // cents, e.g. 1000 = $10.00
	Status     string
	CreatedAt  time.Time
}
