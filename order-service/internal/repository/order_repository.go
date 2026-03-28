package repository

import (
	"context"

	"github.com/Altusha4/microservice/order-service/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status string) error
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	SaveIdempotencyKey(ctx context.Context, key, orderID string) error
}
