package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Altusha4/microservice/payment-service/internal/domain"
	"github.com/Altusha4/microservice/payment-service/internal/repository"
	"github.com/google/uuid"
)

// #######################################
// ERRORS
// #######################################

var ErrPaymentNotFound = errors.New("payment not found")
var ErrInvalidAmount = errors.New("amount must be greater than 0")

const maxPaymentAmount int64 = 100000

// #######################################
// EVENT TYPES
// #######################################

// PaymentCompletedEvent is the JSON payload published to the broker
// after a successful payment is persisted.
type PaymentCompletedEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	Amount        int64     `json:"amount"`
	CustomerEmail string    `json:"customer_email"`
	Status        string    `json:"status"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// EventPublisher hides the concrete broker (RabbitMQ) from the use case.
// The usecase depends only on this interface — Separation of Concerns.
type EventPublisher interface {
	PublishPaymentCompleted(ctx context.Context, event PaymentCompletedEvent) error
}

// #######################################
// USECASE
// #######################################

type PaymentUseCase struct {
	repo      repository.PaymentRepository
	publisher EventPublisher
}

func NewPaymentUseCase(repo repository.PaymentRepository, publisher EventPublisher) *PaymentUseCase {
	return &PaymentUseCase{repo: repo, publisher: publisher}
}

// ##############################
// ProcessPayment
// ##############################

func (uc *PaymentUseCase) ProcessPayment(
	ctx context.Context,
	orderID string,
	amount int64,
	customerEmail string,
) (*domain.Payment, error) {

	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	status := domain.StatusAuthorized
	transactionID := uuid.New().String()

	if amount > maxPaymentAmount {
		status = domain.StatusDeclined
		transactionID = ""
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: transactionID,
		CustomerEmail: customerEmail,
		Amount:        amount,
		Status:        status,
	}

	// DB transaction commit happens here.
	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("persist payment: %w", err)
	}

	// Publish event ONLY after a successful Authorized payment.
	// We log a warning but do not fail the whole RPC if the broker is unreachable —
	// the payment itself is already committed. Producer-side reliability
	// (publisher confirms) lives in the rabbitmq adapter.
	if status == domain.StatusAuthorized && uc.publisher != nil {
		event := PaymentCompletedEvent{
			EventID:       uuid.New().String(),
			OrderID:       payment.OrderID,
			Amount:        payment.Amount,
			CustomerEmail: payment.CustomerEmail,
			Status:        payment.Status,
			OccurredAt:    time.Now().UTC(),
		}
		if err := uc.publisher.PublishPaymentCompleted(ctx, event); err != nil {
			log.Printf("[payment][warn] publish payment.completed failed: %v", err)
		}
	}

	return payment, nil
}

// ##############################
// GetPaymentByOrderID
// ##############################

func (uc *PaymentUseCase) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	payment, err := uc.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	if payment == nil {
		return nil, ErrPaymentNotFound
	}
	return payment, nil
}
