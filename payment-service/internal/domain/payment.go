package domain

const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"
)

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	CustomerEmail string
	Amount        int64
	Status        string
}
