package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Altusha4/microservice/notification-service/internal/cache"
	"github.com/Altusha4/microservice/notification-service/internal/domain"
	"github.com/Altusha4/microservice/notification-service/internal/email"
	"github.com/Altusha4/microservice/notification-service/internal/repository"
)

// #######################################
// USECASE
// #######################################
//
// HandlePaymentCompleted now does the full A4 pipeline:
//
//  1. Validate event_id (cannot dedupe without it).
//  2. Demo trigger: fail@... → return error so the DLQ flow stays observable.
//  3. Idempotency claim:
//        Redis first (fast, <1ms),
//        Postgres second (durable, the source of truth).
//  4. Send the email via the EmailSender interface.
//     Retries on transient errors with exponential backoff.
//  5. On success, write a "sent" marker to Redis to make repeated
//     attempts on the same payment_id even cheaper.

type RetryConfig struct {
	MaxAttempts int           // total tries; e.g. 3 = first + 2 retries
	BaseDelay   time.Duration // first backoff; doubles each retry
}

type NotificationUseCase struct {
	idempotencyDB    repository.IdempotencyRepository
	idempotencyCache cache.IdempotencyCache // optional, can be nil
	sender           email.EmailSender
	retry            RetryConfig
}

func NewNotificationUseCase(
	idempotencyDB repository.IdempotencyRepository,
	idempotencyCache cache.IdempotencyCache,
	sender email.EmailSender,
	retry RetryConfig,
) *NotificationUseCase {
	if retry.MaxAttempts < 1 {
		retry.MaxAttempts = 3
	}
	if retry.BaseDelay <= 0 {
		retry.BaseDelay = 2 * time.Second
	}
	return &NotificationUseCase{
		idempotencyDB:    idempotencyDB,
		idempotencyCache: idempotencyCache,
		sender:           sender,
		retry:            retry,
	}
}

// HandlePaymentCompleted processes one event end-to-end.
// Returns nil on success OR on a duplicate (the message must be Ack'd).
// Returns error only when the consumer should requeue / DLQ the message.
func (uc *NotificationUseCase) HandlePaymentCompleted(ctx context.Context, event domain.PaymentCompletedEvent) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id is empty - cannot deduplicate")
	}

	// ##############################
	// DLQ DEMO HOOK
	// ##############################
	// Runs BEFORE any idempotency claim so retries are actually observable.
	if isFailureProbeEmail(event.CustomerEmail) {
		return fmt.Errorf("simulated transient failure for %s", event.CustomerEmail)
	}

	// ##############################
	// Two-tier idempotency
	// ##############################
	// Redis says "already seen" → return nil, message gets Ack'd.
	// Redis says "first time" → also claim it in Postgres (durable).
	// If Redis is unhealthy, we degrade to Postgres-only behaviour.
	if uc.idempotencyCache != nil {
		first, err := uc.idempotencyCache.Claim(ctx, event.EventID)
		if err != nil {
			log.Printf("[notification][cache] claim %s failed: %v - falling back to DB", event.EventID, err)
		} else if !first {
			log.Printf("[notification] duplicate event %s ignored via Redis (order=%s)",
				event.EventID, event.OrderID)
			return nil
		}
	}

	dbFirst, err := uc.idempotencyDB.MarkProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("idempotency DB: %w", err)
	}
	if !dbFirst {
		log.Printf("[notification] duplicate event %s ignored via DB (order=%s)",
			event.EventID, event.OrderID)
		return nil
	}

	// ##############################
	// Send the email with retry + exponential backoff
	// ##############################
	amountUS := fmt.Sprintf("$%.2f", float64(event.Amount)/100.0)
	msg := email.Message{
		To:       event.CustomerEmail,
		OrderID:  event.OrderID,
		AmountUS: amountUS,
	}

	if err := uc.sendWithRetry(ctx, msg); err != nil {
		// All retries exhausted — return error so the consumer
		// can decide (retry queue → DLQ).
		return fmt.Errorf("send email after %d attempts: %w", uc.retry.MaxAttempts, err)
	}

	// ##############################
	// Mark "sent" in Redis to short-circuit any future attempts
	// ##############################
	if uc.idempotencyCache != nil {
		if err := uc.idempotencyCache.MarkSent(ctx, event.OrderID); err != nil {
			log.Printf("[notification][cache] mark-sent %s failed: %v", event.OrderID, err)
		}
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: %s",
		event.CustomerEmail, event.OrderID, amountUS)
	return nil
}

// sendWithRetry attempts uc.retry.MaxAttempts deliveries.
// Backoff doubles each round: 2s → 4s → 8s (for BaseDelay=2s).
func (uc *NotificationUseCase) sendWithRetry(ctx context.Context, msg email.Message) error {
	var lastErr error

	for attempt := 1; attempt <= uc.retry.MaxAttempts; attempt++ {
		if err := uc.sender.SendOrderConfirmation(ctx, msg); err == nil {
			if attempt > 1 {
				log.Printf("[notification][retry] order=%s succeeded on attempt %d/%d",
					msg.OrderID, attempt, uc.retry.MaxAttempts)
			}
			return nil
		} else {
			lastErr = err
		}

		if attempt == uc.retry.MaxAttempts {
			break // budget exhausted
		}

		// Exponential backoff: BaseDelay * 2^(attempt-1)
		// e.g. BaseDelay=2s, attempts=3 → 2s, 4s, 8s
		delay := uc.retry.BaseDelay * time.Duration(1<<uint(attempt-1))
		log.Printf("[notification][retry] order=%s attempt %d/%d failed: %v - sleeping %s",
			msg.OrderID, attempt, uc.retry.MaxAttempts, lastErr, delay)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		}
	}

	return lastErr
}

func isFailureProbeEmail(email string) bool {
	return strings.HasPrefix(strings.ToLower(email), "fail@")
}
