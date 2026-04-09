package domain

import "time"

// #######################################
// ORDER CONSTANTS
// #######################################

const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

// #######################################
// ORDER MODEL
// #######################################

type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64
	Status     string
	CreatedAt  time.Time
}
