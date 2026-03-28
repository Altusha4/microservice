package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Altusha4/microservice/payment-service/internal/domain"
)

// PaymentRepo is the PostgreSQL implementation of repository.PaymentRepository.
type PaymentRepo struct {
	db *sql.DB
}

// NewPaymentRepo creates a new PostgreSQL payment repository.
func NewPaymentRepo(db *sql.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, transaction_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query,
		payment.ID, payment.OrderID, payment.TransactionID,
		payment.Amount, payment.Status,
	)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *PaymentRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status
		FROM payments WHERE order_id = $1`
	row := r.db.QueryRowContext(ctx, query, orderID)

	var p domain.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by order id: %w", err)
	}
	return &p, nil
}
