package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Altusha4/microservice/payment-service/internal/domain"
	"github.com/Altusha4/microservice/payment-service/internal/repository"
	"github.com/google/uuid"
)

var ErrPaymentNotFound = errors.New("payment not found")
var ErrInvalidAmount = errors.New("amount must be greater than 0")

const maxPaymentAmount int64 = 100000

type PaymentUseCase struct {
	repo repository.PaymentRepository
}

func NewPaymentUseCase(repo repository.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
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
		Amount:        amount,
		Status:        status,
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("persist payment: %w", err)
	}

	return payment, nil
}

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
