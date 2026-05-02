package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/Altusha4/microservice/notification-service/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

// #######################################
// CONSUMER
// #######################################

const MaxDeliveryAttempts = 3

// ErrPermanent — handler decided no retry will help. Message goes
// straight to DLQ on the first failure.
var ErrPermanent = errors.New("permanent processing error")

type Handler func(ctx context.Context, event domain.PaymentCompletedEvent) error

type Consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel

	// In-process retry counter keyed by MessageId.
	// Cannot rely on x-death header because Nack(requeue=true)
	// returns the message to the same queue without dead-lettering,
	// so x-death is never stamped. Counting in memory survives
	// perfectly within one consumer instance.
	mu       sync.Mutex
	attempts map[string]int
}

func NewConsumer(conn *amqp.Connection) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	if err := DeclareTopology(ch); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}
	return &Consumer{
		conn:     conn,
		ch:       ch,
		attempts: make(map[string]int),
	}, nil
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	deliveries, err := c.ch.ConsumeWithContext(
		ctx,
		QueueCompleted,
		"notification-service",
		false, // autoAck = false → MANUAL ACK
		false, false, false, nil,
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
				return fmt.Errorf("delivery channel closed")
			}
			c.handleDelivery(ctx, d, handler)
		}
	}
}

// handleDelivery — single-message lifecycle.
//
// Decision tree:
//  1. Bad JSON                     -> Nack(requeue=false) -> DLQ
//  2. ErrPermanent                 -> Nack(requeue=false) -> DLQ
//  3. Other error, attempts < N    -> Nack(requeue=true)
//  4. Other error, attempts >= N   -> Nack(requeue=false) -> DLQ + clear counter
//  5. Success                      -> Ack + clear counter
func (c *Consumer) handleDelivery(ctx context.Context, d amqp.Delivery, handler Handler) {
	var event domain.PaymentCompletedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("[notification][error] decode body: %v - dead-lettering", err)
		_ = d.Nack(false, false)
		return
	}

	key := d.MessageId
	if key == "" {
		key = event.EventID
	}

	err := handler(ctx, event)
	if err == nil {
		c.clearAttempts(key)
		if ackErr := d.Ack(false); ackErr != nil {
			log.Printf("[notification][error] ack failed for %s: %v", event.EventID, ackErr)
		}
		return
	}

	if errors.Is(err, ErrPermanent) {
		c.clearAttempts(key)
		log.Printf("[notification][permanent] event %s: %v - dead-lettering", event.EventID, err)
		_ = d.Nack(false, false)
		return
	}

	attempt := c.bumpAttempts(key)
	if attempt < MaxDeliveryAttempts {
		log.Printf("[notification][retry %d/%d] event %s: %v - requeueing",
			attempt, MaxDeliveryAttempts, event.EventID, err)
		_ = d.Nack(false, true)
		return
	}

	c.clearAttempts(key)
	log.Printf("[notification][give-up] event %s exhausted %d attempts: %v - dead-lettering",
		event.EventID, MaxDeliveryAttempts, err)
	_ = d.Nack(false, false)
}

func (c *Consumer) bumpAttempts(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts[key]++
	return c.attempts[key]
}

func (c *Consumer) clearAttempts(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.attempts, key)
}

func (c *Consumer) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
