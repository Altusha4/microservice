package domain

// Payment statuses
const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"
)

// Payment is the core domain entity for a payment.
// It has zero dependency on HTTP, JSON, or any framework.
type Payment struct {
	ID            string
	OrderID       string
	TransactionID string // unique UUID per successful transaction
	Amount        int64  // cents
	Status        string
}
