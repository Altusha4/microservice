package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Altusha4/microservice/notification-service/internal/domain"
	"github.com/Altusha4/microservice/notification-service/internal/repository"
)

// #######################################
// USECASE
// #######################################

type NotificationUseCase struct {
	idempotency repository.IdempotencyRepository
}

func NewNotificationUseCase(idempotency repository.IdempotencyRepository) *NotificationUseCase {
	return &NotificationUseCase{idempotency: idempotency}
}

// HandlePaymentCompleted is idempotent and supports the DLQ demo:
// emails matching "fail@..." simulate a transient failure on every attempt
// so that after MaxDeliveryAttempts the message lands in the DLQ.
func (uc *NotificationUseCase) HandlePaymentCompleted(ctx context.Context, event domain.PaymentCompletedEvent) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id is empty - cannot deduplicate")
	}

	// DLQ DEMO HOOK - must run BEFORE the idempotency claim,
	// otherwise the first failed attempt would mark the event
	// as processed and silently drop subsequent retries.
	if isFailureProbeEmail(event.CustomerEmail) {
		return fmt.Errorf("simulated transient failure for %s", event.CustomerEmail)
	}

	first, err := uc.idempotency.MarkProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if !first {
		log.Printf("[Notification] duplicate event %s ignored (order=%s)", event.EventID, event.OrderID)
		return nil
	}

	log.Printf(
		"[Notification] Sent email to %s for Order #%s. Amount: $%.2f",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100.0,
	)
	return nil
}

func isFailureProbeEmail(email string) bool {
	return strings.HasPrefix(strings.ToLower(email), "fail@")
}
