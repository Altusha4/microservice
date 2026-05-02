package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// #######################################
// TOPOLOGY CONSTANTS
// #######################################

const (
	ExchangePayments    = "payments"
	RoutingKeyCompleted = "payment.completed"
	QueueCompleted      = "payment.completed"

	ExchangeDLX    = "payments.dlx"
	RoutingKeyDead = "payment.completed.dead"
	QueueDLQ       = "payment.completed.dlq"
)

// DeclareTopology declares the main and dead-letter setup.
// Idempotent. Identical to notification-service version on purpose.
func DeclareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		ExchangePayments, "direct", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare main exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(
		ExchangeDLX, "direct", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}
	if _, err := ch.QueueDeclare(
		QueueDLQ, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := ch.QueueBind(
		QueueDLQ, RoutingKeyDead, ExchangeDLX, false, nil,
	); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    ExchangeDLX,
		"x-dead-letter-routing-key": RoutingKeyDead,
	}
	if _, err := ch.QueueDeclare(
		QueueCompleted, true, false, false, false, mainArgs,
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}
	if err := ch.QueueBind(
		QueueCompleted, RoutingKeyCompleted, ExchangePayments, false, nil,
	); err != nil {
		return fmt.Errorf("bind main queue: %w", err)
	}

	return nil
}
