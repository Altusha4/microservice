package email

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// #######################################
// SIMULATED ADAPTER
// #######################################
//
// SimulatedSender mimics a real third-party email provider.
// It intentionally:
//   - sleeps for `latency` to simulate network/processing delay,
//   - returns a synthetic error with probability `failureRate`
//     so we can observe retry and exponential backoff in action.
//
// Configured via env:
//   SIMULATOR_LATENCY_MS   (default 200)
//   SIMULATOR_FAILURE_RATE (default 0.3)

type SimulatedSender struct {
	latency     time.Duration
	failureRate float64

	mu  sync.Mutex // rand.Rand is NOT safe for concurrent use
	rng *rand.Rand
}

// NewSimulatedSender returns a sender that fails approximately
// failureRate of the time after sleeping for `latency`.
func NewSimulatedSender(latency time.Duration, failureRate float64) *SimulatedSender {
	if failureRate < 0 {
		failureRate = 0
	}
	if failureRate > 1 {
		failureRate = 1
	}
	return &SimulatedSender{
		latency:     latency,
		failureRate: failureRate,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SimulatedSender) SendOrderConfirmation(ctx context.Context, msg Message) error {
	// Respect cancellation while we "talk to the provider".
	select {
	case <-time.After(s.latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	s.mu.Lock()
	roll := s.rng.Float64()
	s.mu.Unlock()

	if roll < s.failureRate {
		log.Printf("[email][simulated] FAIL order=%s to=%s (roll=%.2f < rate=%.2f)",
			msg.OrderID, msg.To, roll, s.failureRate)
		return fmt.Errorf("simulated provider error")
	}

	log.Printf("[email][simulated] OK order=%s to=%s amount=%s (latency=%s)",
		msg.OrderID, msg.To, msg.AmountUS, s.latency)
	return nil
}
