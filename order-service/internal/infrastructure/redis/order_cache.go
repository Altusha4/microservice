package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Altusha4/microservice/order-service/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

// #######################################
// ORDER CACHE
// #######################################

// OrderCache implements usecase.OrderCache on top of Redis.
// Key layout:   order:<order_id>
// Value layout: JSON-encoded domain.Order
// TTL:          configured via constructor (env ORDER_CACHE_TTL_SECONDS)
type OrderCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewOrderCache(client *goredis.Client, ttl time.Duration) *OrderCache {
	return &OrderCache{client: client, ttl: ttl}
}

func key(id string) string { return "order:" + id }

// Get returns (nil, nil) on cache miss. Any other error means the cache
// is unhealthy — the caller should fall back to the database.
func (c *OrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	raw, err := c.client.Get(ctx, key(id)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var o domain.Order
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, fmt.Errorf("decode cached order: %w", err)
	}
	return &o, nil
}

// Set writes the order with the configured TTL.
func (c *OrderCache) Set(ctx context.Context, order *domain.Order) error {
	if order == nil {
		return nil
	}
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("encode order: %w", err)
	}
	if err := c.client.Set(ctx, key(order.ID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// Delete removes the cached entry. A missing key is not an error.
func (c *OrderCache) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, key(id)).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
