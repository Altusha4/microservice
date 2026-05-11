package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Altusha4/microservice/order-service/internal/domain"
	"github.com/Altusha4/microservice/order-service/internal/repository"
	"github.com/google/uuid"
)

// #######################################
// ERRORS
// #######################################

var ErrOrderNotFound = errors.New("order not found")
var ErrCannotCancelPaidOrder = errors.New("paid orders cannot be cancelled")
var ErrInvalidAmount = errors.New("amount must be greater than 0")
var ErrPaymentServiceUnavailable = errors.New("payment service unavailable")

// #######################################
// PAYMENT TYPES
// #######################################

type PaymentRequest struct {
	OrderID       string
	Amount        int64
	CustomerEmail string
}

type PaymentResponse struct {
	Status        string
	TransactionID string
}

type PaymentClient interface {
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

// #######################################
// CACHE INTERFACE
// #######################################

// OrderCache abstracts the read-through cache for orders.
// Implementations live in internal/infrastructure/redis.
// The usecase depends only on this interface — Clean Architecture.
type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error) // returns (nil, nil) on miss
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}

// #######################################
// ORDER USECASE
// #######################################

type OrderUseCase struct {
	repo          repository.OrderRepository
	paymentClient PaymentClient
	cache         OrderCache // optional; can be nil
}

// NewOrderUseCase wires repo + payment client + cache.
// Pass nil for cache to disable caching (useful in tests).
func NewOrderUseCase(repo repository.OrderRepository, pc PaymentClient, cache OrderCache) *OrderUseCase {
	return &OrderUseCase{repo: repo, paymentClient: pc, cache: cache}
}

// invalidate removes the order from cache. Errors are logged but
// not returned — a cache failure must NEVER break the request path.
func (uc *OrderUseCase) invalidate(ctx context.Context, id string) {
	if uc.cache == nil {
		return
	}
	if err := uc.cache.Delete(ctx, id); err != nil {
		log.Printf("[order][cache] invalidate %s failed: %v", id, err)
	}
}

// ##############################
// CreateOrder
// ##############################

func (uc *OrderUseCase) CreateOrder(
	ctx context.Context,
	customerID, customerEmail, itemName string,
	amount int64,
	idempotencyKey string,
) (*domain.Order, error) {

	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Idempotency check
	if idempotencyKey != "" {
		existing, err := uc.repo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("check idempotency key: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	order := &domain.Order{
		ID:            uuid.New().String(),
		CustomerID:    customerID,
		CustomerEmail: customerEmail,
		ItemName:      itemName,
		Amount:        amount,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("persist order: %w", err)
	}

	if idempotencyKey != "" {
		if err := uc.repo.SaveIdempotencyKey(ctx, idempotencyKey, order.ID); err != nil {
			return nil, fmt.Errorf("save idempotency key: %w", err)
		}
	}

	// Process payment (synchronous gRPC).
	payResp, err := uc.paymentClient.ProcessPayment(ctx, PaymentRequest{
		OrderID:       order.ID,
		Amount:        order.Amount,
		CustomerEmail: order.CustomerEmail,
	})
	if err != nil {
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		order.Status = domain.StatusFailed
		// Invalidate just in case (status changed to Failed).
		uc.invalidate(ctx, order.ID)
		return order, ErrPaymentServiceUnavailable
	}

	newStatus := domain.StatusPaid
	if payResp.Status == "Declined" {
		newStatus = domain.StatusFailed
	}

	if err := uc.repo.UpdateStatus(ctx, order.ID, newStatus); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}
	order.Status = newStatus

	// We DO NOT populate the cache here on purpose:
	//   the first GetOrder request will warm it up.
	// This avoids caching transient states like "just created, about to update".

	return order, nil
}

// ##############################
// GetOrder — cache-aside read path
// ##############################

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	// 1) Try cache first.
	if uc.cache != nil {
		cached, err := uc.cache.Get(ctx, id)
		if err != nil {
			// Treat cache errors as a miss — never break the request.
			log.Printf("[order][cache] get %s failed: %v", id, err)
		} else if cached != nil {
			return cached, nil
		}
	}

	// 2) Fall back to the database.
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	// 3) Populate the cache for next time.
	if uc.cache != nil {
		if err := uc.cache.Set(ctx, order); err != nil {
			log.Printf("[order][cache] set %s failed: %v", id, err)
		}
	}
	return order, nil
}

// ##############################
// CancelOrder
// ##############################

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) error {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if order == nil {
		return ErrOrderNotFound
	}
	if order.Status == domain.StatusPaid {
		return ErrCannotCancelPaidOrder
	}
	if order.Status == domain.StatusCancelled {
		return nil
	}

	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled); err != nil {
		return err
	}

	// Status changed — drop the cache entry.
	uc.invalidate(ctx, id)
	return nil
}
