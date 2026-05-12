# Assignment 4 — Changelog

Documents what changed between Assignment 3 and Assignment 4.
Everything from A3 (RabbitMQ topology, DLQ logic, manual ACK, publisher confirms,
Postgres idempotency) is retained unchanged as the foundation.

---

## Added

### Infrastructure

- **Redis 7-alpine container** in `docker-compose.yml` with `appendonly yes`
  persistence and a named volume (`redis_data`). Used by both order-service
  (cache + rate limiter) and notification-service (idempotency cache).
- **Centralized `.env`** at the repo root. Both services load it via
  `env_file: .env` in compose, replacing per-service env blocks.
- **`.env.example`** committed to git with all keys present and secrets blanked —
  safe to share; developers copy it to `.env` and fill in credentials.

---

### order-service

| File | What it does |
|---|---|
| `internal/cache/cache.go` | `OrderCache` interface (`Get`, `Set`, `Delete`) |
| `internal/infrastructure/redis/order_cache.go` | Redis implementation: JSON marshal/unmarshal, TTL configurable via `ORDER_CACHE_TTL_SECONDS` |
| `internal/transport/http/ratelimit.go` | **Bonus (+10%)** — fixed-window rate limiter Gin middleware using Redis `INCR` + `EXPIRE`; 429 + `Retry-After` when exceeded |
| `cmd/order-service/main.go` (updated) | Wires Redis client, `OrderCache`, and `RateLimiter`; applies `r.Use(rateLimiter.Middleware())` globally |

Additional changes inside existing files:

- `usecase/order_usecase.go` — cache-aside logic in `GetOrder`; cache invalidation (`Delete`) on `CancelOrder` and on failed-payment paths.
- `repository/postgres/order_repo.go` — bug fix: `customer_email` was missing from `Create`, `GetByID`, and `GetByIdempotencyKey` queries; added to `SELECT` and `INSERT`.

New env vars read by order-service:

| Variable | Default | Meaning |
|---|---|---|
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `REDIS_PASSWORD` | _(empty)_ | Redis AUTH password |
| `REDIS_DB` | `0` | Redis database index |
| `ORDER_CACHE_TTL_SECONDS` | `300` | Cache TTL in seconds |
| `RATE_LIMIT_PER_MIN` | `10` | Fixed-window rate limit per IP |

---

### notification-service

| File | What it does |
|---|---|
| `internal/cache/cache.go` | `IdempotencyCache` interface (`Claim`, `MarkSent`, `IsSent`) |
| `internal/infrastructure/redis/idempotency_cache.go` | Redis implementation: `SETNX` for atomic claims, `SET` for sent markers; both use `NOTIFICATION_IDEMPOTENCY_TTL_SECONDS` TTL |
| `internal/email/sender.go` | `EmailSender` interface + `Message` struct |
| `internal/email/simulated.go` | `SimulatedSender` — configurable latency + probabilistic failure rate, used to exercise the retry logic |
| `internal/email/smtp.go` | `SMTPSender` — context-aware real SMTP send via goroutine; compatible with Mailtrap sandbox |
| `internal/email/factory.go` | `Build()` reads `PROVIDER_MODE` and returns the correct `EmailSender` |
| `internal/usecase/notification_usecase.go` (updated) | Two-tier idempotency (Redis → Postgres), retry loop with exponential backoff (`sendWithRetry`), `RetryConfig` struct |
| `cmd/notification-service/main.go` (rewritten) | Correct wiring: Postgres + Redis + RabbitMQ + `email.Build()` + `NewNotificationUseCase`; graceful shutdown via `signal.NotifyContext` |

New env vars read by notification-service:

| Variable | Default | Meaning |
|---|---|---|
| `NOTIFICATION_IDEMPOTENCY_TTL_SECONDS` | `86400` | TTL for Redis idempotency keys |
| `EMAIL_MAX_RETRIES` | `3` | Max send attempts including first try |
| `EMAIL_BACKOFF_BASE_MS` | `2000` | Base delay in ms (doubles each retry) |
| `PROVIDER_MODE` | `SIMULATED` | `SIMULATED` or `SMTP` |
| `SIMULATOR_LATENCY_MS` | `200` | Artificial latency for simulated provider |
| `SIMULATOR_FAILURE_RATE` | `0.3` | Failure probability for simulated provider |
| `SMTP_HOST` | `sandbox.smtp.mailtrap.io` | SMTP hostname |
| `SMTP_PORT` | `2525` | SMTP port |
| `SMTP_USERNAME` | _(empty)_ | SMTP credentials |
| `SMTP_PASSWORD` | _(empty)_ | SMTP credentials |
| `SMTP_FROM` | `no-reply@microservice.local` | Sender address |

---

## Did NOT change

- **A3 RabbitMQ work:** exchange/queue topology, DLQ routing, `Nack(requeue=true/false)` decision tree, in-process attempt counter, manual ACK — all retained as-is.
- **Publisher confirms** in payment-service — unchanged.
- **gRPC contract** between Order and Payment — proto files and generated code unchanged.
- **HTTP API surface** — no new endpoints added. The rate limiter applies globally to the existing Gin router; callers see new headers but the same routes.
- **go.mod / go.sum** — no new dependencies. `go-redis/v9 v9.7.0` and `amqp091-go` were already present from A3.
