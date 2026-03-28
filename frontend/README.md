# Frontend Dashboard

A single-page dashboard that demonstrates all Order & Payment platform functionality. Served by nginx on port 3000, with built-in API proxying to eliminate CORS issues.

## Running

```bash
docker compose up --build
```

Open http://localhost:3000

## API Proxying

nginx proxies all API calls so the browser never hits services directly:

| Frontend path | Proxied to |
|---|---|
| `POST /api/orders` | `order-service:8080/orders` |
| `GET /api/orders/:id` | `order-service:8080/orders/:id` |
| `PATCH /api/orders/:id/cancel` | `order-service:8080/orders/:id/cancel` |
| `POST /api/payments` | `payment-service:8081/payments` |
| `GET /api/payments/:order_id` | `payment-service:8081/payments/:order_id` |

## Features

- **Create Order** — form with Customer ID, Item Name, Amount, optional Idempotency-Key
- **Order Actions** — get by ID, cancel (shows 409 for paid orders)
- **Payment Lookup** — retrieve payment status by order ID
- **Activity Log** — real-time scrollable log of every request with collapsible response bodies, color-coded by HTTP status
- **Quick Demo Scenarios** — one-click form prefill for common test cases
