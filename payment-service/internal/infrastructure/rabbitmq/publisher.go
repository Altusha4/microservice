package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Altusha4/microservice/payment-service/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

// #######################################
// PUBLISHER
// #######################################

// Publisher implements usecase.EventPublisher on top of RabbitMQ
// with publisher confirms enabled — we wait for the broker ACK
// before returning success.
type Publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex // amqp.Channel is NOT goroutine-safe for Publish + confirms
}

// NewPublisher opens a channel, declares the topology, and turns
// publisher confirms on. Caller is responsible for calling Close.
func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := DeclareTopology(ch); err != nil {
		_ = ch.Close()
		return nil, err
	}

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &Publisher{conn: conn, ch: ch}, nil
}

// PublishPaymentCompleted serializes the event as JSON and publishes
// it persistently. Blocks until the broker confirms the message —
// this gives us at-least-once delivery on the producer side.
func (p *Publisher) PublishPaymentCompleted(ctx context.Context, event usecase.PaymentCompletedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	confirmation, err := p.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		ExchangePayments,
		RoutingKeyCompleted,
		true,  // mandatory — fail if no queue bound
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			MessageId:    event.EventID,
			Timestamp:    event.OccurredAt,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	// Wait synchronously for broker confirmation.
	ok, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait confirm: %w", err)
	}
	if !ok {
		return fmt.Errorf("broker NACKed message %s", event.EventID)
	}
	return nil
}

// Close shuts the channel and the underlying connection cleanly.
func (p *Publisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
