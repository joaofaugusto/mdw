package health

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type ChaosChecker struct {
	wrapped             HealthChecker
	failureRate         int
	lastRestart         time.Time
	minUptime           time.Duration
	consecutiveFails    int
	maxConsecutiveFails int
	shutdownSimulated   bool // Track if shutdown was simulated
	shutdownRate        int  // Probability of simulating shutdown
}

func NewChaosChecker(checker HealthChecker, failureRate int, shutdownRate int) *ChaosChecker {
	return &ChaosChecker{
		wrapped:             checker,
		failureRate:         failureRate,
		minUptime:           1 * time.Minute, // Minimum time between chaos restarts
		lastRestart:         time.Now(),
		maxConsecutiveFails: 3,            // Maximum number of consecutive chaos failures
		shutdownRate:        shutdownRate, // Initialize shutdown rate
	}
}

func (c *ChaosChecker) Check(ctx context.Context) error {
	// First check if the actual service is healthy
	if err := c.wrapped.Check(ctx); err != nil {
		c.consecutiveFails = 0 // Reset chaos fails on real failure
		return fmt.Errorf("real health check failed: %w", err)
	}

	// Only apply chaos if minimum uptime has passed
	if time.Since(c.lastRestart) < c.minUptime {
		return nil
	}

	// Apply chaos with decreasing probability for consecutive failures
	adjustedRate := c.failureRate / (c.consecutiveFails + 1)
	if rand.Intn(100) < adjustedRate {
		c.consecutiveFails++
		if c.consecutiveFails > c.maxConsecutiveFails {
			c.consecutiveFails = 0 // Reset after max consecutive failures
			return nil
		}
		return fmt.Errorf("chaos mode: simulated failure (attempt %d)", c.consecutiveFails)
	}

	// Check for shutdown simulation
	if rand.Intn(100) < c.shutdownRate {
		c.shutdownSimulated = true
		return fmt.Errorf("chaos mode: simulated shutdown")
	}

	c.consecutiveFails = 0
	return nil
}

// ChaosChecker now has a method to trigger server shutdown
func (c *ChaosChecker) GetServerShutdown() bool {
	// Return true if shutdown needs to be simulated
	return rand.Intn(100) < c.failureRate
}

// Reset shutdown status after triggering a shutdown
func (c *ChaosChecker) ResetShutdownStatus() {
	// You can reset any internal state or flags here if needed
}

func (c *ChaosChecker) GetStatus() Status {
	return c.wrapped.GetStatus()
}
