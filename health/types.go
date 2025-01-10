package health

import (
	"context"
	"time"
)

type CheckConfig struct {
	URL           string
	Timeout       time.Duration
	CheckInterval time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

type HealthChecker interface {
	Check(context.Context) error
	GetStatus() Status
}

type Status struct {
	Healthy   bool
	LastCheck time.Time
	Error     error
}
