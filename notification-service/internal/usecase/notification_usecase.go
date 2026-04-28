package usecase

import (
	"context"
	"fmt"
	"log"

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

// HandlePaymentCompleted is the business logic of the consumer.
// It is idempotent: if the same event_id arrives twice, the email
// is "sent" only once.
//
// Flow:
//  1. Try to claim the event_id in the DB (atomic insert).
//  2. If the row was inserted — first time, do the work.
//  3. If the row already existed — duplicate, skip silently.
func (uc *NotificationUseCase) HandlePaymentCompleted(ctx context.Context, event domain.PaymentCompletedEvent) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id is empty — cannot deduplicate")
	}

	first, err := uc.idempotency.MarkProcessed(ctx, event.EventID)
	if err != nil {
		// DB is down or transient — return error so the consumer
		// nacks-with-requeue and we retry later.
		return fmt.Errorf("idempotency check: %w", err)
	}

	if !first {
		log.Printf("[Notification] duplicate event %s ignored (order=%s)", event.EventID, event.OrderID)
		return nil
	}

	// "Send the email" — i.e. log it. In production this would call
	// SendGrid / SES / etc. through an EmailSender interface.
	log.Printf(
		"[Notification] Sent email to %s for Order #%s. Amount: $%.2f",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100.0,
	)
	return nil
}
