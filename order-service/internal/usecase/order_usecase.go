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
	OrderID string
	Amount  int64
}

type PaymentResponse struct {
	Status        string
	TransactionID string
}

type PaymentClient interface {
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

// #######################################
// ORDER USECASE
// #######################################

type OrderUseCase struct {
	repo          repository.OrderRepository
	paymentClient PaymentClient
}

func NewOrderUseCase(repo repository.OrderRepository, pc PaymentClient) *OrderUseCase {
	return &OrderUseCase{repo: repo, paymentClient: pc}
}

// ##############################
// CreateOrder
// ##############################

func (uc *OrderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {

	// ####################
	// Validation
	// ####################

	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// ####################
	// Idempotency check
	// ####################

	if idempotencyKey != "" {
		existing, err := uc.repo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("check idempotency key: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	// ####################
	// Create order object
	// ####################

	order := &domain.Order{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	// ####################
	// Persist order
	// ####################

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("persist order: %w", err)
	}

	if idempotencyKey != "" {
		if err := uc.repo.SaveIdempotencyKey(ctx, idempotencyKey, order.ID); err != nil {
			return nil, fmt.Errorf("save idempotency key: %w", err)
		}
	}

	// ####################
	// Process payment
	// ####################

	payResp, err := uc.paymentClient.ProcessPayment(ctx, PaymentRequest{
		OrderID: order.ID,
		Amount:  order.Amount,
	})
	if err != nil {
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		order.Status = domain.StatusFailed
		return order, ErrPaymentServiceUnavailable
	}

	// ####################
	// Status Update
	// ####################

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

// ##############################
// GetOrder
// ##############################

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
	return uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled)
}
