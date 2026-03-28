package domain

import "time"

const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64
	Status     string
	CreatedAt  time.Time
}
