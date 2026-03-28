package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Altusha4/microservice/payment-service/internal/domain"
	"github.com/Altusha4/microservice/payment-service/internal/repository"
	"github.com/google/uuid"
)

// ErrPaymentNotFound is returned when a payment record does not exist.
var ErrPaymentNotFound = errors.New("payment not found")

// ErrInvalidAmount is returned when the payment amount is invalid.
var ErrInvalidAmount = errors.New("amount must be greater than 0")

// maxPaymentAmount is the business rule threshold: amounts above this are Declined.
const maxPaymentAmount int64 = 100000

// PaymentUseCase contains all business logic for payments.
type PaymentUseCase struct {
	repo repository.PaymentRepository
}

// NewPaymentUseCase creates a new PaymentUseCase with its required dependencies.
func NewPaymentUseCase(repo repository.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

// ProcessPayment applies business rules and persists the payment result.
// Amounts > 100000 cents are Declined; otherwise Authorized.
func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	status := domain.StatusAuthorized
	transactionID := uuid.New().String()

	if amount > maxPaymentAmount {
		status = domain.StatusDeclined
		transactionID = "" // no transaction for declined payments
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        status,
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("persist payment: %w", err)
	}

	return payment, nil
}

// GetPaymentByOrderID retrieves a payment record by its associated order ID.
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
