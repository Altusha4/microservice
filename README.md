# Order & Payment Platform

A two-service microservice platform built with Go, Gin, and PostgreSQL, following Clean Architecture and Domain-Driven Design principles.

Includes a **frontend dashboard** at `http://localhost:13000` for interactive demo during presentations.

---

## Proto & Generated Code Repositories

| Repository | URL |
|---|---|
| **Proto definitions** | https://github.com/Altusha4/ap2-protos |
| **Generated Go code** | https://github.com/Altusha4/ap2-generated |

---

## What Changed in Assignment 2

- **Order → Payment via gRPC**: Order Service now calls Payment Service over gRPC instead of REST.
- **Payment Service gRPC server**: Payment Service runs a gRPC server on port `50051` alongside its HTTP server on port `8081`.
- **Order Service gRPC streaming server**: Order Service runs a gRPC streaming server on port `50052` for real-time order status updates.
- **Contract-First approach**: `.proto` files live in the [ap2-protos](https://github.com/Altusha4/ap2-protos) repo; auto-generated `.pb.go` files are pushed to the [ap2-generated](https://github.com/Altusha4/ap2-generated) repo via GitHub Actions.
- **gRPC Logging Interceptor** *(bonus)*: A server-side unary interceptor on Payment Service logs every incoming RPC with the method name and duration.

---

## Getting Started

### Prerequisites
- Docker & Docker Compose

### Run everything (all services + dashboard)

```bash
docker compose up --build
```

| Service | URL |
|---|---|
| **Frontend Dashboard** | http://localhost:13000 |
| Order Service API | http://localhost:18080 |
| Payment Service API | http://localhost:18081 |
| Order Service gRPC | localhost:50052 |
| Payment Service gRPC | localhost:50051 |
| order_db (PostgreSQL) | localhost:5433 |
| payment_db (PostgreSQL) | localhost:5434 |

---

## Frontend Dashboard

The dashboard is a single-page app served by nginx that proxies all API calls to avoid CORS issues.

### Sections

| Section | Description |
|---|---|
| **Quick Demo** | One-click buttons that prefill the form with test scenarios |
| **Create Order** | Form with Customer ID, Item Name, Amount (cents), optional Idempotency-Key |
| **Order Actions** | Get order by ID; cancel order (demonstrates 409 for paid orders) |
| **Payment Lookup** | Retrieve payment status by order ID |
| **Activity Log** | Real-time log of every request — method, endpoint, status code, collapsible body |

### Quick Demo Scenarios

| Button | Amount | Expected Result |
|---|---|---|
| Normal Order ($150) | 15 000 | Status: **Paid** |
| Over Limit ($1 500) | 150 000 | Payment **Declined** → Order **Failed** |
| Tiny Order ($1) | 100 | Status: **Paid** |
| Zero Amount | 0 | **400** Bad Request |
| Test Idempotency | 2 500 | Run twice — same order returned, no duplicate |

### Status Color Coding

| Status | Color |
|---|---|
| Paid / Authorized | Green |
| Failed / Declined | Red |
| Pending | Yellow |
| Cancelled | Gray |
| 503 / Network Error | Red border |

---

## Architecture Decisions

### Clean Architecture (per service)

Each service is structured in strict layers with a single direction of dependency:

```
Transport (HTTP) → Use Case → Domain
                      ↑
                 Repository (interface)
                      ↑
               Postgres (implementation)
```

- **Domain**: Pure Go structs, zero framework dependencies.
- **Repository (port)**: Interface defined in the `repository` package. Use cases depend only on the interface.
- **Use Case**: All business logic lives here. Depends on repository interfaces and external client interfaces — never on concrete implementations.
- **Transport**: Thin Gin handlers. Only parse requests, call use cases, return responses.
- **Composition Root** (`main.go`): The only place where concrete types are instantiated and wired together (manual DI).

### Money Representation

All monetary amounts are stored and transmitted as `int64` (cents). `float64` is **never** used for money to avoid floating-point precision errors.

### No Shared Code

Each service has its own domain models, interfaces, and utilities. There is no shared/common package. This enforces bounded-context isolation and allows services to evolve independently.

---

## Bounded Contexts

| Context | Responsibility | DB |
|---|---|---|
| **Order** | Lifecycle of a customer purchase: creation, payment orchestration, cancellation | `order_db` (port 5433) |
| **Payment** | Authorization of a payment transaction for a given order | `payment_db` (port 5434) |

The Order service **orchestrates** the flow: it creates the order, calls the Payment service synchronously via gRPC, and updates the order status based on the result. The Payment service is stateless with respect to orders — it only decides Authorized/Declined.

---

## Inter-Service Communication

- **Protocol**: gRPC (synchronous unary RPC)
- **Order → Payment**: `PaymentService.ProcessPayment` RPC with `order_id` and `amount`
- **gRPC Address**: Configured via the `PAYMENT_GRPC_ADDR` environment variable (e.g. `payment-service:50051`)
- **Failure Handling**: If the Payment service is unreachable or the RPC fails, the Order service marks the order as `"Failed"` and returns `503 Service Unavailable` to the caller.

---

## Test Real-time Streaming

Order Service exposes a gRPC server-side streaming endpoint that pushes order status changes in real time.

**Terminal 1 — connect the stream client:**
```bash
cd order-service && go run cmd/stream-client/main.go <order-id>
```

**Terminal 2 — trigger a status change:**
```bash
docker exec -it order_db psql -U postgres order_db -c \
  "UPDATE orders SET status='Cancelled' WHERE id='<order-id>'"
```

The stream client in Terminal 1 will print the new status as soon as the database row is updated.

---

## Failure Handling

| Failure Scenario | Behavior |
|---|---|
| Payment service down | Order marked `Failed`, HTTP 503 returned |
| Payment service returns Declined | Order marked `Failed`, HTTP 201 returned with status |
| Order not found | HTTP 404 |
| Cancel a Paid order | HTTP 409 Conflict |
| Amount ≤ 0 | HTTP 400 Bad Request |
| Amount > 100000 cents | Payment `Declined`, Order `Failed` |
| Duplicate request (idempotency key) | Same order returned, no duplicate created |

---

## Architecture Diagram

```mermaid
graph TD
    Browser([Browser<br/>localhost:13000])

    subgraph frontend
        Nginx[nginx<br/>static files + proxy]
    end

    subgraph order-service
        OH[HTTP Handler<br/>Gin :18080]
        OUC[Order UseCase]
        OR[(OrderRepository<br/>interface)]
        ODB[(order_db<br/>PostgreSQL :5433)]
        GS[gRPC Streaming Server<br/>:50052]
        GPC[gRPC Payment Client]
    end

    subgraph payment-service
        PH[HTTP Handler<br/>Gin :18081]
        PUC[Payment UseCase]
        PR[(PaymentRepository<br/>interface)]
        PDB[(payment_db<br/>PostgreSQL :5434)]
        PG[gRPC Server<br/>:50051]
        LI[Logging Interceptor]
    end

    Browser -->|/api/orders<br/>/api/payments| Nginx
    Nginx -->|proxy /api/orders → :18080/orders| OH
    Nginx -->|proxy /api/payments → :18081/payments| PH
    OH --> OUC
    OUC --> OR
    OR --> ODB
    OUC -->|ProcessPayment RPC| GPC
    GPC -->|gRPC :50051| LI
    LI --> PG
    PG --> PUC
    PUC --> PR
    PR --> PDB
    GS -.->|stream status updates| Browser
```

---


## API Reference

### Order Service (`localhost:18080`)

#### Create Order

```bash
curl -X POST http://localhost:18080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-request-id-1" \
  -d '{"customer_id": "cust-123", "item_name": "Laptop", "amount": 15000}'
```

**Response (201):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust-123",
  "item_name": "Laptop",
  "amount": 15000,
  "status": "Paid",
  "created_at": "2026-03-28T12:00:00Z"
}
```

#### Get Order

```bash
curl http://localhost:18080/orders/550e8400-e29b-41d4-a716-446655440000
```

#### Cancel Order

```bash
curl -X PATCH http://localhost:18080/orders/550e8400-e29b-41d4-a716-446655440000/cancel
```

**Error (409) — paid order:**
```json
{"error": "paid orders cannot be cancelled"}
```

#### Test Payment Decline (amount > 100000)

```bash
curl -X POST http://localhost:18080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "cust-456", "item_name": "Yacht", "amount": 999999}'
```

**Response (201):**
```json
{
  "id": "...",
  "status": "Failed",
  ...
}
```

---

### Payment Service (`localhost:18081`)

#### Process Payment

```bash
curl -X POST http://localhost:18081/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 15000}'
```

**Response (201 — Authorized):**
```json
{
  "id": "...",
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "transaction_id": "a1b2c3d4-...",
  "amount": 15000,
  "status": "Authorized"
}
```

#### Get Payment by Order ID

```bash
curl http://localhost:18081/payments/550e8400-e29b-41d4-a716-446655440000
```

---

## Idempotency

`POST /orders` supports the `Idempotency-Key` header. If a request is retried with the same key, the original order is returned without creating a duplicate.

```bash
# First call — creates order
curl -X POST http://localhost:18080/orders \
  -H "Idempotency-Key: req-abc-123" \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "cust-1", "item_name": "Book", "amount": 2500}'

# Retry — returns the same order, no duplicate
curl -X POST http://localhost:18080/orders \
  -H "Idempotency-Key: req-abc-123" \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "cust-1", "item_name": "Book", "amount": 2500}'
```

---

## Order Status Flow

```
Pending → Paid       (payment authorized)
Pending → Failed     (payment declined or payment service unavailable)
Pending → Cancelled  (explicit cancel)
Paid    → ✗          (cannot cancel)
```
