package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type HTTPChecker struct {
	client *http.Client
	config CheckConfig
	status Status
	mu     sync.RWMutex
}

func NewHTTPChecker(config CheckConfig) *HTTPChecker {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	return &HTTPChecker{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

func (c *HTTPChecker) Check(ctx context.Context) error {
	var lastErr error

	for attempt := 0; attempt < c.config.RetryAttempts; attempt++ {
		if err := c.doCheck(ctx); err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.config.RetryDelay):
				continue
			}
		} else {
			c.updateStatus(true, nil)
			return nil
		}
	}

	c.updateStatus(false, lastErr)
	return lastErr
}

func (c *HTTPChecker) doCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPChecker) updateStatus(healthy bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = Status{
		Healthy:   healthy,
		LastCheck: time.Now(),
		Error:     err,
	}
}

func (c *HTTPChecker) GetStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}
