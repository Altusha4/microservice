package repository

import (
	"context"

	"github.com/Altusha4/microservice/order-service/internal/domain"
)

// OrderRepository defines the port (interface) for order persistence.
// Use cases depend on this interface, not on any concrete implementation.
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status string) error
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	SaveIdempotencyKey(ctx context.Context, key, orderID string) error
}
