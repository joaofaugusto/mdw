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
					failCount = 1 // Redefinir a contagem de falhas apos um minuto sem falhas
					log.Printf("[%s] contagem de falhas redefinida devido a 1 minuto sem falhas", m.config.Name)
				}
				lastFailure = time.Now()

				if failCount > m.config.MaxRetries {
					log.Printf("[%s] várias falhas detectadas (%d), acionando reinicialização", m.config.Name, failCount)
					if m.config.OnUnhealthy != nil {
						if err := m.config.OnUnhealthy(); err != nil {
							log.Printf("[%s] erro ao lidar com estado unhealthy: %v", m.config.Name, err)
						}
					}
					failCount = 0                   // Redefinir apos reiniciar
					time.Sleep(m.config.RetryDelay) // Aguarda antes da proxima verificacao
				} else {
					log.Printf("[%s] Falha na verificação (%d/%d): %v",
						m.config.Name, failCount, m.config.MaxRetries, err)
				}
			} else {
				failCount = 0 // Redefinir na verificacao bem sucedida
				log.Printf("[%s] Teste de integridade checado", m.config.Name)
			}
		}
	}
}
