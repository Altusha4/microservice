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
)

// DeclareTopology — same shape as in payment-service.
// Both sides declare it; whoever boots first wins, the other
// is a no-op (declarations must be identical).
//
// Durable so messages survive a broker restart.
func DeclareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		ExchangePayments,
		"direct",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(
		QueueCompleted,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(
		QueueCompleted,
		RoutingKeyCompleted,
		ExchangePayments,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	return nil
}
