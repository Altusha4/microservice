package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// #######################################
// RATE LIMITER (Assignment 4 bonus)
// #######################################
//
// Fixed-window rate limiter backed by Redis.
//
// Algorithm:
//   key   = "rate:<client_ip>"
//   value = number of requests in the current 60-second window
//
// On each request:
//   n = INCR key
//   if n == 1: EXPIRE key 60
//   if n > limit: 429
//
// Why Redis instead of in-process counters:
//   - shared across instances
//   - INCR is atomic
//   - EXPIRE handles cleanup automatically

type RateLimiter struct {
	client *goredis.Client
	limit  int
	window time.Duration
}

func NewRateLimiter(client *goredis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{client: client, limit: limit, window: window}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		key := "rate:" + ip

		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		count, err := rl.client.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("[ratelimit] INCR failed for %s: %v - allowing request", ip, err)
			c.Next()
			return
		}

		if count == 1 {
			if err := rl.client.Expire(ctx, key, rl.window).Err(); err != nil {
				log.Printf("[ratelimit] EXPIRE failed for %s: %v", ip, err)
			}
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(maxInt(0, rl.limit-int(count))))

		if int(count) > rl.limit {
			ttl, _ := rl.client.TTL(ctx, key).Result()
			c.Header("Retry-After", strconv.Itoa(int(ttl.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"limit":       rl.limit,
				"window_sec":  int(rl.window.Seconds()),
				"retry_after": fmt.Sprintf("%ds", int(ttl.Seconds())),
			})
			return
		}

		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
