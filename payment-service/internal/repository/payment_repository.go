package repository

import (
	"context"

	"github.com/Altusha4/microservice/payment-service/internal/domain"
)

// PaymentRepository defines the port (interface) for payment persistence.
// Use cases depend on this interface, not on any concrete implementation.
type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}
