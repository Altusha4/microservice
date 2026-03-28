package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Altusha4/microservice/order-service/internal/domain"
	"github.com/Altusha4/microservice/order-service/internal/repository"
	"github.com/google/uuid"
)

// ErrOrderNotFound is returned when an order does not exist.
var ErrOrderNotFound = errors.New("order not found")

// ErrCannotCancelPaidOrder is returned when trying to cancel a paid order.
var ErrCannotCancelPaidOrder = errors.New("paid orders cannot be cancelled")

// ErrInvalidAmount is returned when the order amount is invalid.
var ErrInvalidAmount = errors.New("amount must be greater than 0")

// ErrPaymentServiceUnavailable is returned when the payment service cannot be reached.
var ErrPaymentServiceUnavailable = errors.New("payment service unavailable")

// PaymentRequest is what the order use case sends to the payment service.
type PaymentRequest struct {
	OrderID string
	Amount  int64
}

// PaymentResponse is what the order use case receives from the payment service.
type PaymentResponse struct {
	Status        string
	TransactionID string
}

// PaymentClient is the port (interface) that the order use case depends on
// to communicate with the payment service. The concrete implementation lives
// in the transport layer (HTTP client), keeping the use case framework-free.
type PaymentClient interface {
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

// OrderUseCase contains all business logic for orders.
type OrderUseCase struct {
	repo          repository.OrderRepository
	paymentClient PaymentClient
}

// NewOrderUseCase creates a new OrderUseCase with its required dependencies.
func NewOrderUseCase(repo repository.OrderRepository, pc PaymentClient) *OrderUseCase {
	return &OrderUseCase{repo: repo, paymentClient: pc}
}

// CreateOrder creates a new order in Pending state, calls the payment service,
// and updates the order to Paid or Failed accordingly.
func (uc *OrderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Idempotency: return existing order for the same key
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
		ID:         uuid.New().String(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("persist order: %w", err)
	}

	// Save idempotency key before calling payment service
	if idempotencyKey != "" {
		if err := uc.repo.SaveIdempotencyKey(ctx, idempotencyKey, order.ID); err != nil {
			return nil, fmt.Errorf("save idempotency key: %w", err)
		}
	}

	// Call payment service
	payResp, err := uc.paymentClient.ProcessPayment(ctx, PaymentRequest{
		OrderID: order.ID,
		Amount:  order.Amount,
	})
	if err != nil {
		// Payment service unavailable — mark order as Failed
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		order.Status = domain.StatusFailed
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

	return order, nil
}

// GetOrder retrieves an order by its ID.
func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// CancelOrder cancels a Pending order. Paid orders cannot be cancelled.
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
		return nil // already cancelled, idempotent
	}
	return uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled)
}
