package health

import (
	"context"
	"log"
	"time"
)

// Monitor struct definition was missing
type Monitor struct {
	config  MonitorConfig
	checker HealthChecker
}

type MonitorConfig struct {
	Name          string
	CheckInterval time.Duration
	OnUnhealthy   func() error
	MaxRetries    int
	RetryDelay    time.Duration
}

func NewMonitor(config MonitorConfig, checker HealthChecker) *Monitor {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}
	return &Monitor{
		config:  config,
		checker: checker,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	failCount := 0
	lastFailure := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := m.checker.Check(ctx)
			if err != nil {
				failCount++
				if time.Since(lastFailure) > time.Minute {
					failCount = 1 // Reset fail count after a minute of no failures
					log.Printf("[%s] Fail count reset due to 1 minute of no failures", m.config.Name)
				}
				lastFailure = time.Now()

				if failCount > m.config.MaxRetries {
					log.Printf("[%s] Multiple failures detected (%d), triggering restart", m.config.Name, failCount)
					if m.config.OnUnhealthy != nil {
						if err := m.config.OnUnhealthy(); err != nil {
							log.Printf("[%s] Error handling unhealthy state: %v", m.config.Name, err)
						}
					}
					failCount = 0                   // Reset after restart
					time.Sleep(m.config.RetryDelay) // Wait before next check
				} else {
					log.Printf("[%s] Check failed (%d/%d): %v",
						m.config.Name, failCount, m.config.MaxRetries, err)
				}
			} else {
				failCount = 0 // Reset on successful check
				log.Printf("[%s] Health check passed", m.config.Name)
			}
		}
	}
}
