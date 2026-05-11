package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// #######################################
// IDEMPOTENCY CACHE
// #######################################
//
// Key layout:
//   ie:<event_id>      → "1"  (claim marker for an incoming event)
//   sent:<payment_id>  → "1"  (sent marker per payment)
//
// Both keys carry the same TTL — long enough to outlive any
// reasonable redelivery window, short enough to keep Redis tidy.

type IdempotencyCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewIdempotencyCache(client *goredis.Client, ttl time.Duration) *IdempotencyCache {
	return &IdempotencyCache{client: client, ttl: ttl}
}

func eventKey(id string) string   { return "ie:" + id }
func paymentKey(id string) string { return "sent:" + id }

// Claim uses SETNX semantics: SetArgs with Mode="NX" + TTL is
// a single atomic round-trip. Returns true if we are the first.
func (c *IdempotencyCache) Claim(ctx context.Context, eventID string) (bool, error) {
	ok, err := c.client.SetNX(ctx, eventKey(eventID), "1", c.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis SETNX: %w", err)
	}
	return ok, nil
}

func (c *IdempotencyCache) MarkSent(ctx context.Context, paymentID string) error {
	if err := c.client.Set(ctx, paymentKey(paymentID), "1", c.ttl).Err(); err != nil {
		return fmt.Errorf("redis SET sent: %w", err)
	}
	return nil
}

func (c *IdempotencyCache) IsSent(ctx context.Context, paymentID string) (bool, error) {
	n, err := c.client.Exists(ctx, paymentKey(paymentID)).Result()
	if err != nil {
		return false, fmt.Errorf("redis EXISTS sent: %w", err)
	}
	return n > 0, nil
}

