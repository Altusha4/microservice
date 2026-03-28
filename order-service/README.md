# Order Service

Manages the full lifecycle of customer orders. Creates orders as `Pending`, orchestrates payment via the Payment Service, and updates order status to `Paid` or `Failed`.

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/orders` | Create a new order and process payment |
| `GET` | `/orders/:id` | Get order by ID |
| `PATCH` | `/orders/:id/cancel` | Cancel a Pending order |

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ORDER_DB_DSN` | `postgres://postgres:postgres@localhost:5433/order_db?sslmode=disable` | PostgreSQL DSN |
| `PAYMENT_SERVICE_URL` | `http://localhost:8081` | Base URL of the Payment Service |
| `PORT` | `8080` | HTTP listen port |

## Running locally

```bash
go run ./cmd/order-service
```
