package domain

import "time"

// PaymentCompletedEvent mirrors the JSON payload produced by payment-service.
// Kept as an independent struct here — the consumer must NOT import payment's
// types. Decoupling is enforced at compile time.
type PaymentCompletedEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	Amount        int64     `json:"amount"`
	CustomerEmail string    `json:"customer_email"`
	Status        string    `json:"status"`
	OccurredAt    time.Time `json:"occurred_at"`
}
