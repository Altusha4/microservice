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

// DeclareTopology declares all exchanges, queues and bindings used by
// the payment notification flow. Idempotent — safe to call from both
// producer and consumer on every startup.
//
// All entities are durable so they survive broker restarts.
func DeclareTopology(ch *amqp.Channel) error {
	// Direct exchange — durable.
	if err := ch.ExchangeDeclare(
		ExchangePayments, // name
		"direct",         // kind
		true,             // durable
		false,            // auto-delete
		false,            // internal
		false,            // no-wait
		nil,              // args
	); err != nil {
		return fmt.Errorf("declare exchange %s: %w", ExchangePayments, err)
	}

	// Durable queue for payment.completed.
	if _, err := ch.QueueDeclare(
		QueueCompleted, // name
		true,           // durable
		false,          // auto-delete
		false,          // exclusive
		false,          // no-wait
		nil,            // args
	); err != nil {
		return fmt.Errorf("declare queue %s: %w", QueueCompleted, err)
	}

	// Bind queue to exchange.
	if err := ch.QueueBind(
		QueueCompleted,      // queue
		RoutingKeyCompleted, // routing key
		ExchangePayments,    // exchange
		false,               // no-wait
		nil,                 // args
	); err != nil {
		return fmt.Errorf("bind queue %s: %w", QueueCompleted, err)
	}

	return nil
}
