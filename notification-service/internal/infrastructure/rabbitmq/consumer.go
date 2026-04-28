package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Altusha4/microservice/notification-service/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

// #######################################
// CONSUMER
// #######################################

// Handler is the business-level callback the consumer invokes for
// every successfully decoded event. The consumer manages ACK/NACK;
// the handler only decides "done" or "error".
type Handler func(ctx context.Context, event domain.PaymentCompletedEvent) error

type Consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewConsumer opens a channel, declares topology and sets QoS=1
// so RabbitMQ delivers one message at a time per consumer —
// keeps memory bounded and prevents one slow handler from
// hoarding hundreds of unacked messages.
func NewConsumer(conn *amqp.Connection) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	if err := DeclareTopology(ch); err != nil {
		_ = ch.Close()
		return nil, err
	}
	// prefetch=1 — deliver one message, wait for ACK before sending the next.
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}
	return &Consumer{conn: conn, ch: ch}, nil
}

// Run blocks until ctx is cancelled. For every delivery it:
//   - decodes JSON
//   - calls handler
//   - ACKs on success, NACKs (without requeue) on permanent error
//   - NACKs (with requeue) on transient error so the broker redelivers
//
// AutoAck is OFF — this is the key reliability requirement.
func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	deliveries, err := c.ch.ConsumeWithContext(
		ctx,
		QueueCompleted,         // queue
		"notification-service", // consumer tag
		false,                  // autoAck = false → MANUAL ACK
		false,                  // exclusive
		false,                  // no-local
		false,                  // no-wait
		nil,                    // args
	)
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}

	log.Println("[notification] consumer started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[notification] consumer stopping (context cancelled)")
			return nil

		case d, ok := <-deliveries:
			if !ok {
				// Channel closed by broker — return so main can decide
				// to reconnect or shut down.
				return fmt.Errorf("delivery channel closed")
			}

			c.handleDelivery(ctx, d, handler)
		}
	}
}

// handleDelivery processes a single message.
// All paths must end with either Ack or Nack — never leave a delivery
// unacked, otherwise RabbitMQ holds it forever.
func (c *Consumer) handleDelivery(ctx context.Context, d amqp.Delivery, handler Handler) {
	var event domain.PaymentCompletedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		// Broken JSON — never going to succeed on retry.
		// Drop the message (in the bonus step, this becomes "send to DLQ").
		log.Printf("[notification][error] decode body: %v — discarding", err)
		_ = d.Nack(false, false)
		return
	}

	if err := handler(ctx, event); err != nil {
		// Transient error (e.g. DB temporarily down) — requeue so we retry.
		// In the DLQ bonus step we'll cap retries; for the base milestone
		// we just keep retrying.
		log.Printf("[notification][error] handler failed for %s: %v — requeueing", event.EventID, err)
		_ = d.Nack(false, true)
		return
	}

	// Manual ACK — only after the handler reported success
	// (which means the email was logged AND idempotency row committed).
	if err := d.Ack(false); err != nil {
		log.Printf("[notification][error] ack failed for %s: %v", event.EventID, err)
	}
}

// Close shuts the channel and connection cleanly.
func (c *Consumer) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
