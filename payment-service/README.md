# Payment Service

Processes payment authorizations for orders. Applies business rules (amount threshold) and persists payment records.

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/payments` | Process a payment for an order |
| `GET` | `/payments/:order_id` | Get payment status by order ID |

## Business Rules

- Amount `> 100000` cents (> $1000.00) → `Declined`
- Amount `≤ 100000` cents → `Authorized` with a unique `transaction_id`

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PAYMENT_DB_DSN` | `postgres://postgres:postgres@localhost:5434/payment_db?sslmode=disable` | PostgreSQL DSN |
| `PORT` | `8081` | HTTP listen port |

## Running locally

```bash
go run ./cmd/payment-service
```
