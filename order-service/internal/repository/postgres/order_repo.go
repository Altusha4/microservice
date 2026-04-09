package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Altusha4/microservice/order-service/internal/domain"
)

// #######################################
// ORDER REPOSITORY
// #######################################

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

// ##############################
// Create
// ##############################

func (r *OrderRepo) Create(ctx context.Context, order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, item_name, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query,
		order.ID, order.CustomerID, order.ItemName,
		order.Amount, order.Status, order.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

// ##############################
// GetByID
// ##############################

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, created_at
		FROM orders WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var o domain.Order
	var createdAt time.Time
	err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	o.CreatedAt = createdAt
	return &o, nil
}

// ##############################
// UpdateStatus
// ##############################

func (r *OrderRepo) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update order status rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("order %s not found", id)
	}
	return nil
}

// ##############################
// GetByIdempotencyKey
// ##############################

func (r *OrderRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	query := `
		SELECT o.id, o.customer_id, o.item_name, o.amount, o.status, o.created_at
		FROM orders o
		JOIN idempotency_keys ik ON ik.order_id = o.id
		WHERE ik.key = $1`
	row := r.db.QueryRowContext(ctx, query, key)

	var o domain.Order
	var createdAt time.Time
	err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get order by idempotency key: %w", err)
	}
	o.CreatedAt = createdAt
	return &o, nil
}

// ##############################
// SaveIdempotencyKey
// ##############################

func (r *OrderRepo) SaveIdempotencyKey(ctx context.Context, key, orderID string) error {
	query := `INSERT INTO idempotency_keys (key, order_id) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, key, orderID)
	if err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}
	return nil
}
